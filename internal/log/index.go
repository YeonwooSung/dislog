package log

import (
	"io"
	"os"

	"github.com/tysonmote/gommap"
)

var (
	offWidth uint64 = 4
	posWidth uint64 = 8
	entWidth uint64 = offWidth + posWidth
)

type index struct {
	file *os.File
	mmap gommap.MMap // use gommap to mmap the index file
	size uint64
}

func newIndex(f *os.File, c Config) (*index, error) {
	idx := &index{
		file: f,
	}

	// get file stat of the given file
	fi, err := os.Stat(f.Name())
	if err != nil {
		return nil, err
	}
	// get the size of the index file
	idx.size = uint64(fi.Size())

	// use os.Trucate to change the size of the index file
	if err = os.Truncate(
		f.Name(), int64(c.Segment.MaxIndexBytes),
	); err != nil {
		return nil, err
	}

	// initialise the memory mapped file
	if idx.mmap, err = gommap.Map(
		idx.file.Fd(),
		gommap.PROT_READ|gommap.PROT_WRITE,
		gommap.MAP_SHARED,
	); err != nil {
		return nil, err
	}
	return idx, nil
}

func (i *index) Close() error {
	// Sync the memory mapped file by flushing the data to the disk
	if err := i.mmap.Sync(gommap.MS_SYNC); err != nil {
		return err
	}
	// Commit the changes of the index file to the disk
	if err := i.file.Sync(); err != nil {
		return err
	}

	// Truncate the index file to the size of the index
	if err := i.file.Truncate(int64(i.size)); err != nil {
		return err
	}
	return i.file.Close()
}

func (i *index) Read(in int64) (out uint32, pos uint64, err error) {
	// if the index file is empty, return EOF
	if i.size == 0 {
		return 0, 0, io.EOF
	}

	// if the given offset is -1, return the last entry
	if in == -1 {
		out = uint32((i.size / entWidth) - 1)
	} else {
		out = uint32(in)
	}

	// get the position of the entry
	pos = uint64(out) * entWidth

	var pos_start uint64 = pos + offWidth
	var pos_end uint64 = pos + entWidth

	// if the position is out of the index file, return EOF
	if i.size < pos_end {
		return 0, 0, io.EOF
	}

	out = enc.Uint32(i.mmap[pos:pos_start])
	pos = enc.Uint64(i.mmap[pos_start:pos_end])
	return out, pos, nil
}

func (i *index) Write(off uint32, pos uint64) error {
	var idx_size_offset uint64 = i.size + offWidth
	var idx_size_end uint64 = i.size + entWidth

	// if the index file is full, return EOF
	if uint64(len(i.mmap)) < idx_size_end {
		return io.EOF
	}

	// write the offset and position to the index file
	enc.PutUint32(i.mmap[i.size:idx_size_offset], off)
	enc.PutUint64(i.mmap[idx_size_offset:idx_size_end], pos)
	i.size += uint64(entWidth)
	return nil
}

func (i *index) Name() string {
	return i.file.Name()
}

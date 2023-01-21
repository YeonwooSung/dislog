package log

import (
	"bufio"
	"encoding/binary"
	"os"
	"sync"
)

var (
	enc = binary.BigEndian
)

const (
	lenWidth = 8
)

type store struct {
	*os.File
	mu   sync.Mutex
	buf  *bufio.Writer
	size uint64
}

func newStore(f *os.File) (*store, error) {
	fi, err := f.Stat()
	if err != nil {
		return nil, err
	}
	size := uint64(fi.Size())
	return &store{
		File: f,
		size: size,
		buf:  bufio.NewWriter(f),
	}, nil
}

//---------------------------------------------
// "store" struct methods

func (s *store) Append(p []byte) (n uint64, pos uint64, err error) {
	// Lock the mutex to ensure only one goroutine can write to the file at a time
	s.mu.Lock()
	// Unlock the mutex when the function returns
	defer s.mu.Unlock()

	// write the length of the record to the store buffer, so that later we can know how many bytes to read from the store
	pos = s.size
	if err := binary.Write(s.buf, enc, uint64(len(p))); err != nil {
		return 0, 0, err
	}

	// write the record to the buffer
	w, err := s.buf.Write(p)
	if err != nil {
		return 0, 0, err
	}

	// update the size of the store
	w += lenWidth
	s.size += uint64(w)
	return uint64(w), pos, nil
}

func (s *store) Read(pos uint64) ([]byte, error) {
	// Lock the mutex to ensure only one goroutine can write to the file at a time
	s.mu.Lock()
	// Unlock the mutex when the function returns
	defer s.mu.Unlock()

	// flush the buffer, so that we can read all pushed data
	if err := s.buf.Flush(); err != nil {
		return nil, err
	}

	// read the length of the record from the store
	size := make([]byte, lenWidth)
	if _, err := s.File.ReadAt(size, int64(pos)); err != nil {
		return nil, err
	}

	// read the record from the store
	b := make([]byte, enc.Uint64(size))
	if _, err := s.File.ReadAt(b, int64(pos+lenWidth)); err != nil {
		return nil, err
	}
	return b, nil
}

func (s *store) ReadAt(p []byte, off int64) (int, error) {
	// Lock the mutex to ensure only one goroutine can write to the file at a time
	s.mu.Lock()
	// Unlock the mutex when the function returns
	defer s.mu.Unlock()

	// flush the buffer, so that we can read all pushed data
	if err := s.buf.Flush(); err != nil {
		return 0, err
	}
	// read "p" bytes from the store at offset "off", and return the number of bytes read
	return s.File.ReadAt(p, off)
}

func (s *store) Close() error {
	// Lock the mutex to ensure only one goroutine can write to the file at a time
	s.mu.Lock()
	// Unlock the mutex when the function returns
	defer s.mu.Unlock()

	// flush the buffer, so that we can read all pushed data
	err := s.buf.Flush()
	if err != nil {
		return err
	}
	// close the file stream
	return s.File.Close()
}
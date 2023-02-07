package log

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"
	"sync"

	api "github.com/YeonwooSung/dislog/api/v1"
)

type Log struct {
	mu sync.RWMutex

	Dir    string
	Config Config

	activeSegment *segment
	segments      []*segment
}

func NewLog(dir string, c Config) (*Log, error) {
	// if the user did not specify a max segment size, we default value which is 1KB
	if c.Segment.MaxStoreBytes == 0 {
		c.Segment.MaxStoreBytes = 1024
	}
	// if the user did not specify a max index size, we default value which is 1KB
	if c.Segment.MaxIndexBytes == 0 {
		c.Segment.MaxIndexBytes = 1024
	}

	// create a new log instance
	l := &Log{
		Dir:    dir,
		Config: c,
	}

	return l, l.setup()
}

func (l *Log) setup() error {
	// open the log directory, and get a list of all the files in the directory
	files, err := ioutil.ReadDir(l.Dir)
	if err != nil {
		return err
	}

	// if there are existing segments, start with those segments
	var baseOffsets []uint64
	for _, file := range files {
		// trim the file extension to get the base offset
		offStr := strings.TrimSuffix(
			file.Name(),
			path.Ext(file.Name()),
		)
		// parse the base offset
		off, _ := strconv.ParseUint(offStr, 10, 0)
		// append the base offset to the baseOffsets slice
		baseOffsets = append(baseOffsets, off)
	}
	// sort the base offsets
	sort.Slice(baseOffsets, func(i, j int) bool {
		return baseOffsets[i] < baseOffsets[j]
	})

	// create a new segment for each base offset
	for i := 0; i < len(baseOffsets); i++ {
		if err = l.newSegment(baseOffsets[i]); err != nil {
			return err
		}

		// baseOffset contains dup for index and store so we skip the dup
		i += 1
	}

	// if there are no existing segments, create a new segment
	if l.segments == nil {
		// create new segment with initial offset
		if err = l.newSegment(l.Config.Segment.InitialOffset); err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) Append(record *api.Record) (uint64, error) {
	// lock mutex before appending
	l.mu.Lock()
	// unlock mutex when done appending
	defer l.mu.Unlock()

	// append the record to the active segment
	off, err := l.activeSegment.Append(record)
	if err != nil {
		return 0, err
	}
	// if the active segment is full, create a new segment
	if l.activeSegment.IsMaxed() {
		err = l.newSegment(off + 1)
	}
	return off, err
}

func (l *Log) Read(off uint64) (*api.Record, error) {
	// lock read lock mutex before reading
	l.mu.RLock()
	// unlock read lock mutex when done reading
	defer l.mu.RUnlock()

	// for each segment in log, check if the given offset is within the segment
	var s *segment
	for _, segment := range l.segments {
		if segment.baseOffset <= off && off < segment.nextOffset {
			s = segment
			break
		}
	}

	if s == nil || s.nextOffset <= off {
		return nil, fmt.Errorf("offset out of range: %d", off)
	}
	return s.Read(off)
}

func (l *Log) newSegment(off uint64) error {
	// create new segment
	s, err := newSegment(l.Dir, off, l.Config)
	if err != nil {
		return err
	}
	// append the created segment to the segment list
	l.segments = append(l.segments, s)
	// make the created segment as a active segment
	l.activeSegment = s
	return nil
}

func (l *Log) Close() error {
	// lock mutex before closing
	l.mu.Lock()
	// unlock mutex when done closing
	defer l.mu.Unlock()

	// call segment.Close() to close all segments in this log
	for _, segment := range l.segments {
		if err := segment.Close(); err != nil {
			return err
		}
	}
	return nil
}

func (l *Log) Remove() error {
	// close before removing
	if err := l.Close(); err != nil {
		return err
	}
	// remove all files in the log directory
	return os.RemoveAll(l.Dir)
}

func (l *Log) Reset() error {
	// remove before resetting
	if err := l.Remove(); err != nil {
		return err
	}
	// setup after resetting
	return l.setup()
}

func (l *Log) LowestOffset() (uint64, error) {
	// lock read lock mutex before reading
	l.mu.RLock()
	// unlock read lock mutex when done reading
	defer l.mu.RUnlock()
	// return the base offset of the first segment
	return l.segments[0].baseOffset, nil
}

func (l *Log) HighestOffset() (uint64, error) {
	// lock read lock mutex before reading
	l.mu.RLock()
	// unlock read lock mutex when done reading
	defer l.mu.RUnlock()

	// return the next offset of the last segment
	off := l.segments[len(l.segments)-1].nextOffset
	if off == 0 {
		return 0, nil
	}
	return off - 1, nil
}

func (l *Log) Truncate(lowest uint64) error {
	// lock mutex before truncating
	l.mu.Lock()
	// unlock mutex when done truncating
	defer l.mu.Unlock()

	// For each segment, check if the next offset is less than or equal to the lowest offset
	// If so, remove the segment
	// By running this in a loop, we could let a new segment list only contain segments that are not removed
	var segments []*segment
	for _, s := range l.segments {
		if s.nextOffset <= lowest+1 {
			if err := s.Remove(); err != nil {
				return err
			}
			continue
		}
		segments = append(segments, s)
	}
	l.segments = segments
	return nil
}

func (l *Log) Reader() io.Reader {
	// lock read lock mutex before reading
	l.mu.RLock()
	// unlock read lock mutex when done reading
	defer l.mu.RUnlock()

	// create a reader for each segment
	readers := make([]io.Reader, len(l.segments))
	for i, segment := range l.segments {
		readers[i] = &originReader{segment.store, 0}
	}
	return io.MultiReader(readers...)
}

type originReader struct {
	*store
	off int64
}

func (o *originReader) Read(p []byte) (int, error) {
	n, err := o.ReadAt(p, o.off)
	o.off += int64(n)
	return n, err
}

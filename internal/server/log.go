package server

import (
	"errors"
	"sync"
)

type Log struct {
	mu      sync.Mutex
	records []Record
}

type Record struct {
	Value  []byte `json:"value"`  // automatically match JSON filed value with Value when marshalling and unmarshalling
	Offset uint64 `json:"offset"` // automatically match JSON filed offset with Offset when marshalling and unmarshalling
}

func NewLog() *Log {
	return &Log{}
}

func (c *Log) Append(record Record) (uint64, error) {
	c.mu.Lock()

	// "defer" is something like "finally" in Java -> it will be executed right before the function returns the result
	defer c.mu.Unlock()

	c.records = append(c.records, record)
	return uint64(len(c.records)), nil
}

func (c *Log) Read(offset uint64) (Record, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if offset >= uint64(len(c.records)) {
		return Record{}, ErrOffsetNotFound
	}

	return c.records[offset], nil
}

var ErrOffsetNotFound = errors.New("offset not found")

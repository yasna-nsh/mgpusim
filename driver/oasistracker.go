package driver

import (
	"fmt"
	"sync"
)

const maxObjects = 16

type allocRecord struct {
	baseAddr Ptr
	endAddr  Ptr
	objID    uint8
}

type ObjectTracker struct {
	mu      sync.Mutex
	records []allocRecord
	nextID  uint8
}

func (t *ObjectTracker) Track(baseAddr Ptr, size uint64) (uint8, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if int(t.nextID) >= maxObjects {
		return 0, fmt.Errorf("objecttracker: number of objects exceeds %d", maxObjects)
	}

	id := t.nextID
	t.nextID++

	t.records = append(t.records, allocRecord{
		baseAddr: baseAddr,
		endAddr:  baseAddr + Ptr(size),
		objID:    id,
	})

	return id, nil
}

func (t *ObjectTracker) Identify(addr Ptr) (uint8, bool) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for _, r := range t.records {
		if r.baseAddr <= addr && addr < r.endAddr {
			return r.objID, true
		}
	}

	return maxObjects, false
}

func (t *ObjectTracker) Free(baseAddr Ptr) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, r := range t.records {
		if r.baseAddr == baseAddr {
			t.records = append(t.records[:i], t.records[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("objecttracker: no record for base addr 0x%x", baseAddr)
}

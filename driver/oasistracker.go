package driver

import (
	"fmt"
	"log"
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
	freeIDs []uint8
}

func (t *ObjectTracker) Track(baseAddr Ptr, size uint64) (uint8, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var id uint8
	if len(t.freeIDs) > 0 {
		id = t.freeIDs[len(t.freeIDs)-1]
		t.freeIDs = t.freeIDs[:len(t.freeIDs)-1]
	} else if int(t.nextID) < maxObjects {
		id = t.nextID
		t.nextID++
	} else {
		return 0, fmt.Errorf("objecttracker: number of objects exceeds %d", maxObjects)
	}

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
			t.freeIDs = append(t.freeIDs, r.objID)
			t.records = append(t.records[:i], t.records[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("objecttracker: no record for base addr 0x%x", baseAddr)
}

func (t *ObjectTracker) GetBaseSize(objID uint8) (uint64, uint64) {
	for _, r := range t.records {
		if r.objID == objID {
			return uint64(r.baseAddr), uint64(r.endAddr - r.baseAddr)
		}
	}
	return 0, 0
}

func (t *ObjectTracker) PrintPagePolicyPercentage(objTable *OTable) {
	var percentages [3]uint64
	total := uint64(0)
	for _, e := range t.records {
		pcount := t.calcPageCount(e)
		policy := objTable.find(e.objID).Policy
		percentages[policy] += pcount
		total += pcount
	}
	log.Printf("[policy percentage] total=%v, %v", total, percentages)
}

func (t *ObjectTracker) GetTotalPCount() uint64 {
	//
	total := uint64(0)
	for _, e := range t.records {
		total += t.calcPageCount(e)
	}
	return total
}

func (t *ObjectTracker) calcPageCount(rec allocRecord) uint64 {
	size := uint64(rec.endAddr - rec.baseAddr)
	psize := uint64(4096)
	pcount := size / psize
	if size%psize != 0 {
		pcount++
	}
	return pcount
}

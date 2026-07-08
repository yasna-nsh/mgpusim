package driver

import (
	"log"
	"sync"

	"github.com/sarchlab/akita/v3/mem/vm"
)

type OTableEntry struct {
	Policy  vm.MigrationPolicy
	PFCount uint8 // 3-bit page fault counter
	mu      sync.Mutex
}

const ResetThreshold = 8

// add LRU and fixed size OTable if necessary
const OTableSize = 16

type OTable struct {
	mu      sync.Mutex
	entries map[uint8]*OTableEntry
}

func NewOTable() *OTable {
	return &OTable{
		entries: make(map[uint8]*OTableEntry),
	}
}

func (t *OTable) find(objID uint8) *OTableEntry {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.entries[objID]
}

func (t *OTable) Insert(objID uint8) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.entries[objID] = &OTableEntry{Policy: vm.PolicyOnTouch}
}

func (t *OTable) Remove(objID uint8) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.entries, objID)
}

func (t *OTable) Lookup(objID uint8) *OTableEntry {
	return t.find(objID)
}

func (t *OTable) Update(objID uint8, policy vm.MigrationPolicy) {
	ent := t.find(objID)
	if ent == nil {
		return
	}
	ent.mu.Lock()
	defer ent.mu.Unlock()
	ent.Policy = policy
}

func (t *OTable) RecordPageFault(objID uint8, write bool) {
	ent := t.find(objID)
	if ent == nil {
		return
	}
	ent.mu.Lock()
	defer ent.mu.Unlock()

	// update
	if ent.PFCount == 0 {
		log.Printf("[update policy] objID=%d write=%v\n", objID, write)
		if write {
			ent.Policy = vm.PolicyAccessCounter
		} else {
			ent.Policy = vm.PolicyDuplication
		}
	}
	ent.PFCount++
	log.Printf("[page fault] objID=%d, PF counter=%d\n", objID, ent.PFCount)
	if ent.PFCount == 8 {
		ent.PFCount = 0
	}
}

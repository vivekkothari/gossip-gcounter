package crdt

import (
	"strings"
	"sync"
)

type CRDTChar struct {
	ID        string // "timestamp-nodeID"
	Value     rune
	IsDeleted bool
}

type CRDTText struct {
	sequence []CRDTChar
	mu       sync.RWMutex
}

func (c *CRDTText) Insert(afterID, newID string, val rune) {
	c.mu.Lock()
	defer c.mu.Unlock()

	idx := c.findIndex(afterID)
	newChar := CRDTChar{ID: newID, Value: val}
	if idx == -1 {
		c.sequence = append([]CRDTChar{newChar}, c.sequence...)
	} else {
		c.sequence = append(c.sequence[:idx+1],
			append([]CRDTChar{newChar}, c.sequence[idx+1:]...)...)
	}
}

func (c *CRDTText) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	for i := range c.sequence {
		if c.sequence[i].ID == id {
			c.sequence[i].IsDeleted = true
			break
		}
	}
}

func (c *CRDTText) findIndex(id string) int {
	for i, ch := range c.sequence {
		if ch.ID == id {
			return i
		}
	}
	return -1
}

// GetVisibleText returns the concatenated string of non-deleted characters
func (c *CRDTText) GetVisibleText() string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	var sb strings.Builder
	for _, ch := range c.sequence {
		if !ch.IsDeleted {
			sb.WriteRune(ch.Value)
		}
	}
	return sb.String()
}

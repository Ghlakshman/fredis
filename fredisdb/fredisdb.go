package fredisdb

import (
	"sync"
	"time"
)

type Value struct {
	mu         sync.RWMutex
	Value      any
	Expiry     *time.Time
	LastAccess time.Time
	IsVolatile bool
}

type FredisStore struct {
	mu         sync.RWMutex
	data       map[string]*Value
	maxEntries uint64
	policy     string
}

func NewFredisStore(policy string, maxEntries uint64) *FredisStore {
	return &FredisStore{
		data:       make(map[string]*Value),
		policy:     policy,
		maxEntries: maxEntries,
	}
}

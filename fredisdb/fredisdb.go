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
	mu             sync.RWMutex
	data           map[string]*Value
	maxEntries     uint64
	EvictionPolicy EvictionPolicy
}

type EvictionPolicy string

const (
	PolicyNone          EvictionPolicy = "noeviction"
	PolicyAllKeysRandom EvictionPolicy = "allkeys-random"
	PolicyVolatileLRU   EvictionPolicy = "volatile-lru"
)

func IsValidEvictionPolicy(policy string) bool {
	switch EvictionPolicy(policy) {
	case PolicyNone, PolicyAllKeysRandom, PolicyVolatileLRU:
		return true
	default:
		return false
	}
}

func NewFredisStore(policy EvictionPolicy, maxEntries uint64) *FredisStore {
	return &FredisStore{
		data:           make(map[string]*Value),
		EvictionPolicy: policy,
		maxEntries:     maxEntries,
	}
}

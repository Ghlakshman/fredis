package fredisdb

import (
	"errors"
	"time"
)

type FredisCmds struct {
	fredisDb *FredisStore
}

func NewFredisCmds(store *FredisStore) *FredisCmds {
	return &FredisCmds{
		fredisDb: store,
	}
}

func (fc *FredisCmds) SetValue(key string, val *Value) {
	now := time.Now()
	fc.fredisDb.mu.RLock()
	existing, exists := fc.fredisDb.data[key]

	if exists {
		existing.mu.Lock()
		existing.Value = val.Value
		existing.LastAccess = now
		existing.Expiry = val.Expiry
		existing.mu.Unlock()
		fc.fredisDb.mu.RUnlock()
		return
	}
	fc.fredisDb.mu.RUnlock()
	fc.fredisDb.mu.Lock()
	defer fc.fredisDb.mu.Unlock()

	//double check if someone in the meanwhile we were waiting for the lock has updated the store with the key!!
	still_existing, still_exists := fc.fredisDb.data[key]

	if still_exists {
		still_existing.mu.Lock()
		defer still_existing.mu.Unlock()

		still_existing.Value = val.Value
		still_existing.LastAccess = now
		still_existing.Expiry = val.Expiry
		return
	} else {
		//Insert new key
		fc.fredisDb.data[key] = val
	}
}

func (fc *FredisCmds) GetValue(key string) (*Value, error) {
	now := time.Now()

	fc.fredisDb.mu.RLock()
	existing, exists := fc.fredisDb.data[key]

	if !exists {
		return nil, errors.New("key not found in FredisDB")
	}

	existing.mu.Lock()

	if existing.Expiry != nil && now.After(*existing.Expiry) {
		existing.mu.Unlock()
		fc.fredisDb.mu.RUnlock()
		fc.DelValue(key)
		return nil, errors.New("key has expired")
	}

	existing.LastAccess = now

	existing.mu.Unlock()
	fc.fredisDb.mu.RUnlock()

	return existing, nil
}

func (fc *FredisCmds) DelValue(key string) bool {
	fc.fredisDb.mu.Lock()
	existing, exists := fc.fredisDb.data[key]
	if !exists {
		fc.fredisDb.mu.Unlock()
		return false
	}
	existing.mu.Lock()
	val, ok := fc.fredisDb.data[key]
	if !ok || val != existing {
		existing.mu.Unlock()
		fc.fredisDb.mu.Unlock()
		return false
	}

	delete(fc.fredisDb.data, key)

	existing.mu.Unlock()
	fc.fredisDb.mu.Unlock()
	return true
}

func (fc *FredisCmds) SetExpiry(key string, seconds int64) (int8, error) {
	now := time.Now()
	fc.fredisDb.mu.RLock()
	existing, exists := fc.fredisDb.data[key]

	if !exists {
		fc.fredisDb.mu.RUnlock()
		return -1, errors.New("key does not exist")
	}

	existing.mu.Lock()

	if existing.Expiry != nil && now.After(*existing.Expiry) {
		existing.mu.Unlock()
		fc.fredisDb.mu.RUnlock()
		fc.DelValue(key)
		return -2, errors.New("key has expired")
	}

	expiry := time.Now().Add(time.Duration(seconds) * time.Second)
	existing.Expiry = &expiry

	existing.mu.Unlock()
	fc.fredisDb.mu.RUnlock()

	return 1, nil
}

func (fc *FredisCmds) TTL(key string) int {
	now := time.Now()

	fc.fredisDb.mu.RLock()
	defer fc.fredisDb.mu.RUnlock()
	existing, exists := fc.fredisDb.data[key]

	if !exists {
		return -2
	}
	existing.mu.RLock()

	if existing.Expiry == nil {
		existing.mu.RUnlock()
		return -1
	}

	if now.After(*existing.Expiry) {
		existing.mu.RUnlock()
		fc.DelValue(key)
		return -2
	}

	existing.mu.RUnlock()
	return int(existing.Expiry.Sub(now).Seconds())
}

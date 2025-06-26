package fredisdb

import (
	"errors"
	"log"
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

		log.Printf("[SET] Updated existing key: %s", key)
		return
	}
	fc.fredisDb.mu.RUnlock()

	fc.fredisDb.mu.Lock()
	defer fc.fredisDb.mu.Unlock()

	still_existing, still_exists := fc.fredisDb.data[key]
	if still_exists {
		still_existing.mu.Lock()
		defer still_existing.mu.Unlock()

		still_existing.Value = val.Value
		still_existing.LastAccess = now
		still_existing.Expiry = val.Expiry

		log.Printf("[SET] Updated (after double-check) existing key: %s", key)
		return
	}

	fc.fredisDb.data[key] = val
	log.Printf("[SET] Inserted new key: %s", key)
}

func (fc *FredisCmds) GetValue(key string) (*Value, error) {
	log.Printf("Inside GET")
	now := time.Now()

	fc.fredisDb.mu.RLock()
	existing, exists := fc.fredisDb.data[key]
	if !exists {
		fc.fredisDb.mu.RUnlock()
		log.Printf("[GET] Key not found: %s", key)
		return nil, errors.New("key not found in FredisDB")
	}

	existing.mu.Lock()
	defer existing.mu.Unlock()

	if existing.Expiry != nil && now.After(*existing.Expiry) {
		fc.fredisDb.mu.RUnlock()
		log.Printf("Deleting KEY")
		go fc.DelValue(key)
		log.Printf("[GET] Key expired: %s", key)
		return nil, errors.New("key has expired")
	}

	existing.LastAccess = now
	fc.fredisDb.mu.RUnlock()
	log.Printf("[GET] Fetched key: %s", key)

	return existing, nil
}

func (fc *FredisCmds) DelValue(key string) bool {
	log.Printf("Inside Delete Acquiring lock")
	fc.fredisDb.mu.Lock()
	log.Printf("Inside Delete Acquired MS lock")
	existing, exists := fc.fredisDb.data[key]
	if !exists {
		fc.fredisDb.mu.Unlock()
		log.Printf("[DEL] Key not found: %s", key)
		return false
	}
	log.Printf("Inside Delete Acquiring  Entry lock")
	existing.mu.Lock()
	log.Printf("Inside Delete Acquired  Entry lock")
	val, ok := fc.fredisDb.data[key]
	if !ok || val != existing {
		existing.mu.Unlock()
		fc.fredisDb.mu.Unlock()
		log.Printf("[DEL] Conflict deleting key (may have been overwritten): %s", key)
		return false
	}

	go delete(fc.fredisDb.data, key)
	existing.mu.Unlock()
	fc.fredisDb.mu.Unlock()

	log.Printf("[DEL] Deleted key: %s", key)
	return true
}

func (fc *FredisCmds) SetExpiry(key string, seconds int64) (int8, error) {
	now := time.Now()

	fc.fredisDb.mu.RLock()
	existing, exists := fc.fredisDb.data[key]
	if !exists {
		fc.fredisDb.mu.RUnlock()
		log.Printf("[EXPIRE] Key does not exist: %s", key)
		return -1, errors.New("key does not exist")
	}

	existing.mu.Lock()
	defer existing.mu.Unlock()

	if existing.Expiry != nil && now.After(*existing.Expiry) {
		fc.fredisDb.mu.RUnlock()
		go fc.DelValue(key)
		log.Printf("[EXPIRE] Key already expired: %s", key)
		return -2, errors.New("key has expired")
	}

	expiry := now.Add(time.Duration(seconds) * time.Second)
	existing.Expiry = &expiry
	fc.fredisDb.mu.RUnlock()

	log.Printf("[EXPIRE] Expiry set for key %s to %v", key, expiry)
	return 1, nil
}

func (fc *FredisCmds) TTL(key string) int {
	now := time.Now()

	fc.fredisDb.mu.RLock()
	defer fc.fredisDb.mu.RUnlock()
	existing, exists := fc.fredisDb.data[key]

	if !exists {
		log.Printf("[TTL] Key not found: %s", key)
		return -2
	}

	existing.mu.RLock()
	defer existing.mu.RUnlock()

	if existing.Expiry == nil {
		log.Printf("[TTL] Key %s has no expiry", key)
		return -1
	}

	if now.After(*existing.Expiry) {
		go fc.DelValue(key) // async delete to avoid deadlock
		log.Printf("[TTL] Key expired on TTL: %s", key)
		return -2
	}

	ttl := int(existing.Expiry.Sub(now).Seconds())
	log.Printf("[TTL] TTL for key %s: %ds", key, ttl)
	return ttl
}

package fredisdb

import (
	"errors"
	"fmt"
	"log"
	"math/rand"
	"strconv"
	"time"
)

type FredisCmds struct {
	FredisDb    *FredisStore
	AOF         *AOF
	IsReplaying bool
}

func NewFredisCmds(store *FredisStore, aof *AOF) *FredisCmds {
	return &FredisCmds{
		FredisDb: store,
		AOF:      aof,
	}
}

func (fc *FredisCmds) SetValue(key string, val *Value) {

	if !fc.IsReplaying {
		fc.AOF.LogCommand(fmt.Sprintf("*3\r\n$3\r\nSET\r\n$%d\r\n%s\r\n$%d\r\n%s\r\n",
			len(key),
			key, len(val.Value.(string)),
			val.Value.(string)))
	}

	now := time.Now()
	fc.FredisDb.mu.RLock()
	existing, exists := fc.FredisDb.data[key]

	if exists {
		existing.mu.Lock()
		existing.Value = val.Value
		existing.LastAccess = now
		existing.Expiry = val.Expiry
		existing.mu.Unlock()
		fc.FredisDb.mu.RUnlock()

		log.Printf("[SET] Updated existing key: %s", key)
		return
	}
	fc.FredisDb.mu.RUnlock()

	fc.FredisDb.mu.Lock()
	defer fc.FredisDb.mu.Unlock()

	still_existing, still_exists := fc.FredisDb.data[key]
	if still_exists {
		still_existing.mu.Lock()
		defer still_existing.mu.Unlock()

		still_existing.Value = val.Value
		still_existing.LastAccess = now
		still_existing.Expiry = val.Expiry

		log.Printf("[SET] Updated (after double-check) existing key: %s", key)
		return
	}

	if uint64(len(fc.FredisDb.data)) >= fc.FredisDb.maxEntries {
		log.Printf("[SET] Reached max key limit (%d), triggering eviction", fc.FredisDb.maxEntries)
		fc.evict()
	}

	fc.FredisDb.data[key] = val
	log.Printf("[SET] Inserted new key: %s", key)
}

func (fc *FredisCmds) GetValue(key string) (*Value, error) {
	now := time.Now()

	fc.FredisDb.mu.RLock()
	existing, exists := fc.FredisDb.data[key]
	if !exists {
		fc.FredisDb.mu.RUnlock()
		log.Printf("[GET] Key not found: %s", key)
		return nil, errors.New("key not found in FredisDB")
	}

	existing.mu.Lock()
	defer existing.mu.Unlock()

	if existing.Expiry != nil && now.After(*existing.Expiry) {
		fc.FredisDb.mu.RUnlock()
		log.Printf("Deleting KEY")
		go fc.DelValue(key)
		log.Printf("[GET] Key expired: %s", key)
		return nil, errors.New("key has expired")
	}

	existing.LastAccess = now
	fc.FredisDb.mu.RUnlock()
	log.Printf("[GET] Fetched key: %s", key)

	return existing, nil
}

func (fc *FredisCmds) DelValue(key string) bool {

	if !fc.IsReplaying {
		fc.AOF.LogCommand(fmt.Sprintf("*2\r\n$3\r\nDEL\r\n$%d\r\n%s\r\n",
			len(key), key))
	}

	fc.FredisDb.mu.Lock()
	existing, exists := fc.FredisDb.data[key]
	if !exists {
		fc.FredisDb.mu.Unlock()
		log.Printf("[DEL] Key not found: %s", key)
		return false
	}

	existing.mu.Lock()
	val, ok := fc.FredisDb.data[key]
	if !ok || val != existing {
		existing.mu.Unlock()
		fc.FredisDb.mu.Unlock()
		log.Printf("[DEL] Conflict deleting key (may have been overwritten): %s", key)
		return false
	}

	go delete(fc.FredisDb.data, key)
	existing.mu.Unlock()
	fc.FredisDb.mu.Unlock()

	log.Printf("[DEL] Deleted key: %s", key)
	return true
}

func (fc *FredisCmds) SetExpiry(key string, seconds int64) (int8, error) {

	if !fc.IsReplaying {
		fc.AOF.LogCommand(fmt.Sprintf("*3\r\n$6\r\nEXPIRE\r\n$%d\r\n%s\r\n$%d\r\n%d\r\n",
			len(key), key,
			len(strconv.FormatInt(seconds, 10)), seconds))
	}

	now := time.Now()

	fc.FredisDb.mu.RLock()
	existing, exists := fc.FredisDb.data[key]
	if !exists {
		fc.FredisDb.mu.RUnlock()
		log.Printf("[EXPIRE] Key does not exist: %s", key)
		return -1, errors.New("key does not exist")
	}

	existing.mu.Lock()
	defer existing.mu.Unlock()

	if existing.Expiry != nil && now.After(*existing.Expiry) {
		fc.FredisDb.mu.RUnlock()
		go fc.DelValue(key)
		log.Printf("[EXPIRE] Key already expired: %s", key)
		return -2, errors.New("key has expired")
	}

	expiry := now.Add(time.Duration(seconds) * time.Second)
	existing.Expiry = &expiry
	fc.FredisDb.mu.RUnlock()

	log.Printf("[EXPIRE] Expiry set for key %s to %v", key, expiry)
	return 1, nil
}

func (fc *FredisCmds) TTL(key string) int {
	now := time.Now()

	fc.FredisDb.mu.RLock()
	defer fc.FredisDb.mu.RUnlock()
	existing, exists := fc.FredisDb.data[key]

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
		go fc.DelValue(key)
		log.Printf("[TTL] Key expired on TTL: %s", key)
		return -2
	}

	ttl := int(existing.Expiry.Sub(now).Seconds())
	log.Printf("[TTL] TTL for key %s: %ds", key, ttl)
	return ttl
}

func (fc *FredisCmds) evict() {
	switch fc.FredisDb.EvictionPolicy {
	case "volatile-lru":
		fc.evictVolatileLRU()
	case "allkeys-random":
		fc.evictAllKeysRandom()
	}
}

func (fc *FredisCmds) evictVolatileLRU() {
	var lruKey string
	var oldest time.Time

	fc.FredisDb.mu.Lock()
	defer fc.FredisDb.mu.Unlock()

	for key, val := range fc.FredisDb.data {
		val.mu.RLock()
		if val.Expiry != nil {
			if lruKey == "" || val.LastAccess.Before(oldest) {
				lruKey = key
				oldest = val.LastAccess
			}
		}
		val.mu.Unlock()
	}

	if lruKey != "" {
		log.Println("[Evict] Volatile LRU evicting:", lruKey)
		fc.DelValue(lruKey)
	}
}

func (fc *FredisCmds) evictAllKeysRandom() {
	fc.FredisDb.mu.Lock()
	defer fc.FredisDb.mu.Unlock()

	keys := make([]string, 0, len(fc.FredisDb.data))
	for k := range fc.FredisDb.data {
		keys = append(keys, k)
	}

	if len(keys) == 0 {
		return
	}

	randomKey := keys[rand.Intn(len(keys))]
	log.Println("[Evict] Randomly evicting:", randomKey)
	fc.DelValue(randomKey)
}

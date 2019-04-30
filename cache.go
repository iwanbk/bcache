package bcache

import (
	"sync"
	"time"

	"github.com/hashicorp/golang-lru"
	"github.com/weaveworks/mesh"
)

type cache struct {
	peerID mesh.PeerName
	mux    sync.RWMutex
	cc     *lru.Cache
}

func newCache(maxKeys int) (*cache, error) {
	cc, err := lru.New(maxKeys)
	if err != nil {
		return nil, err
	}

	return &cache{
		cc: cc,
	}, nil
}

// value represent cache value
type value struct {
	value   string
	expired int64 // expiration timestamp of the value
	deleted int64 // deletion timestamp of the value
}

// Set sets the value of a cache
func (c *cache) Set(key, val string, expiredTimestamp, deleted int64) {
	c.cc.Add(key, value{
		value:   val,
		expired: expiredTimestamp,
		deleted: deleted,
	})

}

// Delete del the value of a cache.
// returns true if the key exists in cache, false otherwise
func (c *cache) Delete(key string, deleteTimestamp int64) (string, int64, bool) {
	val, ok := c.get(key)
	if !ok {
		return "", 0, false
	}
	c.Set(key, val.value, val.expired, deleteTimestamp)

	return val.value, val.expired, true
}

// Get gets cache value of the given key
func (c *cache) get(key string) (*value, bool) {
	cacheVal, ok := c.cc.Get(key)
	if !ok {
		return nil, false
	}
	val := cacheVal.(value)
	return &val, true
}

// Get gets cache value of the given key
func (c *cache) Get(key string) (string, bool) {
	val, ok := c.get(key)
	if !ok {
		return "", false
	}

	now := time.Now().UnixNano()

	if now >= val.expired || (now >= val.deleted && val.deleted > 0) {
		// delete the key if:
		// - expired
		// - deleted
		c.cc.Remove(key)
		return "", false
	}

	return val.value, val.deleted <= 0
}

func (c *cache) Messages() *message {
	m := newMessage(c.peerID, c.cc.Len())

	for _, k := range c.cc.Keys() {
		key := k.(string)
		cacheVal, ok := c.get(key)
		if !ok {
			continue
		}
		m.add(key, cacheVal.value, cacheVal.expired, cacheVal.deleted)
	}
	return m

}

// merges received data into state and returns a
// representation of the received data (typically a delta) for further
// propagation.
func (c *cache) mergeDelta(msg *message) (delta mesh.GossipData) {
	delta, _ = c.mergeChange(msg)
	return delta
}

// merges received data into state and returns "everything new
// I've just learnt", or nil if nothing in the received data was new.
func (c *cache) mergeNew(msg *message) (delta mesh.GossipData) {
	var changedKey int
	delta, changedKey = c.mergeChange(msg)
	if changedKey == 0 {
		return nil
	}
	return delta
}

func (c *cache) mergeChange(msg *message) (delta mesh.GossipData, changedKey int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	if len(msg.Entries) == 0 {
		return
	}

	var existingKeys []string
	for key, e := range msg.Entries {
		cacheVal, ok := c.get(key)
		if ok && cacheVal.expired >= e.Expired && cacheVal.deleted == e.Deleted {
			// no changes:
			// - key already exists
			// - has bigger expiration value
			// - has same deleted val
			existingKeys = append(existingKeys, key)
			continue
		}
		c.Set(key, e.Val, e.Expired, e.Deleted)
		changedKey++
	}

	// delete key that already existed in this cache
	for _, key := range existingKeys {
		delete(msg.Entries, key)
	}

	return newMessageFromEntries(c.peerID, msg.Entries), changedKey
}

func (c *cache) mergeComplete(msg *message) {
	for key, ent := range msg.Entries {
		cacheVal, ok := c.get(key)
		if !ok || cacheVal.expired < ent.Expired {
			// if !exist in cache, set it
			// if val in cache is older, set it
			c.Set(key, ent.Val, ent.Expired, ent.Deleted)
		}
	}
}

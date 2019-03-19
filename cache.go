package bcache

import (
	"sync"
	//"time"

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
	expired int64
	deleted bool
}

// Set sets the value of a cache
func (c *cache) Set(key, val string, expiredTimestamp int64, deleted bool) {
	c.cc.Add(key, value{
		value:   val,
		expired: expiredTimestamp,
		deleted: deleted,
	})

}

// Delete del the value of a cache.
// returns true if the key exists in cache, false otherwise
func (c *cache) Delete(key string, expiredTimestamp int64) bool {
	val, ok := c.get(key)
	if !ok {
		return false
	}
	c.Set(key, val.value, expiredTimestamp, true)

	return true
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
func (c *cache) Get(key string) (string, int64, bool) {
	val, ok := c.get(key)
	if !ok || val.deleted {
		return "", 0, false
	}
	return val.value, val.expired, true
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
	for _, e := range msg.Entries {
		cacheVal, ok := c.get(e.Key)
		if ok && cacheVal.expired >= e.Expired {
			// the key already exists and has bigger expiration value
			existingKeys = append(existingKeys, e.Key)
			continue
		}
		c.Set(e.Key, e.Val, e.Expired, e.Deleted)
		changedKey++
	}

	// delete key that already existed in this cache
	for _, key := range existingKeys {
		delete(msg.Entries, key)
	}

	return newMessageFromEntries(c.peerID, msg.Entries), changedKey
}

func (c *cache) mergeComplete(msg *message) {
	for _, ent := range msg.Entries {
		cacheVal, ok := c.get(ent.Key)
		if !ok || cacheVal.expired < ent.Expired {
			// if !exist in cache, set it
			// if val in cache is older, set it
			c.Set(ent.Key, ent.Val, ent.Expired, ent.Deleted)
		}
	}
}

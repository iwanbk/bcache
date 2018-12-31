package bcache

import (
	"log"
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

func newLRU(maxKeys int) (*lru.Cache, error) {
	return lru.New(maxKeys)
}

// TODO: optimize it:
// - use goroutines to do the encoding
func (c *cache) Encode() [][]byte {
	c.mux.RLock()
	keys := c.cc.Keys()
	c.mux.RUnlock()

	var (
		result [][]byte
		msg    = newMessage(c.peerID, 0)
	)

	// func to encode message and add it to the [][]byte result
	encodeMsg := func() {
		b, err := msg.encode()
		if err != nil {
			log.Printf("failed to encode cache(%v)", err)
		} else {
			result = append(result, b)
		}

	}

	// iterate all the keys
	for _, k := range keys {
		key := k.(string)

		val, expired, ok := getCache(c.cc, key)
		if !ok {
			continue
		}

		msg.add(key, val, expired)

		if !msg.fullCap() {
			continue
		}
		encodeMsg()
		msg = newMessage(c.peerID, 0)
	}

	if !msg.empty() {
		encodeMsg()
	}
	return result
}

// value represent cache value
type value struct {
	value   string
	expired int64
}

func (c *cache) Set(key, val string, expiredTimestamp int64) *cache {
	setCache(c.cc, key, val, expiredTimestamp)
	return &cache{
		cc: c.cc,
	}
}

func setCache(cc *lru.Cache, key, val string, expired int64) {
	cc.Add(key, value{
		value:   val,
		expired: expired,
	})
}

func (c *cache) Get(key string) (string, bool) {
	val, _, ok := getCache(c.cc, key)
	return val, ok
}

func getCache(cc *lru.Cache, key string) (string, int64, bool) {
	cacheVal, ok := cc.Get(key)
	if !ok {
		return "", 0, false
	}
	val := cacheVal.(value)

	// check for expiration
	if val.expired > 0 && time.Now().Unix() > val.expired {
		cc.Remove(key)
		return "", 0, false
	}

	return val.value, val.expired, true

}

func (c *cache) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return c.mergeComplete(other.(*cache).copy().cc)
}

func (c *cache) mergeComplete(cc *lru.Cache) (complete mesh.GossipData) {
	c.mux.Lock()
	defer c.mux.Unlock()

	keys := c.cc.Keys()

	for _, k := range keys {
		key := k.(string)
		val, expired, ok := getCache(c.cc, key)
		if !ok {
			log.Printf("[error]mergeComplete no val for key: %v", key)
			continue
		}
		setCache(c.cc, key, val, expired)
	}
	complete = &cache{
		cc: c.cc,
	}
	return
}

func (c *cache) mergeReceived(msg *message) (delta mesh.GossipData) {
	delta, _ = c.mergeChange(msg)
	return delta
}

func (c *cache) mergeDelta(msg *message) (delta mesh.GossipData) {
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

	var (
		lenEntries = len(msg.Entries)
	)

	if lenEntries == 0 {
		return
	}

	// create copied cache to send
	copyCache, err := newLRU(lenEntries)
	if err != nil {
		log.Printf("[mergeDelta][error] create lru cache: %v", err)
		return
	}

	for _, e := range msg.Entries {
		val, ok := c.cc.Get(e.Key)
		if ok && val == e.Val {
			continue
		}
		setCache(c.cc, e.Key, e.Val, e.Expired)
		setCache(copyCache, e.Key, e.Val, e.Expired)

		changedKey++
	}
	return &cache{
		cc: copyCache,
	}, changedKey
}
func (c *cache) copy() *cache {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return &cache{
		cc: c.cc,
	}
}

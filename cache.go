package bcache

import (
	"log"
	"sync"

	"github.com/hashicorp/golang-lru"
	"github.com/weaveworks/mesh"
)

type cache struct {
	mux sync.RWMutex
	cc  *lru.Cache
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
// - use fastest encoder
// - use goroutines to do the encoding
func (c *cache) Encode() [][]byte {
	c.mux.RLock()
	defer c.mux.RUnlock()

	var result [][]byte

	keys := c.cc.Keys()
	for _, key := range keys {
		val, ok := c.cc.Get(key)
		if !ok {
			continue
		}
		e := entry{
			Key: key,
			Val: val,
		}

		b, err := e.encode()
		if err != nil {
			log.Printf("failed to encode cache(%v)", key)
			continue
		}
		result = append(result, b)
	}
	return result
}

func (c *cache) Set(key, val interface{}) *cache {
	c.cc.Add(key, val)
	return &cache{
		cc: c.cc,
	}
}

func (c *cache) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return c.mergeComplete(other.(*cache).copy().cc)
}

func (c *cache) mergeComplete(cc *lru.Cache) (complete mesh.GossipData) {
	c.mux.Lock()
	defer c.mux.Unlock()

	keys := c.cc.Keys()

	for _, key := range keys {
		val, ok := c.cc.Get(key)
		if !ok {
			log.Printf("[error]mergeComplete no val for key: %v", key)
			continue
		}
		c.cc.Add(key, val)
	}
	complete = &cache{
		cc: c.cc,
	}
	return
}

func (c *cache) mergeReceived(e *entry) (delta mesh.GossipData) {
	cc, err := newLRU(1)
	if err != nil {
		log.Printf("[mergeDelta][error] create lru cache: %v", err)
		return nil
	}

	val, ok := c.cc.Get(e.Key)
	if !ok || val != e.Val {
		c.cc.Add(e.Key, e.Val)
		cc.Add(e.Key, e.Val)
	}

	return &cache{
		cc: cc,
	}
}

func (c *cache) mergeDelta(e *entry) (delta mesh.GossipData) {
	val, ok := c.cc.Get(e.Key)
	if ok && val == e.Val {
		return nil
	}
	cc, err := newLRU(1)
	if err != nil {
		log.Printf("[mergeDelta][error] create lru cache: %v", err)
		return nil
	}

	c.cc.Add(e.Key, e.Val)
	cc.Add(e.Key, e.Val)
	return &cache{
		cc: cc,
	}
}
func (c *cache) copy() *cache {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return &cache{
		cc: c.cc,
	}
}

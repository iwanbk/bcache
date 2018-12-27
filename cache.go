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
	keys := c.cc.Keys()
	c.mux.RUnlock()

	// TODO :alloc it with exact length
	var result [][]byte
	var es []entry

	for i, key := range keys {
		val, ok := c.cc.Get(key)
		if !ok {
			continue
		}
		es = append(es, entry{
			Key: key,
			Val: val,
		})

		if i%5 != 0 {
			continue
		}

		b := c.encodeEntries(es)
		if b != nil {
			result = append(result, b)
		}

		// reset entries buffer
		es = []entry{}
	}

	if len(es) > 0 {
		b := c.encodeEntries(es)
		if b != nil {
			result = append(result, b)
		}
	}
	return result
}

func (c *cache) encodeEntries(es []entry) []byte {
	ents := entries{
		Entries: es,
	}
	b, err := ents.encode()
	if err != nil {
		log.Printf("failed to encode cache(%v)", err)
		return nil
	}
	return b
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

func (c *cache) mergeReceived(ents *entries) (delta mesh.GossipData) {
	delta, _ = c.mergeChange(ents)
	return delta
}

func (c *cache) mergeDelta(ents *entries) (delta mesh.GossipData) {
	var changedKey int
	delta, changedKey = c.mergeChange(ents)
	if changedKey == 0 {
		return nil
	}
	return delta
}
func (c *cache) mergeChange(ents *entries) (delta mesh.GossipData, changedKey int) {
	c.mux.Lock()
	defer c.mux.Unlock()

	var (
		lenEntries = len(ents.Entries)
	)
	if lenEntries == 0 {
		return
	}

	cc, err := newLRU(lenEntries)
	if err != nil {
		log.Printf("[mergeDelta][error] create lru cache: %v", err)
		return
	}

	for _, e := range ents.Entries {
		val, ok := c.cc.Get(e.Key)
		if ok && val == e.Val {
			continue
		}
		c.cc.Add(e.Key, e.Val)
		cc.Add(e.Key, e.Val)
		changedKey += 1
	}
	return &cache{
		cc: cc,
	}, changedKey
}
func (c *cache) copy() *cache {
	c.mux.RLock()
	defer c.mux.RUnlock()

	return &cache{
		cc: c.cc,
	}
}

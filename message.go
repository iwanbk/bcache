package bcache

import (
	"log"
	"sync"

	"github.com/weaveworks/mesh"
)

const (
	defaultNumEntries = 5
)

// message defines gossip message used for communication between peers
// TODO: use protobuf for encoding
type message struct {
	mux     sync.RWMutex
	PeerID  mesh.PeerName
	Entries map[string]entry
}

// entry is a single key value entry
type entry struct {
	Key     string
	Val     string
	Expired int64
	Deleted bool
}

func newMessage(peerID mesh.PeerName, numEntries int) *message {
	if numEntries == 0 {
		numEntries = defaultNumEntries
	}
	return newMessageFromEntries(peerID, make(map[string]entry, numEntries))
}

func newMessageFromEntries(peerID mesh.PeerName, entries map[string]entry) *message {
	return &message{
		PeerID:  peerID,
		Entries: entries,
	}
}

func newMessageFromBuf(b []byte) (*message, error) {
	var m message
	err := unmarshal(b, &m)
	return &m, err
}

func (m *message) add(key, val string, expired int64, deleted bool) {
	m.Entries[key] = entry{
		Key:     key,
		Val:     val,
		Expired: expired,
		Deleted: deleted,
	}
}

// Encode implements mesh.GossipData.Encode
// TODO: split the encoding by X number of keys
func (m *message) Encode() [][]byte {
	b, err := marshal(m)
	if err != nil {
		log.Printf("failed to encode message: %v", err)
	}
	return [][]byte{b}
}

// Merge implements mesh.GossipData.Merge
func (m *message) Merge(other mesh.GossipData) (complete mesh.GossipData) {
	return m.mergeComplete(other.(*message))
}

func (m *message) mergeComplete(other *message) mesh.GossipData {
	m.mux.Lock()
	defer m.mux.Unlock()

	for k, v := range other.Entries {
		existing, ok := m.Entries[k]
		if !ok || existing.Expired < v.Expired {
			m.Entries[k] = v
		}
	}
	return newMessageFromEntries(m.PeerID, m.Entries)
}

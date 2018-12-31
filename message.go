package bcache

import (
	"encoding/json"

	"github.com/weaveworks/mesh"
)

const (
	defaultNumEntries = 5
)

// message defines gossip message used for communication between peers
// TODO: use fastest encoding possible
type message struct {
	PeerID  mesh.PeerName
	Entries []entry
	cap     int
}

// entry is a single key value entry
type entry struct {
	Key     string
	Val     string
	Expired int64
}

func newMessage(peerID mesh.PeerName, numEntries int) *message {
	if numEntries == 0 {
		numEntries = defaultNumEntries
	}
	return &message{
		PeerID:  peerID,
		Entries: make([]entry, 0, numEntries),
		cap:     numEntries,
	}
}

func newMessageFromBuf(b []byte) (*message, error) {
	var m message
	err := json.Unmarshal(b, &m)
	return &m, err
}

func (m *message) add(key, val string, expired int64) {
	m.Entries = append(m.Entries, entry{
		Key:     key,
		Val:     val,
		Expired: expired,
	})
}

func (m *message) fullCap() bool {
	return len(m.Entries) >= m.cap
}

func (m *message) empty() bool {
	return len(m.Entries) == 0
}
func (m *message) encode() ([]byte, error) {
	return json.Marshal(m)
}

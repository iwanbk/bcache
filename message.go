package bcache

import (
	"encoding/json"
)

// message defines gossip message used for communication between peers
type message struct {
	Entries []entry
}

// entry is a single key value entry
type entry struct {
	Key interface{}
	Val interface{}
}

func (m *message) encode() ([]byte, error) {
	return json.Marshal(m)
}

func newMessageFromBuf(b []byte) (*message, error) {
	var m message
	err := json.Unmarshal(b, &m)
	return &m, err
}

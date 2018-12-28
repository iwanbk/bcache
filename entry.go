package bcache

import (
	"github.com/iwanbk/bcache/json"
)

type entry struct {
	Key interface{}
	Val interface{}
}

type entries struct {
	Entries []entry
}

func (e *entries) encode() ([]byte, error) {
	return json.Marshal(e)
}

func newEntryFromBuf(b []byte) (*entries, error) {
	var e entries
	return &e, json.Unmarshal(b, &e)
}

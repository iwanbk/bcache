package bcache

import (
	"encoding/json"
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
	err := json.Unmarshal(b, &e)
	return &e, err
}

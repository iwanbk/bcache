package bcache

import (
	"encoding/json"
)

type entry struct {
	Key interface{}
	Val interface{}
}

func (e *entry) encode() ([]byte, error) {
	return json.Marshal(e)
}

func newEntryFromBuf(b []byte) (*entry, error) {
	var e entry
	err := json.Unmarshal(b, &e)
	return &e, err
}

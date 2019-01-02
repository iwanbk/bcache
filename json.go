package bcache

import "github.com/json-iterator/go"

var (
	json      = jsoniter.ConfigCompatibleWithStandardLibrary
	marshal   = json.Marshal
	unmarshal = json.Unmarshal
)

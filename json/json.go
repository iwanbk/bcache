package json

import "github.com/json-iterator/go"

var (
	json = jsoniter.ConfigCompatibleWithStandardLibrary
	// Marshal into json
	Marshal = json.Marshal
	// Unmarshal from json
	Unmarshal = json.Unmarshal
	// NewDecoder create json iter new decoder
	NewDecoder = json.NewDecoder
)

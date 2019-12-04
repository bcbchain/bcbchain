package jsoniter

import (
	"github.com/json-iterator/go"
)

var json = jsoniter.Config{
	EscapeHTML:             false,
	SortMapKeys:            true,
	ValidateJsonRawMessage: true,
}.Froze()

// Marshal convert object to JSON
func Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

// Unmarshal parse JSON and set the result to object _v
func Unmarshal(bz []byte, v interface{}) error {
	return json.Unmarshal(bz, v)
}

package types

import (
	"encoding/hex"
	"fmt"
	"strings"
)

// HexBytes uses for raw bytes is to enable HEX-encoding and JSON-encoding
type HexBytes []byte

// Marshal needed for protobuf compatibility
func (bz HexBytes) Marshal() ([]byte, error) {
	return bz, nil
}

// Unmarshal needed for protobuf compatibility
func (bz *HexBytes) Unmarshal(data []byte) error {
	*bz = data
	return nil
}

// MarshalJSON This is the point of Bytes.
func (bz HexBytes) MarshalJSON() ([]byte, error) {
	s := strings.ToUpper(hex.EncodeToString(bz))
	jbz := make([]byte, len(s)+2)
	jbz[0] = '"'
	copy(jbz[1:], []byte(s))
	jbz[len(jbz)-1] = '"'
	return jbz, nil
}

// UnmarshalJSON This is the point of Bytes.
func (bz *HexBytes) UnmarshalJSON(data []byte) error {
	if data == nil || len(data) < 2 || data[0] != '"' || data[len(data)-1] != '"' {
		return fmt.Errorf("Invalid hex string: %s", data)
	}
	bz2, err := hex.DecodeString(string(data[1 : len(data)-1]))
	if err != nil {
		return err
	}
	*bz = bz2
	return nil
}

// Bytes allow it to fulfill various interfaces in light-client, etc...
func (bz HexBytes) Bytes() []byte {
	return bz
}

// String encode to string by hex
func (bz HexBytes) String() string {
	return strings.ToUpper(hex.EncodeToString(bz))
}

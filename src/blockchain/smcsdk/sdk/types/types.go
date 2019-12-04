package types

// Address uses for Account & Contract
type Address = string

// Hash uses for SHA3 & ...
type Hash = HexBytes

// PubKey uses for public key and others, PubKeyEd25519
type PubKey = HexBytes

// KVPair define key value pair
type KVPair struct {
	Key   []byte `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value []byte `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

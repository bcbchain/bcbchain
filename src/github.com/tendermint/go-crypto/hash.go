package crypto

import (
	"crypto/sha256"
	"golang.org/x/crypto/ripemd160"
	"hash"
)

func Sha256(bytes []byte) []byte {
	hasher := sha256.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

func Ripemd160(bytes []byte) []byte {
	hasher := ripemd160.New()
	hasher.Write(bytes)
	return hasher.Sum(nil)
}

func NewRipemd160() hash.Hash {
	return ripemd160.New()
}

package common

import (
	"github.com/tendermint/go-amino"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/tmlibs/merkle"
	"golang.org/x/crypto/ripemd160"
	"reflect"
)

var CDC = amino.NewCodec()

func init() {
	crypto.RegisterAmino(CDC)
}

type hasher struct {
	item interface{}
}

func (h hasher) Hash() []byte {
	hasher := ripemd160.New()
	if h.item != nil && !isTypedNil(h.item) && !isEmpty(h.item) {
		bz, err := CDC.MarshalBinaryBare(h.item)
		if err != nil {
			panic(err)
		}
		_, err = hasher.Write(bz)
		if err != nil {
			panic(err)
		}
	}
	return hasher.Sum(nil)
}

func AminoHasher(item interface{}) merkle.Hasher {
	return hasher{item}
}

// Go lacks a simple and safe way to see if something is a typed nil.
func isTypedNil(o interface{}) bool {
	rv := reflect.ValueOf(o)
	switch rv.Kind() {
	case reflect.Chan, reflect.Func, reflect.Map, reflect.Ptr, reflect.Slice:
		return rv.IsNil()
	default:
		return false
	}
}

// Returns true if it has zero length.
func isEmpty(o interface{}) bool {
	rv := reflect.ValueOf(o)
	switch rv.Kind() {
	case reflect.Array, reflect.Chan, reflect.Map, reflect.Slice, reflect.String:
		return rv.Len() == 0
	default:
		return false
	}
}

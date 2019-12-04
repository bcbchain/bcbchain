package rpcclient

import (
	amino "github.com/tendermint/go-amino"
	crypto "github.com/tendermint/go-crypto"
)

var CDC = amino.NewCodec()

func init() {
	crypto.RegisterAmino(CDC)
}

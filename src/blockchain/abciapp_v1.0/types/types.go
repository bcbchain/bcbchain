package types

import "github.com/tendermint/go-crypto"

type Ed25519Sig struct {
	SigType  string
	PubKey   crypto.PubKeyEd25519
	SigValue crypto.SignatureEd25519
}

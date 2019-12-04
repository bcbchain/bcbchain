package ed25519

import (
	"github.com/tendermint/go-crypto"
)

// VerifySign verify signature
func VerifySign(pubkey, data, sign []byte) bool {
	pubKey := crypto.PubKeyEd25519FromBytes(pubkey)
	signature := crypto.SignatureEd25519FromBytes(sign)

	return pubKey.VerifyBytes(data, signature)
}

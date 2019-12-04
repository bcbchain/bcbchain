package sig

import (
	"github.com/tendermint/go-crypto"
)

type Ed25519Sig struct {
	SigType  string
	PubKey   crypto.PubKeyEd25519
	SigValue crypto.SignatureEd25519
}

type FileSig struct {
	PubKey1   string `json:"pubkey,omitempty"`       //主链签名文件
	PubKey2   string `json:"publicEccKey,omitempty"` //加密机签名文件
	Signature string `json:"signature"`
}

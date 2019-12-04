package evm

import (
	"crypto/ecdsa"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"testing"

	"github.com/hyperledger/burrow/execution/evm/ecrypto"

	"github.com/btcsuite/btcd/btcec"

	"github.com/stretchr/testify/assert"
)

func TestSigToPub(t *testing.T) {

	privateKey, err := ecdsa.GenerateKey(secp256k1.S256(), rand.Reader)
	assert.Equal(t, nil, err)

	msg := "hello, world"
	hash := sha256.Sum256([]byte(msg))

	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	assert.Equal(t, nil, err)

	fmt.Printf("signature: (r=0x%x, s=0x%x)\n", r, s)
	fmt.Printf("hash: 0x%x\n", hash)

	expect := (*btcec.PublicKey)(&privateKey.PublicKey).SerializeUncompressed()
	fmt.Printf("pubKey: 0x%x\n", expect)

	valid := ecdsa.Verify(&privateKey.PublicKey, hash[:], r, s)
	fmt.Println("signature verified:", valid)

	pk := secp256k1.GenPrivKey()
	sigTendermint, err := pk.Sign(hash[:])
	assert.Equal(t, nil, err)

	_, err = ecrypto.EcRecover(hash[:], sigTendermint[:])
	assert.Equal(t, nil, err)

	sig0, err := ecrypto.Sign(hash[:], privateKey)
	assert.Equal(t, nil, err)
	fmt.Printf("sign is 0x%x\n", sig0)
}

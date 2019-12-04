package crypto

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	secp256k1 "github.com/btcsuite/btcd/btcec"
	"github.com/btcsuite/btcutil/base58"
	"github.com/tendermint/ed25519"
	"github.com/tendermint/ed25519/extra25519"
	cmn "github.com/tendermint/tmlibs/common"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
	"strings"
)

// An address is a []byte, but hex-encoded even in JSON.
// []byte leaves us the option to change the address length.
// Use an alias so Unmarshal methods (with ptr receivers) are available too.
type Address = string

// An UdcHash is a []byte, but hex-encoded even in JSON.
// []byte leaves us the option to change the address length.
// Use an alias so Unmarshal methods (with ptr receivers) are available too.
type Hash = cmn.HexBytes

func PubKeyFromBytes(pubKeyBytes []byte) (pubKey PubKey, err error) {
	err = cdc.UnmarshalBinaryBare(pubKeyBytes, &pubKey)
	return
}

//----------------------------------------

type PubKey interface {
	Address(chainID string) Address
	Bytes() []byte
	VerifyBytes(msg []byte, sig Signature) bool
	Equals(PubKey) bool
}

//-------------------------------------

var chainId []byte
var _ PubKey = PubKeyEd25519{}

// Implements PubKeyInner
type PubKeyEd25519 [32]byte

func SetChainId(cid string) {
	chainId = make([]byte, 0, 0)
	chainId = append(chainId, []byte(cid)...)
}

func GetChainId() string {
	if string(chainId) == "" {
		panic("crypto.SetChainId must be called first")
	}
	return string(chainId)
}

func PubKeyEd25519FromBytes(data []byte) PubKey {
	var pubkey PubKeyEd25519
	copy(pubkey[:], data)
	return pubkey
}

func (pubKey PubKeyEd25519) Address(chainID string) Address {
	if chainID == "" {
		panic("chainID cannot be empty")
	}

	mainChainID := chainID
	if strings.Contains(chainID, "[") {
		mainChainID = chainID[:strings.Index(chainID, "[")]
	}

	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(mainChainID))
	hasherSHA3256.Write(pubKey[:])
	sha := hasherSHA3256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	rpd := hasherRIPEMD160.Sum(nil)

	hasher := ripemd160.New()
	hasher.Write(rpd)
	md := hasher.Sum(nil)

	addr := make([]byte, 0, 0)
	addr = append(addr, rpd...)
	addr = append(addr, md[:4]...)

	return chainID + base58.Encode(addr)
}

func (pubKey PubKeyEd25519) Bytes() []byte {
	bz, err := cdc.MarshalBinaryBare(pubKey)
	if err != nil {
		panic(err)
	}
	return bz
}

func (pubKey PubKeyEd25519) VerifyBytes(msg []byte, sig_ Signature) bool {
	// make sure we use the same algorithm to sign
	sig, ok := sig_.(SignatureEd25519)
	if !ok {
		return false
	}
	pubKeyBytes := [32]byte(pubKey)
	sigBytes := [64]byte(sig)
	return ed25519.Verify(&pubKeyBytes, msg, &sigBytes)
}

// For use with golang/crypto/nacl/box
// If error, returns nil.
func (pubKey PubKeyEd25519) ToCurve25519() *[32]byte {
	keyCurve25519, pubKeyBytes := new([32]byte), [32]byte(pubKey)
	ok := extra25519.PublicKeyToCurve25519(keyCurve25519, &pubKeyBytes)
	if !ok {
		return nil
	}
	return keyCurve25519
}

func (pubKey PubKeyEd25519) String() string {
	return fmt.Sprintf("%X", pubKey[:])
}

func (pubKey PubKeyEd25519) Equals(other PubKey) bool {
	if otherEd, ok := other.(PubKeyEd25519); ok {
		return bytes.Equal(pubKey[:], otherEd[:])
	} else {
		return false
	}
}

//-------------------------------------

var _ PubKey = PubKeySecp256k1{}

// Implements PubKey.
// Compressed pubkey (just the x-cord),
// prefixed with 0x02 or 0x03, depending on the y-cord.
type PubKeySecp256k1 [33]byte

// Implements Bitcoin style addresses: RIPEMD160(SHA256(pubkey))
func (pubKey PubKeySecp256k1) Address(chainID string) Address {
	hasherSHA256 := sha256.New()
	hasherSHA256.Write(pubKey[:]) // does not error
	sha := hasherSHA256.Sum(nil)

	hasherRIPEMD160 := ripemd160.New()
	hasherRIPEMD160.Write(sha) // does not error
	return Address(hasherRIPEMD160.Sum(nil))
}

func (pubKey PubKeySecp256k1) Bytes() []byte {
	bz, err := cdc.MarshalBinaryBare(pubKey)
	if err != nil {
		panic(err)
	}
	return bz
}

func (pubKey PubKeySecp256k1) VerifyBytes(msg []byte, sig_ Signature) bool {
	// and assert same algorithm to sign and verify
	sig, ok := sig_.(SignatureSecp256k1)
	if !ok {
		return false
	}

	pub__, err := secp256k1.ParsePubKey(pubKey[:], secp256k1.S256())
	if err != nil {
		return false
	}
	sig__, err := secp256k1.ParseDERSignature(sig[:], secp256k1.S256())
	if err != nil {
		return false
	}
	return sig__.Verify(Sha256(msg), pub__)
}

func (pubKey PubKeySecp256k1) String() string {
	return fmt.Sprintf("PubKeySecp256k1{%X}", pubKey[:])
}

func (pubKey PubKeySecp256k1) Equals(other PubKey) bool {
	if otherSecp, ok := other.(PubKeySecp256k1); ok {
		return bytes.Equal(pubKey[:], otherSecp[:])
	} else {
		return false
	}
}

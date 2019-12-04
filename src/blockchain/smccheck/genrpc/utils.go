package genrpc

import (
	"blockchain/smcsdk/sdk/crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"fmt"

	"io/ioutil"
	"path/filepath"
)

// SignInfo - sign info of the json file
type SignInfo struct {
	PubKey      string `json:"pubKey"`
	BcbAddr     string `json:"bcbAddr"`
	BcbTestAddr string `json:"bcbTestAddr"`
	DevTestAddr string `json:"devTestAddr"`
	SignStr     string `json:"signStr"`
}

// VerifySign - verify the file
func VerifySign(filename, signFile string) bool {
	rawBytes, err := ioutil.ReadFile(filepath.Clean(filename))
	if err != nil {
		fmt.Printf("Read file \"%v\" failed, %v\n", filename, err.Error())
		return false
	}
	sigBytes, err := ioutil.ReadFile(filepath.Clean(signFile))
	if err != nil {
		fmt.Printf("Read file \"%v\" failed, %v\n", signFile, err.Error())
		return false
	}

	si := SignInfo{}
	err = json.Unmarshal(sigBytes, &si)
	if err != nil || si.PubKey == "" || si.SignStr == "" {
		fmt.Printf("UnmarshalJSON from file \"%v\" failed, %v\n", signFile, err.Error())
		return false
	}

	pubKey, err := hex.DecodeString(si.PubKey)
	if err != nil {
		fmt.Printf("UnmarshalJSON from file \"%v\" failed, %v\n", signFile, err.Error())
		return false
	}

	signature, err := hex.DecodeString(si.SignStr)
	if err != nil {
		fmt.Printf("UnmarshalJSON from file \"%v\" failed, %v\n", signFile, err.Error())
		return false
	}

	ret := ed25519.VerifySign(pubKey, rawBytes, signature)

	if ret == false {
		fmt.Println("Verify signature failed")
	}

	return ret
}

package crypto

import (
	"bytes"
	"crypto/sha256"
	bin "encoding/binary"
	"encoding/json"
	"fmt"
	"github.com/tendermint/go-crypto"
	"github.com/tendermint/go-wire/data/base58"

	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto/sha3"
	"github.com/tmthrgd/go-hex"
)

type EVMAddress binary.Word160

type EVMAddresses []EVMAddress

func (as EVMAddresses) Len() int {
	return len(as)
}

func (as EVMAddresses) Less(i, j int) bool {
	return bytes.Compare(as[i][:], as[j][:]) < 0
}
func (as EVMAddresses) Swap(i, j int) {
	as[i], as[j] = as[j], as[i]
}

const AddressLength = binary.Word160Length
const AddressHexLength = 2 * AddressLength

var ZeroAddress = EVMAddress{}

func ToAddr(address EVMAddress) crypto.Address {
	hasher := crypto.NewRipemd160()
	hasher.Write(address[:])
	md := hasher.Sum(nil)

	addr := make([]byte, 0, 0)
	addr = append(addr, address[:]...)
	addr = append(addr, md[:4]...)

	return crypto.GetChainId() + base58.Encode(addr)
}

func ToEVM(address crypto.Address) EVMAddress {
	bareAddr := address[len(crypto.GetChainId()):] // trim chainId
	byt, _ := base58.Decode(bareAddr)
	ret := EVMAddress{}
	copy(ret[:], byt[:20])
	return ret
}

// Returns a pointer to an EVMAddress that is nil iff len(bs) == 0 otherwise does the same as AddressFromBytes
func MaybeAddressFromBytes(bs []byte) (*EVMAddress, error) {
	if len(bs) == 0 {
		return nil, nil
	}
	address, err := AddressFromBytes(bs)
	if err != nil {
		return nil, err
	}
	return &address, nil
}

// Returns an address consisting of the first 20 bytes of bs, return an error if the bs does not have length exactly 20
// but will still return either: the bytes in bs padded on the right or the first 20 bytes of bs truncated in any case.
func AddressFromBytes(bs []byte) (address EVMAddress, err error) {
	if len(bs) != binary.Word160Length {
		err = fmt.Errorf("slice passed as address '%X' has %d bytes but should have %d bytes",
			bs, len(bs), binary.Word160Length)
		// It is caller's responsibility to check for errors. If they ignore the error we'll assume they want the
		// best-effort mapping of the bytes passed to an address so we don't return here
	}
	copy(address[:], bs)
	return
}

func AddressFromHexString(str string) (EVMAddress, error) {
	bs, err := hex.DecodeString(str)
	if err != nil {
		return ZeroAddress, err
	}
	return AddressFromBytes(bs)
}

func MustAddressFromHexString(str string) EVMAddress {
	address, err := AddressFromHexString(str)
	if err != nil {
		panic(fmt.Errorf("error reading address from hex string: %s", err))
	}
	return address
}

func MustAddressFromBytes(addr []byte) EVMAddress {
	address, err := AddressFromBytes(addr)
	if err != nil {
		panic(fmt.Errorf("error reading address from bytes: %s", err))
	}
	return address
}

func AddressFromWord256(addr binary.Word256) EVMAddress {
	return EVMAddress(addr.Word160())
}

func (address EVMAddress) Word256() binary.Word256 {
	return binary.Word160(address).Word256()
}

// Copy address and return a slice onto the copy
func (address EVMAddress) Bytes() []byte {
	addressCopy := address
	return addressCopy[:]
}

func (address EVMAddress) String() string {
	return hex.EncodeUpperToString(address[:])
}

func (address *EVMAddress) UnmarshalJSON(data []byte) error {
	str := new(string)
	err := json.Unmarshal(data, str)
	if err != nil {
		return err
	}
	err = address.UnmarshalText([]byte(*str))
	if err != nil {
		return err
	}
	return nil
}

func (address EVMAddress) MarshalJSON() ([]byte, error) {
	text, err := address.MarshalText()
	if err != nil {
		return nil, err
	}
	return json.Marshal(string(text))
}

func (address *EVMAddress) UnmarshalText(text []byte) error {
	if len(text) != AddressHexLength {
		return fmt.Errorf("address hex '%s' has length %v but must have length %v to be a valid address",
			string(text), len(text), AddressHexLength)
	}
	_, err := hex.Decode(address[:], text)
	return err
}

func (address EVMAddress) MarshalText() ([]byte, error) {
	return ([]byte)(hex.EncodeUpperToString(address[:])), nil

}

// Gogo proto support
func (address *EVMAddress) Marshal() ([]byte, error) {
	if address == nil {
		return nil, nil
	}
	return address.Bytes(), nil
}

func (address *EVMAddress) Unmarshal(data []byte) error {
	if len(data) == 0 {
		return nil
	}
	if len(data) != binary.Word160Length {
		return fmt.Errorf("error unmarshallling address '%X' from bytes: %d bytes but should have %d bytes",
			data, len(data), binary.Word160Length)
	}
	copy(address[:], data)
	return nil
}

func (address *EVMAddress) MarshalTo(data []byte) (int, error) {
	return copy(data, address[:]), nil
}

func (address *EVMAddress) Size() int {
	return binary.Word160Length
}

func (address *EVMAddress) Equal(b EVMAddress) bool {
	return bytes.Equal(address[:], b[:])
}

func Nonce(caller EVMAddress, nonce []byte) []byte {
	hasher := sha256.New()
	hasher.Write(caller[:]) // does not error
	hasher.Write(nonce)
	return hasher.Sum(nil)
}

// Obtain a nearly unique nonce based on a montonic account sequence number
func SequenceNonce(address EVMAddress, sequence uint64) []byte {
	bs := make([]byte, 8)
	bin.BigEndian.PutUint64(bs, sequence)
	return Nonce(address, bs)
}

func NewContractAddress(caller EVMAddress, nonce []byte) (newAddr EVMAddress) {
	copy(newAddr[:], Nonce(caller, nonce))
	return
}

func NewContractAddress2(caller EVMAddress, salt [binary.Word256Length]byte, initcode []byte) (newAddr EVMAddress) {
	// sha3(0xff ++ caller.EVMAddress() ++ salt ++ sha3(init_code))[12:]
	temp := make([]byte, 0, 1+AddressLength+2*binary.Word256Length)
	temp = append(temp, []byte{0xFF}...)
	temp = append(temp, caller[:]...)
	temp = append(temp, salt[:]...)
	temp = append(temp, sha3.Sha3(initcode)...)
	copy(newAddr[:], sha3.Sha3(temp)[12:])
	return
}

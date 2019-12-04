package algorithm

import (
	"bytes"
	"crypto/aes"
	"crypto/md5"
	"encoding/binary"
	"errors"
	"math/big"
	"strconv"
	"strings"

	"github.com/btcsuite/btcutil/base58"
	"github.com/tendermint/go-crypto"
	"golang.org/x/crypto/ripemd160"
	"golang.org/x/crypto/sha3"
)

func IntToBytes(n int) []byte {
	tmp := int32(n)
	bytesBuffer := bytes.NewBuffer([]byte{})
	binary.Write(bytesBuffer, binary.BigEndian, tmp)
	return bytesBuffer.Bytes()
}

//字节转换成整形
func BytesToInt(b []byte) int {
	return int(BytesToInt32(b))
}

func BytesToInt32(b []byte) int32 {
	var bytesBuffer *bytes.Buffer
	if len(b) < 8 {
		bytesBuffer = bytes.NewBuffer(make([]byte, 8-len(b)))
		bytesBuffer.Write(b)
	} else {
		bytesBuffer = bytes.NewBuffer(b)
	}

	var tmp int64
	err := binary.Read(bytesBuffer, binary.BigEndian, &tmp)
	if err != nil {
		panic(err)
	}
	return int32(tmp)
}

func BytesToUint32(b []byte) uint32 {
	return uint32(BytesToInt32(b))
}

func BytesToUint64(b []byte) uint64 {
	tx8 := make([]byte, 8)
	copy(tx8[len(tx8)-len(b):], b)
	return binary.BigEndian.Uint64(tx8[:])
}

func BytesToInt64(b []byte) int64 {
	return int64(BytesToUint64(b))
}

func CalcAddressFromCdcPubKey(chainID string, pubKey []byte) crypto.Address {
	crptPubKey, err := crypto.PubKeyFromBytes(pubKey)
	if err != nil {
		panic(err)
	}
	return crptPubKey.Address(chainID)
}

func CheckAddress(chainID string, addr string) error {
	if addr == "" {
		return errors.New("Address cannot be empty!")
	}
	if !strings.HasPrefix(addr, chainID) {
		return errors.New("Address chainid is error!")
	}
	base58Addr := strings.Replace(addr, chainID, "", 1)
	addrData := base58.Decode(base58Addr)
	len := len(addrData)
	if len < 4 {
		return errors.New("Base58Addr parse error!")
	}

	hasher := ripemd160.New()
	hasher.Write(addrData[:len-4])
	md := hasher.Sum(nil)

	if bytes.Compare(md[:4], addrData[len-4:]) != 0 {
		return errors.New("Address checksum is error!")
	}

	return nil
}

//定义合约账户地址的计算方法
func CalcContractAddress(chainID string, ownerAddr crypto.Address, contractName, version string) crypto.Address {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(chainID))
	hasherSHA3256.Write([]byte(contractName))
	hasherSHA3256.Write([]byte(version))
	hasherSHA3256.Write([]byte(ownerAddr))
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

	return string(chainID) + base58.Encode(addr)
}

//定义UDC哈希的计算方法
func CalcUdcHash(nonce uint64, token, owner crypto.Address, value big.Int, matureDate string) crypto.Hash {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(strconv.FormatUint(nonce, 10)))
	hasherSHA3256.Write([]byte(token))
	hasherSHA3256.Write([]byte(owner))
	hasherSHA3256.Write(value.Bytes())
	hasherSHA3256.Write([]byte(matureDate))

	return hasherSHA3256.Sum(nil)
}

// 定义MethodID的计算方法
func CalcMethodId(protoType string) []byte {
	// 计算sha3-256, 取前4字节
	d := sha3.New256()
	d.Write([]byte(protoType))
	b := d.Sum(nil)
	return b[0:4]
}

func ConvertMethodID(b []byte) string {
	return strconv.FormatInt(int64(binary.BigEndian.Uint32(b)), 16)
}

//定义合约代码哈希的计算方法
func CalcCodeHash(code string) []byte {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte(code))
	return hasherSHA3256.Sum(nil)
}

//Sha3_256
func SHA3256(datas ...[]byte) []byte {

	hasherSHA3256 := sha3.New256()
	for _, data := range datas {
		hasherSHA3256.Write(data)
	}
	return hasherSHA3256.Sum(nil)
}

//定义对称密钥生成算法
func GenSymmetrickeyFromPassword(password, keyword []byte) []byte {
	hasherSHA3256 := sha3.New256()
	hasherSHA3256.Write([]byte("7g$2HJJhh&&!^&!nNN8812MN31^%!@%*^&*&((&*152"))
	hasherSHA3256.Write(password[:])
	if keyword != nil {
		hasherSHA3256.Write(keyword[:])
	}
	sha := hasherSHA3256.Sum(nil)
	digest := md5.New()
	digest.Write(sha) // does not error
	return digest.Sum(nil)
}

//定义对称加密算法
func EncryptWithPassword(data, password, keyword []byte) []byte {
	if data == nil {
		return nil
	}
	key := GenSymmetrickeyFromPassword(password, keyword)
	enc, _ := aes.NewCipher(key)
	blockSize := enc.BlockSize()
	dat := make([]byte, len(data)+8)
	copy(dat, []byte{0x2e, 0x77, 0x61, 0x6c})
	copy(dat[4:], IntToBytes(len(data)))
	copy(dat[8:], data)
	if n := len(dat) % blockSize; n != 0 {
		m := blockSize - n
		for i := 0; i < m; i++ {
			dat = append(dat, 0)
		}
	}
	for i := 0; i < len(dat)/blockSize; i++ {
		enc.Encrypt(dat[i*blockSize:(i+1)*blockSize], dat[i*blockSize:(i+1)*blockSize])
	}
	return dat
}

//定义对称解密算法
func DecryptWithPassword(data, password, keyword []byte) ([]byte, error) {
	if data == nil || len(data) == 0 {
		return nil, errors.New("Cannot decrypt empty data")
	}

	key := GenSymmetrickeyFromPassword(password, keyword)
	dec, _ := aes.NewCipher(key)
	blockSize := dec.BlockSize()

	if len(data)%blockSize != 0 {
		return nil, errors.New("Decrypt data is not an integral multiple of a block")
	}

	dat := make([]byte, len(data))
	copy(dat, data)
	for i := 0; i < len(dat)/blockSize; i++ {
		dec.Decrypt(dat[i*blockSize:(i+1)*blockSize], dat[i*blockSize:(i+1)*blockSize])
	}
	if len(dat) < 8 {
		return nil, errors.New("Decrypt data failed!")
	}
	mac := make([]byte, 4)
	copy(mac, dat[:4])
	if bytes.Compare(mac, []byte{0x2e, 0x77, 0x61, 0x6c}) != 0 {
		return nil, errors.New("Decrypt data failed!")
	}
	size := BytesToInt(dat[4:8])
	return dat[8 : 8+size], nil
}

package bignumber

import (
	"encoding/binary"
	"github.com/pkg/errors"
	"math/big"
	"reflect"
)

func Compare(v1, v2 big.Int) int {
	return v1.Cmp(&v2)
}

func Zero() big.Int {
	return *big.NewInt(0)
}

func Add(v1, v2 big.Int) big.Int {
	var v1_n, v2_n big.Int
	v1_n.Set(&v1)
	v2_n.Set(&v2)

	return *v1_n.Add(&v1_n, &v2_n)
}

func Sub(v1, v2 big.Int) big.Int {
	var v1_n, v2_n big.Int
	v1_n.Set(&v1)
	v2_n.Set(&v2)

	return *v1_n.Sub(&v1_n, &v2_n)
}

func Multi(v big.Int, m interface{}) (big.Int, error) {
	switch m.(type) {
	case int8, int16, int, int32, int64, uint, uint8, uint16, uint32:
		rv := reflect.ValueOf(m)
		var old big.Int
		return *old.Mul(&v, big.NewInt(rv.Int())), nil
	default:
		return *big.NewInt(0), errors.New("Invalid Parameter")
	}
}

func UintToBigInt(val uint64) big.Int {
	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, val)
	return *new(big.Int).SetBytes(buf)
}

func SafeMul(x, y big.Int) (z big.Int) {
	z.Mul(&x, &y)
	return
}

func SafeDiv(x, y big.Int) (z big.Int) {
	if Compare(y, Zero()) == 0 {
		return
	}
	z.Div(&x, &y)
	return
}

// SetBytes interprets buf as the bytes of a big-endian big.Int
func SetBytes(buf []byte) *big.Int {
	//Uses 0xFF to declare a negative
	z := new(big.Int)
	if len(buf) <= 1 {
		z.SetBytes(buf[:])
	} else if buf[0] == 0xFF {
		z.Neg(z.SetBytes(buf[1:]))
	} else {
		z.SetBytes(buf[:])
	}
	return z
}

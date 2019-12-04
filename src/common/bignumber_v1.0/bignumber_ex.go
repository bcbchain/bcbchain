package bignumber

import (
	"math/big"
)

type Number struct {
	V *big.Int `json:"v"`
}

func N(x int64) Number                { return NewNumber(x) }
func N1(b int64, d int64) Number      { return NewNumberLong(b, d) }
func N2(b int64, d1, d2 int64) Number { return NewNumberLongLong(b, d1, d2) }
func NB(x *big.Int) Number            { return NewNumberBigInt(x) }

//大数标准编码方式,只能转换为正式
func NBytes(x []byte) Number {
	n := N(0)
	n.V.SetBytes(x)
	return n
}

func NewNumberString(s string) Number {
	n := N(0)
	n.V.SetString(s, 10)
	return n
}

func NewNumberStringBase(s string, base int) Number {
	n := N(0)
	n.V.SetString(s, base)
	return n
}

func NewNumber(x int64) Number {
	v := Number{V: big.NewInt(x)}
	return v
}

func NewNumberBigInt(x *big.Int) Number {
	v := Number{V: x}
	return v
}

func NewNumberLong(b int64, d int64) Number {
	x := Number{V: big.NewInt(b)}
	y := Number{V: big.NewInt(d)}
	return x.Mul(y)
}

func NewNumberLongLong(b int64, d1, d2 int64) Number {
	x := Number{V: big.NewInt(b)}
	y := Number{V: big.NewInt(d1)}
	z := Number{V: big.NewInt(d2)}
	return x.Mul(y).Mul(z)
}

func (x Number) String() string {
	return x.V.String()
}

func (x Number) Value() *big.Int {
	return x.V
}

func (x Number) Cmp_(y int64) int {
	return x.V.Cmp(N(y).V)
}

func (x Number) Cmp(y Number) int {
	return x.V.Cmp(y.V)
}

func (x Number) Add_(y int64) Number {
	return x.Add(N(y))
}

func (x Number) Add(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Add(x.V, y.V)
	return z
}

func (x Number) Sub_(y int64) Number {
	return x.Sub(N(y))
}

func (x Number) Sub(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Sub(x.V, y.V)
	return z
}

func (x Number) Mul_(y int64) Number {
	return x.Mul(N(y))
}

func (x Number) Mul(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Mul(x.V, y.V)
	return z
}

func (x Number) Div_(y int64) Number {
	return x.Div(N(y))
}

func (x Number) Div(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Div(x.V, y.V)
	return z
}

func (x Number) Mod_(y int64) Number {
	return x.Mod(N(y))
}

func (x Number) Mod(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Mod(x.V, y.V)
	return z
}

func (x Number) Sq() Number {
	return x.Mul(x)
}

func (x Number) Sqrt() Number {
	z := x.Add_(1).Div_(2)
	y := x
	for z.Cmp(y) < 0 {
		y = z
		z = x.Div(z).Add(z).Div_(2)
	}
	return y
}

func (x Number) AndByBits(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.And(x.V, y.V)
	return z
}

func (x Number) Exp(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Exp(x.V, y.V, nil)
	return z
}

// SetBytes interprets buf as the bytes of a big-endian Number
// sets z to that value, and returns z.
func (z Number) SetBytes(buf []byte) Number {
	//Uses 0xFF to declare a negative
	if len(buf) <= 1 {
		z.V.SetBytes(buf[:])
	} else if buf[0] == 0xFF {
		z.V.Neg(z.V.SetBytes(buf[1:]))
	} else {
		z.V.SetBytes(buf[:])
	}
	return z
}

// Bytes returns the value of x as a big-endian byte slice.
func (x Number) Bytes() []byte {
	z := x.V.Bytes()
	buf := make([]byte, len(z)+1)
	if x.Cmp_(0) < 0 {
		buf[0] = 0xFF
	} else {
		buf[0] = 0x0
	}
	copy(buf[1:], z)
	return buf
}

package bn

import (
	"errors"
	"math/big"
	"strings"
)

//Number wrap big.Int
type Number struct {
	V *big.Int `json:"v"`
}

//N convert x to Number
func N(x int64) Number {
	return NewNumber(x)
}

//N1 convert b*d to Number
func N1(b int64, d int64) Number {
	return NewNumberLong(b, d)
}

//N2 convert b*d1*d2 to Number
func N2(b int64, d1, d2 int64) Number {
	return NewNumberLongLong(b, d1, d2)
}

//NB convert x to Number
func NB(x *big.Int) Number {
	return NewNumberBigInt(x)
}

//NBS convert big-endian encoded unsigned bytes bs to Number
func NBS(bs []byte) Number {
	return NBytes(bs)
}

//NSBS convert big-endian encoded signed bytes bs to Number,
//    the first byte == 0xFF means the sign is negative
func NSBS(bs []byte) Number {
	return NSBytes(bs)
}

//NString convert s to Number
func NString(s string) Number {
	return NewNumberStringBase(s, 10)
}

//NStringHex convert s to Number
func NStringHex(s string) Number {
	if strings.HasPrefix(s, "0x") || strings.HasPrefix(s, "0X") {
		return NewNumberStringBase(s, 0)
	}
	return N(0)
}

//NBytes convert big-endian encoded unsigned bytes bs to Number
func NBytes(bs []byte) Number {
	n := N(0)
	n.V.SetBytes(bs)
	return n
}

//NSBytes convert big-endian encoded signed bytes bs to Number,
//       the first byte == 0xFF means the sign is negative
func NSBytes(x []byte) Number {
	n := N(0)
	n.SetBytes(x)
	return n
}

//NewNumber convert x to Number
func NewNumber(x int64) Number {
	v := Number{V: big.NewInt(x)}
	return v
}

const MaxBase = 10 + ('z' - 'a' + 1) + ('Z' - 'A' + 1)

//NewNumberStringBase convert the value s to Number, interpreted in the given base,
//and returns z and a boolean indicating success. The entire string
//(not just a prefix) must be valid for success. If SetString fails,
//the value of z is undefined but the returned value is nil.
//
//The base argument must be 0 or a value between 2 and MaxBase. If the base
//is 0, the string prefix determines the actual conversion base. A prefix of
//``0x'' or ``0X'' selects base 16; the ``0'' prefix selects base 8, and a
//``0b'' or ``0B'' prefix selects base 2. Otherwise the selected base is 10.
//
//For bases <= 36, lower and upper case letters are considered the same:
//The letters 'a' to 'z' and 'A' to 'Z' represent digit values 10 to 35.
//For bases > 36, the upper case letters 'A' to 'Z' represent the digit
//values 36 to 61.
//
func NewNumberStringBase(s string, base int) Number {
	n := N(0)
	_, success := n.V.SetString(s, base)
	if success {
		return n
	}
	return N(0)
}

//NewNumberBigInt convert x to Number
func NewNumberBigInt(x *big.Int) Number {
	y := *x
	v := Number{V: &y}
	return v
}

//NewNumberLong convert b*d to Number
func NewNumberLong(b int64, d int64) Number {
	x := Number{V: big.NewInt(b)}
	y := Number{V: big.NewInt(d)}
	return x.Mul(y)
}

//NewNumberLongLong convert b*d1*d2 to Number
func NewNumberLongLong(b int64, d1, d2 int64) Number {
	x := Number{V: big.NewInt(b)}
	y := Number{V: big.NewInt(d1)}
	z := Number{V: big.NewInt(d2)}
	return x.Mul(y).Mul(z)
}

//String returns the string representation of x in base 10.
func (x Number) String() string {
	return x.V.String()
}

//Value returns the big.Int representation of x.
func (x Number) Value() *big.Int {
	return x.V
}

//CmpI compares x and y and returns:
//    -1 if x <  y
//     0 if x == y
//    +1 if x >  y
func (x Number) CmpI(y int64) int {
	return x.V.Cmp(N(y).V)
}

//Cmp compares x and y and returns:
//    -1 if x <  y
//     0 if x == y
//    +1 if x >  y
func (x Number) Cmp(y Number) int {
	return x.V.Cmp(y.V)
}

//IsZero test whether x is zero or not
func (x Number) IsZero() bool {
	return x.CmpI(0) == 0
}

//IsPositive test whether x is positive or not
func (x Number) IsPositive() bool {
	return x.CmpI(0) > 0
}

//IsNegative test whether x is negative or not
func (x Number) IsNegative() bool {
	return x.CmpI(0) < 0
}

//IsEqualI test whether x is equal to y or not
func (x Number) IsEqualI(y int64) bool {
	return x.CmpI(y) == 0
}

//IsEqual test whether x is equal to y or not
func (x Number) IsEqual(y Number) bool {
	return x.Cmp(y) == 0
}

//IsGreaterThanI test whether x is greater than y or not
func (x Number) IsGreaterThanI(y int64) bool {
	return x.CmpI(y) > 0
}

//IsGreaterThan test whether x is greater than y or not
func (x Number) IsGreaterThan(y Number) bool {
	return x.Cmp(y) > 0
}

//IsLessThanI test whether x is less than y or not
func (x Number) IsLessThanI(y int64) bool {
	return x.CmpI(y) < 0
}

//IsLessThan test whether x is less than y or not
func (x Number) IsLessThan(y Number) bool {
	return x.Cmp(y) < 0
}

//IsGEI test whether x is greater equal than y or not
func (x Number) IsGEI(y int64) bool {
	return x.CmpI(y) >= 0
}

//IsGE test whether x is greater equal than y or not
func (x Number) IsGE(y Number) bool {
	return x.Cmp(y) >= 0
}

//IsLEI test whether x is less equal than y or not
func (x Number) IsLEI(y int64) bool {
	return x.CmpI(y) <= 0
}

//IsLE test whether x is less equal than y or not
func (x Number) IsLE(y Number) bool {
	return x.Cmp(y) <= 0
}

// AddI calculate x+y and returns the result.
func (x Number) AddI(y int64) Number {
	return x.Add(N(y))
}

// Add calculate x+y and returns the result.
func (x Number) Add(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Add(x.V, y.V)
	return z
}

// SubI calculate x-y and returns the result.
func (x Number) SubI(y int64) Number {
	return x.Sub(N(y))
}

// Sub calculate x-y and returns the result.
func (x Number) Sub(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Sub(x.V, y.V)
	return z
}

// MulI calculate x*y and returns the result.
func (x Number) MulI(y int64) Number {
	return x.Mul(N(y))
}

// Mul calculate x*y and returns the result.
func (x Number) Mul(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Mul(x.V, y.V)
	return z
}

// DivI calculate x/y and returns the result.
func (x Number) DivI(y int64) Number {
	return x.Div(N(y))
}

// Div calculate x/y and returns the result.
func (x Number) Div(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Div(x.V, y.V)
	return z
}

// ModI calculate x%y and returns the result.
func (x Number) ModI(y int64) Number {
	return x.Mod(N(y))
}

// Mod calculate x%y and returns the result.
func (x Number) Mod(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Mod(x.V, y.V)
	return z
}

// Sq calculate x**2 and returns the result.
func (x Number) Sq() Number {
	return x.Mul(x)
}

// Sqrt calculate ⌊√x⌋ and returns the result.
// It panics if x is negative.
func (x Number) Sqrt() Number {
	if x.IsNegative() {
		panic("square root of negative number")
	}

	z := x.AddI(1).DivI(2)
	y := x
	for z.Cmp(y) < 0 {
		y = z
		z = x.Div(z).Add(z).DivI(2)
	}
	return y
}

// Exp calculate x**y and returns the result.
func (x Number) Exp(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Exp(x.V, y.V, nil)
	return z
}

// Lsh calculate x<<n and returns the result.
func (x Number) Lsh(n uint) Number {
	z := Number{V: new(big.Int)}
	z.V.Lsh(x.V, n)
	return z
}

// Rsh calculate x>>n and returns the result.
func (x Number) Rsh(n uint) Number {
	z := Number{V: new(big.Int)}
	z.V.Rsh(x.V, n)
	return z
}

// And calculate x&y and returns the result.
func (x Number) And(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.And(x.V, y.V)
	return z
}

// Or calculate x|y and returns the result.
func (x Number) Or(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Or(x.V, y.V)
	return z
}

// Xor calculate x^y and returns the result.
func (x Number) Xor(y Number) Number {
	z := Number{V: new(big.Int)}
	z.V.Xor(x.V, y.V)
	return z
}

// Not calculate ^x and returns the result.
func (x Number) Not() Number {
	z := Number{V: new(big.Int)}
	z.V.Not(x.V)
	return z
}

//Bytes returns the value of x as a big-endian byte slice.
//    the first byte == 0xFF means the sign is negative
func (x Number) Bytes() []byte {
	if x.V == nil {
		return nil
	}
	z := x.V.Bytes()
	buf := make([]byte, len(z)+1)
	if x.CmpI(0) < 0 {
		buf[0] = 0xFF
	} else {
		buf[0] = 0x0
	}
	copy(buf[1:], z)
	return buf
}

//SetBytes set big-endian encoded signed bytes bs to x and return the result,
//       the first byte == 0xFF means the sign is negative
func (x *Number) SetBytes(buf []byte) Number {
	//Uses 0xFF to declare a negative
	if x.V == nil {
		x.V = big.NewInt(0)
	}
	if buf != nil && len(buf) > 0 {
		if buf[0] == 0xFF {
			x.V.Neg(x.V.SetBytes(buf[1:]))
		} else {
			x.V.SetBytes(buf[:])
		}
	}
	return *x
}

//MarshalJSON json serialization for x
func (x *Number) MarshalJSON() (data []byte, err error) {
	if x == nil || x.V == nil {
		return []byte("null"), nil
	}
	return []byte(x.V.String()), nil
}

//UnmarshalJSON json deserialization for data and convert to x
func (x *Number) UnmarshalJSON(data []byte) error {
	str := strings.TrimSpace(string(data))
	if strings.HasPrefix(str, "{") {
		bare := str[1 : len(str)-1]
		spl := strings.Split(bare, ":")
		if len(spl) != 2 || strings.TrimSpace(spl[0]) != "\"v\"" {
			return errors.New("wrong bn.number json format")
		}
		str = strings.TrimSpace(spl[1])
	}
	if x.V == nil {
		x.V = new(big.Int)
	}
	x.V.SetString(str, 10)
	return nil
}

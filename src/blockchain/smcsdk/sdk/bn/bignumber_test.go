package bn

import (
	"blockchain/smcsdk/sdk/jsoniter"
	"encoding/json"
	"fmt"
	"testing"
)

func TestNumber(t *testing.T) {
	var v Number
	var v1 Number
	var v2 Number

	fmt.Println(v.String())
	v.SetBytes([]byte("\xFF\x01\x00"))
	fmt.Println(v)

	v = NString("123")
	fmt.Println(v)

	v = NString("-123")
	fmt.Println(v)

	v = NStringHex("0x200")
	fmt.Println(v)

	v = NString("8")
	fmt.Println(v)
	fmt.Println(v.Lsh(1))
	fmt.Println(v.Lsh(2))
	fmt.Println(v.Lsh(3))
	fmt.Println(v.Lsh(4))
	fmt.Println(v)
	fmt.Println(v.Rsh(1))
	fmt.Println(v.Rsh(2))
	fmt.Println(v.Rsh(3))
	fmt.Println(v.Rsh(4))

	v = NString("-8")
	fmt.Println(v)
	fmt.Println(v.Lsh(1))
	fmt.Println(v.Lsh(2))
	fmt.Println(v.Lsh(3))
	fmt.Println(v.Lsh(4))
	fmt.Println(v)
	fmt.Println(v.Rsh(1))
	fmt.Println(v.Rsh(2))
	fmt.Println(v.Rsh(3))
	fmt.Println(v.Rsh(4))

	v1 = NString("4")
	v2 = NStringHex("0x0F")
	fmt.Printf("0x%X\n", v1.And(v2).Bytes())
	fmt.Printf("0x%X\n", v1.Or(v2).Bytes())
	fmt.Printf("0x%X\n", v1.Xor(v2).Bytes())
	fmt.Printf("0x%X\n", v1.Not().Bytes())

	v1 = NString("-4")
	v2 = NStringHex("0x0F")
	fmt.Printf("0x%X\n", v1.And(v2).Bytes())
	fmt.Printf("0x%X\n", v1.Or(v2).Bytes())
	fmt.Printf("0x%X\n", v1.Xor(v2).Bytes())
	fmt.Printf("0x%X\n", v1.Not().Bytes())

	v1 = N2(-123456, 1000000000, 100000000)
	s, _ := jsoniter.Marshal(v1)
	fmt.Printf("%s\n", s)
	var v4 Number
	jsoniter.Unmarshal([]byte("{\"v\":-12345600000000000000000}"), &v4)
	fmt.Printf("%s\n", v4)
	//fmt.Printf("%s\n", v3)

	v1 = N2(-123456, 1000000000, 100000000)
	s, _ = json.Marshal(v1)
	fmt.Printf("%s\n", s)
	var v5 Number
	json.Unmarshal([]byte("{\"v\":-12345600000000000000000}"), &v5)
	fmt.Printf("%s\n", v5)
}

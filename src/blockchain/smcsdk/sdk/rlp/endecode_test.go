package rlp

import (
	"reflect"
	"testing"
)

func TestEncodeAndDecode(t *testing.T) {
	mp := make(map[int]string)

	mp[10] = "1111111"
	mp[20] = "222222222"
	mp[30] = "333333333333"
	b, _ := EncodeToBytes(mp)

	temp := make(map[int]string)
	DecodeBytes(b, &temp)
	if !reflect.DeepEqual(mp, temp) {
		panic("Decode map failed")
	}

	mpp := make(map[string]map[int]string)
	mpp["999"] = mp
	mpp["111"] = mp
	mpp["222"] = mp
	mpp["333"] = mp
	mpp["444"] = mp
	mpp["555"] = make(map[int]string)
	bp, _ := EncodeToBytes(mpp)
	tmpp := make(map[string]map[int]string)
	DecodeBytes(bp, &tmpp)
	if !reflect.DeepEqual(mpp, tmpp) {
		panic("Decode map failed")
	}

	type testrlp struct {
		N1    int8
		N2    int64
		Un1   uint8
		Un2   uint64
		Str   string
		Mp    map[int]string
		Bl    bool
		Slice []int
		Bc    byte
	}
	sl := []int{1, 2, 3, 4, 5}
	tstruct := testrlp{
		N1:    99,
		N2:    8888,
		Un1:   66,
		Un2:   44444,
		Str:   "testKMap",
		Mp:    mp,
		Bl:    true,
		Slice: sl,
		Bc:    'c',
	}

	mpstruct := make(map[int]testrlp)
	mpstruct[1] = tstruct
	bmps, _ := EncodeToBytes(mpstruct)

	temp1 := make(map[int]testrlp)
	DecodeBytes(bmps, &temp1)
	if !reflect.DeepEqual(mpstruct, temp1) {
		panic("Decode map failed")
	}
}

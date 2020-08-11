package statedb

import (
	"crypto/md5"
	"github.com/bcbchain/sdk/sdk/bn"
	"strconv"
)

type conflictBits struct {
	length   uint
	bitCount uint
	bits1    []uint64
	bits2    []uint64
	bits3    []uint64
}

func newConflictBits(size int) *conflictBits {
	return &conflictBits{
		length:   (uint(size) + 63) / 64,
		bitCount: ((uint(size) + 63) / 64) * 64,
	}
}

func (bits *conflictBits) ensure_memory() {
	if bits.bits1 == nil {
		bits.bits1 = make([]uint64, bits.length)
		bits.bits2 = make([]uint64, bits.length)
		bits.bits3 = make([]uint64, bits.length)
	}
}

func (bits *conflictBits) String() string {
	return bits.String1()
}

func (bits *conflictBits) String1() string {
	if bits.bits1 == nil {
		return ""
	}

	s := ""
	for i := uint(0); i < bits.length; i++ {
		n := bits.bits1[i]
		for k := uint(0); k < 64; k++ {
			s += strconv.FormatInt(int64(n>>k)&1, 2)
		}
	}
	return s
}

func (bits *conflictBits) String2() string {
	if bits.bits2 == nil {
		return ""
	}

	s := ""
	for i := uint(0); i < bits.length; i++ {
		n := bits.bits2[i]
		for k := uint(0); k < 64; k++ {
			s += strconv.FormatInt(int64(n>>k)&1, 2)
		}
	}
	return s
}

func (bits *conflictBits) String3() string {
	if bits.bits3 == nil {
		return ""
	}

	s := ""
	for i := uint(0); i < bits.length; i++ {
		n := bits.bits3[i]
		for k := uint(0); k < 64; k++ {
			s += strconv.FormatInt(int64(n>>k)&1, 2)
		}
	}
	return s
}

func (bits *conflictBits) Set(key []byte) {
	bits.ensure_memory()

	bits.set1(bits.calcIndex1(key))
	bits.set2(bits.calcIndex2(key))
	bits.set3(bits.calcIndex3(key))
}

func (bits *conflictBits) calcIndex(poly uint, key []byte) uint {
	h := md5.New()
	h.Write(bn.N(int64(poly)).Bytes())
	h.Write(key)
	return uint(bn.NBytes(h.Sum(nil)).ModI(int64(bits.bitCount)).Value().Int64())
}

func (bits *conflictBits) calcIndex1(key []byte) uint {
	poly := uint(0xedb88320)
	return bits.calcIndex(poly, key)
}

func (bits *conflictBits) calcIndex2(key []byte) uint {
	poly := uint(0x82f63b78)
	return bits.calcIndex(poly, key)
}

func (bits *conflictBits) calcIndex3(key []byte) uint {
	poly := uint(0xeb31d82e)
	return bits.calcIndex(poly, key)
}

func (bits *conflictBits) set1(index uint) {
	i := index / 64
	b := uint(index % 64)
	bits.bits1[i] = bits.bits1[i] | (1 << b)
}

func (bits *conflictBits) set2(index uint) {
	i := index / 64
	b := uint(index % 64)
	bits.bits2[i] = bits.bits2[i] | (1 << b)
}

func (bits *conflictBits) set3(index uint) {
	i := index / 64
	b := uint(index % 64)
	bits.bits3[i] = bits.bits3[i] | (1 << b)
}

func (bits *conflictBits) IsConflictTo(c *conflictBits) bool {
	bits.ensure_memory()

	c1 := false
	c2 := false
	c3 := false
	for i := uint(0); i < bits.length; i++ {
		if bits.bits1[i]&c.bits1[i] != 0 {
			c1 = true
			break
		}
	}
	for i := uint(0); i < bits.length; i++ {
		if bits.bits2[i]&c.bits2[i] != 0 {
			c2 = true
			break
		}
	}
	for i := uint(0); i < bits.length; i++ {
		if bits.bits3[i]&c.bits3[i] != 0 {
			c3 = true
			break
		}
	}
	return c1 == true && c2 == true && c3 == true
}

func (bits *conflictBits) Merge(c *conflictBits) (m *conflictBits) {
	m = newConflictBits(int(bits.bitCount))
	m.ensure_memory()
	c.ensure_memory()
	bits.ensure_memory()

	for i := uint(0); i < bits.length; i++ {
		m.bits1[i] = bits.bits1[i] | c.bits1[i]
		m.bits2[i] = bits.bits2[i] | c.bits2[i]
		m.bits3[i] = bits.bits3[i] | c.bits3[i]
	}
	return
}

func (bits *conflictBits) Clear() {
	bits.bits1 = nil
	bits.bits2 = nil
	bits.bits3 = nil
}

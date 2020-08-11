package statedb

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestConflictBits(t *testing.T) {
	cb1 := newConflictBits(127)
	cb2 := newConflictBits(127)

	cb1.Set([]byte("a"))
	cb1.Set([]byte("b"))
	cb1.Set([]byte("c"))
	cb1.Set([]byte("d"))
	cb1.Set([]byte("e"))
	cb1.Set([]byte("f"))
	cb1.set1(0)
	cb1.set1(1)
	cb1.set1(63)
	cb1.set1(64)
	cb1.set1(65)
	cb1.set1(126)
	cb1.set1(127)
	assert.Panics(t, func() { cb1.set1(128) }, "")
	assert.Panics(t, func() { cb1.set1(129) }, "")
	assert.Panics(t, func() { cb1.set1(82763) }, "")
	assert.Panics(t, func() { cb1.set2(128) }, "")
	assert.Panics(t, func() { cb1.set2(129) }, "")
	assert.Panics(t, func() { cb1.set2(82763) }, "")
	assert.Panics(t, func() { cb1.set3(128) }, "")
	assert.Panics(t, func() { cb1.set3(129) }, "")
	assert.Panics(t, func() { cb1.set3(82763) }, "")

	cb1.set1(1)
	cb2.set1(65)
	fmt.Println("cb1:", cb1.String1())
	fmt.Println("cb2:", cb2.String1())
	assert.Equal(t, cb1.IsConflictTo(cb2), false)
	cb2.Set([]byte("f"))
	cb2.Set([]byte("123456"))
	assert.Equal(t, cb1.IsConflictTo(cb2), true)
	cb1.Clear()
	cb2.Clear()
	cb1.set1(73)
	cb1.set2(73)
	cb1.set3(73)
	cb2.set1(73)
	cb2.set2(78)
	cb2.set3(73)
	assert.Equal(t, cb1.IsConflictTo(cb2), false)
	cb2.set2(73)
	assert.Equal(t, cb1.IsConflictTo(cb2), true)
	cb1.Clear()
	cb2.Clear()
	cb1.Set([]byte("123456"))
	cb2.Set([]byte("123456"))
	assert.Equal(t, cb1.IsConflictTo(cb2), true)

	cb1.Clear()
	cb2.Clear()
	cb1.Set([]byte("123456"))
	cb2.Set([]byte("123457"))
	assert.Equal(t, cb1.IsConflictTo(cb2), false)
}
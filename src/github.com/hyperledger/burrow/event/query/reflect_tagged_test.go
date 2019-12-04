package query

import (
	"testing"

	"github.com/hyperledger/burrow/binary"
	"github.com/hyperledger/burrow/crypto"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTaggable struct {
	Foo     string
	Bar     string
	Baz     binary.HexBytes
	Address crypto.EVMAddress
	Indices []int
}

func TestReflectTagged_Keys(t *testing.T) {
	rt, err := ReflectTags(&testTaggable{})
	require.NoError(t, err)
	assert.Equal(t, []string{"Foo", "Bar", "Baz", "EVMAddress", "Indices"}, rt.Keys())
}

func TestReflectTagged_Get(t *testing.T) {
	tt := testTaggable{
		Foo:     "Thumbs",
		Bar:     "Numbed",
		Baz:     []byte{255, 255, 255},
		Address: crypto.EVMAddress{1, 2, 3},
		Indices: []int{5, 7, 9},
	}
	rt, err := ReflectTags(&tt)
	require.NoError(t, err)

	value, ok := rt.Get("Foo")
	assert.True(t, ok)
	assert.Equal(t, tt.Foo, value)

	value, ok = rt.Get("Bar")
	assert.True(t, ok)
	assert.Equal(t, tt.Bar, value)

	value, ok = rt.Get("Baz")
	assert.True(t, ok)
	assert.Equal(t, "FFFFFF", value)

	value, ok = rt.Get("Indices")
	assert.True(t, ok)
	assert.Equal(t, "5;7;9", value)

	value, ok = rt.Get("EVMAddress")
	assert.True(t, ok)
	assert.Equal(t, "0102030000000000000000000000000000000000", value)

	// Make sure we see updates through pointer
	tt.Foo = "Plums"
	value, ok = rt.Get("Foo")
	assert.True(t, ok)
	assert.Equal(t, tt.Foo, value)
}

func TestReflectTagged_Len(t *testing.T) {
	rt, err := ReflectTags(&testTaggable{})
	require.NoError(t, err)
	assert.Equal(t, 5, rt.Len())
}

func TestExplicitFields(t *testing.T) {
	tt := testTaggable{
		Foo:     "Thumbs",
		Bar:     "Numbed",
		Baz:     []byte{255, 255, 255},
		Address: crypto.EVMAddress{1, 2, 3},
	}
	rt, err := ReflectTags(&tt, "Foo", "EVMAddress")
	require.NoError(t, err)

	value, ok := rt.Get("Foo")
	assert.True(t, ok)
	assert.Equal(t, tt.Foo, value)

	value, ok = rt.Get("EVMAddress")
	assert.True(t, ok)
	assert.Equal(t, "0102030000000000000000000000000000000000", value)

	_, ok = rt.Get("Bar")
	assert.False(t, ok)

	_, ok = rt.Get("Barsss")
	assert.False(t, ok)

	_, err = ReflectTags(&tt, "Foo", "EVMAddress", "Balloons")
	require.Error(t, err)
}

func TestReflectTagged_nil(t *testing.T) {
	type testStruct struct {
		Foo string
	}

	var ts *testStruct

	rf, err := ReflectTags(ts)
	require.NoError(t, err)
	value, ok := rf.Get("Foo")
	assert.False(t, ok)
	assert.Equal(t, "", value)
}

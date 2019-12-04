package bn

import (
	"encoding/json"
	"github.com/stretchr/testify/assert"
	"reflect"
	"testing"
)

func TestNB(t *testing.T) {
	x := N(1000)

	y, e := json.Marshal(x)
	assert.Equal(t, e, nil)

	var z1, z2, z3 Number
	e = json.Unmarshal([]byte("1000"), &z1)
	assert.Equal(t, e, nil)
	assert.Equal(t, reflect.DeepEqual(x, z1), true)

	e = json.Unmarshal(y, &z2)
	assert.Equal(t, e, nil)
	assert.Equal(t, reflect.DeepEqual(x, z2), true)

	e = json.Unmarshal([]byte("{\"v\":1000}"), &z3)
	assert.Equal(t, e, nil)
	assert.Equal(t, reflect.DeepEqual(x, z3), true)
}

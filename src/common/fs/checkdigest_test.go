package fs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMD5Gen(t *testing.T) {
	byt, err := Sha2Gen("/home/rustic/Downloads/hugo_0.48_Linux-64bit.tar.gz")
	assert.Equal(t, err, nil)
	assert.Equal(t, byt, true)
}

func TestCheckMD5(t *testing.T) {
	check := CheckSha2("/home/rustic/zzz/Ergo_Chef_My_Juicer_2.mp4")
	assert.Equal(t, check, true)
}

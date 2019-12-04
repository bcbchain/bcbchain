package sha3

import (
	xsha3 "golang.org/x/crypto/sha3"
)

func Sum224(datas ...[]byte) []byte {

	hasher := xsha3.New224()
	for _, data := range datas {
		hasher.Write(data)
	}
	return hasher.Sum(nil)
}

func Sum256(datas ...[]byte) []byte {

	hasher := xsha3.New256()
	for _, data := range datas {
		hasher.Write(data)
	}
	return hasher.Sum(nil)
}

func Sum384(datas ...[]byte) []byte {

	hasher := xsha3.New384()
	for _, data := range datas {
		hasher.Write(data)
	}
	return hasher.Sum(nil)
}

func Sum512(datas ...[]byte) []byte {

	hasher := xsha3.New512()
	for _, data := range datas {
		hasher.Write(data)
	}
	return hasher.Sum(nil)
}

package bc

import (
	"fmt"

	"github.com/bcbchain/bcbchain/hyperledger/burrow/execution/bvm/asm"
)

type ByteSliceAble interface {
	Bytes() []byte
}

// Concatenate multiple byte slices without unnecessary copying
func Concat(bss ...[]byte) []byte {
	offset := 0
	for _, bs := range bss {
		offset += len(bs)
	}
	bytes := make([]byte, offset)
	offset = 0
	for _, bs := range bss {
		for i, b := range bs {
			bytes[offset+i] = b
		}
		offset += len(bs)
	}
	return bytes
}

// Splice or panic
func MustSplice(byteLikes ...interface{}) []byte {
	spliced, err := Splice(byteLikes...)
	if err != nil {
		panic(err)
	}
	return spliced
}

// Convenience function to allow us to mix bytes, ints, and OpCodes that
// represent bytes in an BVM assembly code to make assembly more readable.
// Also allows us to splice together assembly
// fragments because any []byte arguments are flattened in the result.
func Splice(byteLikes ...interface{}) ([]byte, error) {
	bytes := make([]byte, 0, len(byteLikes))
	for _, byteLike := range byteLikes {
		bs, err := byteSlicify(byteLike)
		if err != nil {
			return nil, err
		}
		bytes = append(bytes, bs...)
	}
	return bytes, nil
}

// Convert anything byte or byte slice like to a byte slice
func byteSlicify(byteLike interface{}) ([]byte, error) {
	switch b := byteLike.(type) {
	case byte:
		return []byte{b}, nil
	case asm.OpCode:
		return []byte{byte(b)}, nil
	case int:
		if int(byte(b)) != b {
			return nil, fmt.Errorf("the int %v does not fit inside a byte", b)
		}
		return []byte{byte(b)}, nil
	case int64:
		if int64(byte(b)) != b {
			return nil, fmt.Errorf("the int64 %v does not fit inside a byte", b)
		}
		return []byte{byte(b)}, nil
	case uint64:
		if uint64(byte(b)) != b {
			return nil, fmt.Errorf("the uint64 %v does not fit inside a byte", b)
		}
		return []byte{byte(b)}, nil
	case string:
		return []byte(b), nil
	case ByteSliceAble:
		return b.Bytes(), nil
	case []byte:
		return b, nil
	default:
		return nil, fmt.Errorf("could not convert %s to a byte or sequence of bytes", byteLike)
	}
}

package utils

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"strconv"
	"strings"
)

func ParseHexUint64(hexStr string, fieldName string) (uint64, error) {
	if !strings.HasPrefix(hexStr, "0x") {
		return 0, paramError(fieldName, "must begin with '0x'")
	}
	hexStr = string([]byte(hexStr)[2:])
	if len(hexStr)%2 != 0 {
		return 0, paramError(fieldName, "must be hex string with even length")
	}
	hexBytes, _ := hex.DecodeString(hexStr)
	if sz := len(hexBytes); sz > 8 {
		return 0, paramError(fieldName, "is too large")
	} else if sz == 0 {
		return 0, paramError(fieldName, "must not be null")
	} else if sz < 8 {
		zeros := []byte{0, 0, 0, 0, 0, 0, 0}
		buf := bytes.NewBuffer(zeros[:8-sz])
		buf.Write(hexBytes)
		hexBytes = buf.Bytes()
	}
	valUint64 := binary.BigEndian.Uint64(hexBytes)
	return valUint64, nil
}

func ParseHexUint32(hexStr string, fieldName string) (uint32, error) {
	if !strings.HasPrefix(hexStr, "0x") {
		return 0, paramError(fieldName, "must begin with '0x'")
	}
	hexStr = string([]byte(hexStr)[2:])
	if len(hexStr)%2 != 0 {
		return 0, paramError(fieldName, "must be hex string with even length")
	}
	hexBytes, _ := hex.DecodeString(hexStr)
	if sz := len(hexBytes); sz > 4 {
		return 0, paramError(fieldName, "is too large")
	} else if sz == 0 {
		return 0, paramError(fieldName, "must not be null")
	} else if sz < 4 {
		zeros := []byte{0, 0, 0}
		buf := bytes.NewBuffer(zeros[:4-sz])
		buf.Write(hexBytes)
		hexBytes = buf.Bytes()
	}
	valUint32 := binary.BigEndian.Uint32(hexBytes)
	return valUint32, nil
}

func BytesToHex(valBytes []byte) string {
	return string("0x") + hex.EncodeToString(valBytes)
}

func Uint64ToHex(val uint64) string {
	valBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(valBytes, val)
	return string("0x") + hex.EncodeToString(valBytes)
}

func ParseHexString(hexStr string, fieldName string, lenConstraint int) ([]byte, error) {
	if !strings.HasPrefix(hexStr, "0x") {
		return nil, paramError(fieldName, "must begin with '0x'")
	}
	hexStr = string([]byte(hexStr)[2:])
	if len(hexStr)%2 != 0 {
		return nil, paramError(fieldName, "must be hex string with even length")
	}
	hexBytes, _ := hex.DecodeString(hexStr)
	if lenConstraint > 0 && len(hexBytes) != lenConstraint {
		return nil, paramError(fieldName, "must be "+strconv.Itoa(lenConstraint*2)+" hex-chars")
	}
	return hexBytes, nil
}

func paramError(fieldName, errInfo string) error {
	err := string("{\"code\":-1, \"message\":\"Parameter '") + fieldName + "' " + errInfo + "\",\"data\":\"\"}"
	return errors.New(err)
}

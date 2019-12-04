//unittestplatform
//utils.go 辅助功能

package utest

import (
	"bytes"
	"encoding/binary"
	"golang.org/x/crypto/sha3"
)

func calcMethodID(protoType string) []byte {
	// 计算sha3-256, 取前4字节
	d := sha3.New256()
	_, err := d.Write([]byte(protoType))
	if err != nil {
		panic(err.Error())
	}
	b := d.Sum(nil)
	return b[0:4]
}

//ConvertPrototype2ID convert prototype to method id
func ConvertPrototype2ID(prototype string) uint32 {
	var id uint32
	bytesBuffer := bytes.NewBuffer(calcMethodID(prototype))
	err := binary.Read(bytesBuffer, binary.BigEndian, &id)
	if err != nil {
		panic(err.Error())
	}
	return id
}

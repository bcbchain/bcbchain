package socket

import (
	"bufio"
	"common/jsoniter"
	"encoding/binary"
	"io"
)

type Request struct {
	Method string                 `json:"method"`
	Data   map[string]interface{} `json:"data"`
	Index  uint64                 `json:"index"`
}

type Response struct {
	Code   uint32 `json:"code"`
	Log    string `json:"log"`
	Result struct {
		Method string      `json:"method"`
		Data   interface{} `json:"data"`
		Index  uint64      `json:"index"`
	} `json:"result"`
}

type ReqResp struct {
	Index     uint64
	RespChan  chan *Response
	CloseChan chan error
}

const (
	maxMsgSize = 104857600 // 100MB
)

// WriteMessage writes a varint length-delimited protobuf message.
func writeMessage(data interface{}, w io.Writer) error {

	resBytes, err := jsoniter.Marshal(data)
	if err != nil {
		return err
	}

	return encodeByteSlice(w, resBytes)
}

// ReadMessage reads a varint length-delimited protobuf message.
func readMessage(r io.Reader) (value []byte, err error) {
	return readProtoMsg(r, maxMsgSize)
}

func readProtoMsg(r io.Reader, maxSize int) (value []byte, err error) {
	// binary.ReadVarint takes an io.ByteReader, eg. a bufio.Reader
	reader, ok := r.(*bufio.Reader)
	if !ok {
		reader = bufio.NewReader(r)
	}
	length64, err := binary.ReadVarint(reader)
	if err != nil {
		return
	}
	length := int(length64)
	if length < 0 || length > maxSize {
		return nil, io.ErrShortBuffer
	}
	buf := make([]byte, length)
	if _, err = io.ReadFull(reader, buf); err != nil {
		return
	}

	return buf, nil
}

//-----------------------------------------------------------------------
// NOTE: we copied wire.EncodeByteSlice from go-wire rather than keep
// go-wire as a dep

func encodeByteSlice(w io.Writer, bz []byte) (err error) {
	err = encodeVarint(w, int64(len(bz)))
	if err != nil {
		return
	}
	_, err = w.Write(bz)
	return
}

func encodeVarint(w io.Writer, i int64) (err error) {
	var buf [10]byte
	n := binary.PutVarint(buf[:], i)
	_, err = w.Write(buf[0:n])
	return
}

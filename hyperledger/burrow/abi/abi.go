// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package abi

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/binary"
	"github.com/bcbchain/bcbchain/hyperledger/burrow/execution/exec"
	"io"
)

// The ABI holds information about a contract's context and available
// invokable methods. It will allow you to type check function calls and
// packs data accordingly.
type ABI struct {
	Constructor Method
	Methods     map[string]Method
	Events      map[string]Event
}

type Log struct {
	// Consensus fields:
	// address of the contract that generated the event
	Address Address `json:"address" gencodec:"required"`
	// list of topics provided by the contract.
	Topics []binary.Word256 `json:"topics" gencodec:"required"`
	// supplied by the contract, usually ABI-encoded
	Data binary.HexBytes `json:"data" gencodec:"required"`
}

// CheckpointOracleNewCheckpointVote represents a NewCheckpointVote event raised by the CheckpointOracle contract.
type CheckpointOracleNewCheckpointVote struct {
	Index          uint64
	CheckpointHash [32]byte
	V              uint8
	R              [32]byte
	S              [32]byte
	Raw            exec.LogEvent // Blockchain specific contextual infos
}

// JSON returns a parsed ABI interface and error if it failed.
func JSON(reader io.Reader) (ABI, error) {
	dec := json.NewDecoder(reader)

	var abi ABI
	if err := dec.Decode(&abi); err != nil {
		return ABI{}, err
	}

	return abi, nil
}

// Pack the given method name to conform the ABI. Method call's data
// will consist of method_id, args0, arg1, ... argN. Method id consists
// of 4 bytes and arguments are all 32 bytes.
// Method ids are created from the first 4 bytes of the hash of the
// methods string signature. (signature = baz(uint32,string32))
func (abi ABI) Pack(name string, args ...interface{}) ([]byte, error) {
	// Fetch the ABI of the requested method
	if name == "" {
		// constructor
		arguments, err := abi.Constructor.Inputs.Pack(args...)
		if err != nil {
			return nil, err
		}
		return arguments, nil
	}
	method, exist := abi.Methods[name]
	if !exist {
		return nil, fmt.Errorf("method '%s' not found", name)
	}
	arguments, err := method.Inputs.Pack(args...)
	if err != nil {
		return nil, err
	}
	// Pack up the method ID too if not a constructor and return
	return append(method.ID(), arguments...), nil
}

// Unpack output in v according to the abi specification
func (abi ABI) Unpack(v interface{}, name string, data []byte) (err error) {
	// since there can't be naming collisions with contracts and events,
	// we need to decide whether we're calling a method or an event
	if method, ok := abi.Methods[name]; ok {
		if len(data)%32 != 0 {
			return fmt.Errorf("abi: improperly formatted output: %s - Bytes: [%+v]", string(data), data)
		}
		return method.Outputs.Unpack(v, data)
	}
	if event, ok := abi.Events[name]; ok {
		return event.Inputs.Unpack(v, data)
	}
	return fmt.Errorf("abi: could not locate named method or event")
}

// UnpackIntoMap unpacks a log into the provided map[string]interface{}
func (abi ABI) UnpackIntoMap(v map[string]interface{}, name string, data []byte) (err error) {
	// since there can't be naming collisions with contracts and events,
	// we need to decide whether we're calling a method or an event
	if method, ok := abi.Methods[name]; ok {
		if len(data)%32 != 0 {
			return fmt.Errorf("abi: improperly formatted output")
		}
		return method.Outputs.UnpackIntoMap(v, data)
	}
	if event, ok := abi.Events[name]; ok {
		return event.Inputs.UnpackIntoMap(v, data)
	}
	return fmt.Errorf("abi: could not locate named method or event")
}

// UnmarshalJSON implements json.Unmarshaler interface
func (abi *ABI) UnmarshalJSON(data []byte) error {
	var fields []struct {
		Type            string
		Name            string
		StateMutability string
		Constant        bool
		Anonymous       bool
		Inputs          []Argument
		Outputs         []Argument
	}
	if err := json.Unmarshal(data, &fields); err != nil {
		return err
	}
	abi.Methods = make(map[string]Method)
	abi.Events = make(map[string]Event)
	for _, field := range fields {
		switch field.Type {
		case "constructor":
			abi.Constructor = Method{
				Inputs: field.Inputs,
			}
		// empty defaults to function according to the abi spec
		case "function", "":
			name := field.Name
			_, ok := abi.Methods[name]
			for idx := 0; ok; idx++ {
				name = fmt.Sprintf("%s%d", field.Name, idx)
				_, ok = abi.Methods[name]
			}
			abi.Methods[name] = Method{
				Name:    name,
				RawName: field.Name,
				Const:   field.StateMutability == "view",
				Inputs:  field.Inputs,
				Outputs: field.Outputs,
			}
		case "event":
			name := field.Name
			_, ok := abi.Events[name]
			for idx := 0; ok; idx++ {
				name = fmt.Sprintf("%s%d", field.Name, idx)
				_, ok = abi.Events[name]
			}
			abi.Events[name] = Event{
				Name:      name,
				RawName:   field.Name,
				Anonymous: field.Anonymous,
				Inputs:    field.Inputs,
			}
		}
	}

	return nil
}

// MethodById looks up a method by the 4-byte id
// returns nil if none found
func (abi *ABI) MethodById(sigdata []byte) (*Method, error) {
	if len(sigdata) < 4 {
		return nil, fmt.Errorf("data too short (%d bytes) for abi method lookup", len(sigdata))
	}
	for _, method := range abi.Methods {
		if bytes.Equal(method.ID(), sigdata[:4]) {
			return &method, nil
		}
	}
	return nil, fmt.Errorf("no method with id: %#x", sigdata[:4])
}

// EventByID looks an event up by its topic hash in the
// ABI and returns nil if none found.
func (abi *ABI) EventByID(topic []byte) (*Event, error) {
	for _, event := range abi.Events {
		if bytes.Equal(event.ID().Bytes(), topic) {
			return &event, nil
		}
	}
	return nil, fmt.Errorf("no event with id: %#x", topic)
}

// UnpackLog unpacks a retrieved log into the provided output structure.
func (abi *ABI) UnpackLog(out interface{}, event string, log exec.LogEvent) error {
	if len(log.Data) > 0 {
		if err := abi.Unpack(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed Arguments
	for _, arg := range abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return parseTopics(out, indexed, log.Topics[1:])
}

// UnpackLogIntoMap unpacks a retrieved log into the provided map.
func (abi *ABI) UnpackLogIntoMap(out map[string]interface{}, event string, log exec.LogEvent) error {
	if len(log.Data) > 0 {
		if err := abi.UnpackIntoMap(out, event, log.Data); err != nil {
			return err
		}
	}
	var indexed Arguments
	for _, arg := range abi.Events[event].Inputs {
		if arg.Indexed {
			indexed = append(indexed, arg)
		}
	}
	return parseTopicsIntoMap(out, indexed, log.Topics[1:])
}

// Copyright 2018 The go-ethereum Authors
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
	"encoding/binary"
	"errors"
	"fmt"
	binary2 "github.com/bcbchain/bcbchain/hyperledger/burrow/binary"
	"math/big"
	"reflect"
)

// Big batch of reflect types for topic reconstruction.
var (
	reflectHash    = reflect.TypeOf(Hash{})
	reflectAddress = reflect.TypeOf(Address{})
	reflectBigInt  = reflect.TypeOf(new(big.Int))
)

// parseTopics converts the indexed topic fields into actual log field values.
//
// Note, dynamic types cannot be reconstructed since they get mapped to Keccak256
// hashes as the topic value!
func parseTopics(out interface{}, fields Arguments, topics []binary2.Word256) error {
	// Sanity check that the fields and topics match up
	if len(fields) != len(topics) {
		return errors.New("topic/field count mismatch")
	}
	// Iterate over all the fields and reconstruct them from topics
	for _, arg := range fields {
		if !arg.Indexed {
			return errors.New("non-indexed field in topic reconstruction")
		}
		field := reflect.ValueOf(out).Elem().FieldByName(capitalise(arg.Name))

		// Try to parse the topic back into the fields based on primitive types
		switch field.Kind() {
		case reflect.Bool:
			if topics[0][HashLength-1] == 1 {
				field.Set(reflect.ValueOf(true))
			}
		case reflect.Int8:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(int8(num.Int64())))

		case reflect.Int16:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(int16(num.Int64())))

		case reflect.Int32:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(int32(num.Int64())))

		case reflect.Int64:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(num.Int64()))

		case reflect.Uint8:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(uint8(num.Uint64())))

		case reflect.Uint16:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(uint16(num.Uint64())))

		case reflect.Uint32:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(uint32(num.Uint64())))

		case reflect.Uint64:
			num := new(big.Int).SetBytes(topics[0][:])
			field.Set(reflect.ValueOf(num.Uint64()))

		default:
			// Ran out of plain primitive types, try custom types

			switch field.Type() {
			case reflectHash: // Also covers all dynamic types
				field.Set(reflect.ValueOf(topics[0]))

			case reflectAddress:
				var addr Address
				copy(addr[:], topics[0][HashLength-AddressLength:])
				field.Set(reflect.ValueOf(addr))

			case reflectBigInt:
				num := new(big.Int).SetBytes(topics[0][:])
				field.Set(reflect.ValueOf(num))

			default:
				// Ran out of custom types, try the crazies
				switch {
				// static byte array
				case arg.Type.T == FixedBytesTy:
					reflect.Copy(field, reflect.ValueOf(topics[0][:arg.Type.Size]))
				default:
					return fmt.Errorf("unsupported indexed type: %v", arg.Type)
				}
			}
		}
		topics = topics[1:]
	}
	return nil
}

// parseTopicsIntoMap converts the indexed topic field-value pairs into map key-value pairs
func parseTopicsIntoMap(out map[string]interface{}, fields Arguments, topics []binary2.Word256) error {
	// Sanity check that the fields and topics match up
	if len(fields) != len(topics) {
		return errors.New("topic/field count mismatch")
	}
	// Iterate over all the fields and reconstruct them from topics
	for _, arg := range fields {
		if !arg.Indexed {
			return errors.New("non-indexed field in topic reconstruction")
		}

		switch arg.Type.T {
		case BoolTy:
			out[arg.Name] = topics[0][HashLength-1] == 1
		case IntTy, UintTy:
			num := new(big.Int).SetBytes(topics[0][:])
			out[arg.Name] = num
		case AddressTy:
			var addr Address
			copy(addr[:], topics[0][HashLength-AddressLength:])
			out[arg.Name] = addr
		case HashTy:
			out[arg.Name] = topics[0]
		case FixedBytesTy:
			out[arg.Name] = topics[0][:]
		case StringTy, BytesTy, SliceTy, ArrayTy:
			// Array types (including strings and bytes) have their keccak256 hashes stored in the topic- not a hash
			// whose bytes can be decoded to the actual value- so the best we can do is retrieve that hash
			out[arg.Name] = topics[0]
		case FunctionTy:
			if garbage := binary.BigEndian.Uint64(topics[0][0:8]); garbage != 0 {
				return fmt.Errorf("bind: got improperly encoded function type, got %v", topics[0].Bytes())
			}
			var tmp [24]byte
			copy(tmp[:], topics[0][8:32])
			out[arg.Name] = tmp
		default: // Not handling tuples
			return fmt.Errorf("unsupported indexed type: %v", arg.Type)
		}

		topics = topics[1:]
	}

	return nil
}

// capitalise makes a camel-case string which starts with an upper case character.
func capitalise(input string) string {
	return ToCamelCase(input)
}

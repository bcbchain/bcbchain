package errors

import (
	"blockchain/types"
	"bytes"
	"fmt"
)

type NestedCallError struct {
	CodedError
	Caller     types.Address
	Callee     types.Address
	StackDepth uint64
}

func (err NestedCallError) Error() string {
	return fmt.Sprintf("error in nested call at depth %v: %s (callee) -> %s (caller): %v",
		err.StackDepth, err.Callee, err.Caller, err.CodedError)
}

type CallError struct {
	// The error from the original call which defines the overall error code
	CodedError
	// Errors from nested sub-calls of the original call that may have also occurred
	NestedErrors []NestedCallError
}

func (err CallError) Error() string {
	buf := new(bytes.Buffer)
	buf.WriteString("Call error: ")
	buf.WriteString(err.CodedError.String())
	if len(err.NestedErrors) > 0 {
		buf.WriteString(", nested call errors:\n")
		for _, nestedErr := range err.NestedErrors {
			buf.WriteString(nestedErr.Error())
			buf.WriteByte('\n')
		}
	}
	return buf.String()
}

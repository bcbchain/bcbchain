package abi

import (
	"common/log"
	"fmt"
)

// Variable exist to unpack return values into, so have both the return
// value and its name
type Variable struct {
	Name  string
	Value string
}

func init() {
	var err error
	RevertAbi, err = ReadAbiSpec([]byte(`[{"name":"Error","type":"function","outputs":[{"type":"string"}],"inputs":[{"type":"string"}]}]`))
	if err != nil {
		panic(fmt.Sprintf("internal error: failed to build revert abi: %v", err))
	}
}

// RevertAbi exists to decode reverts. Any contract function call fail using revert(), assert() or require().
// If a function exits this way, the this hardcoded ABI will be used.
var RevertAbi *AbiSpec

// EncodeFunctionCall ABI encodes a function call based on ABI in string abiData
// and the arguments specified as strings.
// The fname specifies which function should called, if
// it doesn't exist exist the fallback function will be called. If fname is the empty
// string, the constructor is called. The arguments must be specified in args. The count
// must match the function being called.
// Returns the ABI encoded function call, whether the function is constant according
// to the ABI (which means it does not modified contract state)
func EncodeFunctionCall(abiData, funcName string, logger log.Logger, args ...interface{}) ([]byte, *FunctionSpec, error) {
	logger.Trace("Packing Call via ABI",
		"spec", abiData,
		"function", funcName,
		"arguments", fmt.Sprintf("%v", args),
	)

	abiSpec, err := ReadAbiSpec([]byte(abiData))
	if err != nil {
		logger.Info("Failed to decode abi spec",
			"abi", abiData,
			"error", err.Error(),
		)
		return nil, nil, err
	}

	packedBytes, funcSpec, err := abiSpec.Pack(funcName, args...)
	if err != nil {
		logger.Info("Failed to encode abi spec",
			"abi", abiData,
			"error", err.Error(),
		)
		return nil, nil, err
	}

	return packedBytes, funcSpec, nil
}

func DecodeFunctionReturn(abiData, name string, data []byte) ([]*Variable, error) {
	abiSpec, err := ReadAbiSpec([]byte(abiData))
	if err != nil {
		return nil, err
	}

	var args []Argument

	if name == "" {
		args = abiSpec.Constructor.Outputs
	} else {
		if _, ok := abiSpec.Functions[name]; ok {
			args = abiSpec.Functions[name].Outputs
		} else {
			args = abiSpec.Fallback.Outputs
		}
	}

	if args == nil {
		return nil, fmt.Errorf("no such function")
	}
	vars := make([]*Variable, len(args))

	if len(args) == 0 {
		return nil, nil
	}

	vals := make([]interface{}, len(args))
	for i := range vals {
		vals[i] = new(string)
	}
	err = Unpack(args, data, vals...)
	if err != nil {
		return nil, err
	}

	for i, a := range args {
		if a.Name != "" {
			vars[i] = &Variable{Name: a.Name, Value: *(vals[i].(*string))}
		} else {
			vars[i] = &Variable{Name: fmt.Sprintf("%d", i), Value: *(vals[i].(*string))}
		}
	}

	return vars, nil
}

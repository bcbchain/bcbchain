package evm

import "github.com/hyperledger/burrow/execution/errors"

func MemoryProvider(memoryProvider func(errors.Sink) Memory) func(*VM) {
	return func(vm *VM) {
		vm.memoryProvider = memoryProvider
	}
}

func DebugOpcodes(vm *VM) {
	vm.debugOpcodes = true
}

func DumpTokens(vm *VM) {
	vm.dumpTokens = true
}

func StackOptions(callStackMaxDepth uint64, dataStackInitialCapacity uint64, dataStackMaxDepth uint64) func(*VM) {
	return func(vm *VM) {
		vm.params.CallStackMaxDepth = callStackMaxDepth
		vm.params.DataStackInitialCapacity = dataStackInitialCapacity
		vm.params.DataStackMaxDepth = dataStackMaxDepth
	}
}

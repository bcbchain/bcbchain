package llfunction

import (
	"blockchain/smcsdk/sdk"
	"blockchain/smcsdk/sdk/bn"
	"blockchain/smcsdk/sdk/std"
	"blockchain/smcsdk/sdk/types"
)

// BuildCallBack call back function of build
type BuildCallBack func(int64, int64, std.ContractMeta) std.BuildResult

// GetCallback call back function of get value from state database
type GetCallback func(int64, int64, string) []byte

// SetCallback call back function of set value to state database
type SetCallback func(int64, int64, map[string][]byte)

// TransferCallBack call back function of transfer
type TransferCallBack func(sdk.ISmartContract, types.Address, types.Address, bn.Number, string) ([]types.KVPair, types.Error)

// GetBlockCallBack call back function of get block data
type GetBlockCallBack func(transID, height int64) std.Block

// IBCInvoke call back function of ibc invoke
type IBCInvoke func(sdk.ISmartContract) (string, []types.KVPair, types.Error)

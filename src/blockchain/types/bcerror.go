package types

//BcError structure of bcerror
type BcError struct {
	ErrorCode uint32 // Error code
	ErrorDesc string // Error description
}

// Error() gets error description with error code
func (bcerror *BcError) Error() string {
	if bcerror.ErrorDesc != "" {
		return bcerror.ErrorDesc
	}

	for _, error := range bcErrors {
		if error.ErrorCode == bcerror.ErrorCode {
			return error.ErrorDesc
		}
	}
	return ""
}

//CodeOK means success
const (
	CodeOK = 200 + iota
)

// internal error code
const (
	ErrInternalFailed = 500 + iota
	ErrDealFailed
	ErrPath
	ErrMarshal
	ErrCallRPC
	ErrAccountLocked
)

//ErrCheckTx beginning error code of checkTx
const (
	ErrCheckTx = 600 + iota
)

//ErrDeliverTx beginning error code of deliverTx
const (
	ErrDeliverTx = 700 + iota
)

const (
	ErrNoAuthorization = 1000 + iota
)

// ErrCodeEVMInvoke beginning error code of EVM execution
const (
	ErrCodeEVMInvoke = 3000 + iota
	ErrRlpDecode
	ErrCallState
	ErrFeeNotEnough
)

const (
	ErrLogicError = 5000 + iota
)

var bcErrors = []BcError{
	{CodeOK, ""},

	{ErrInternalFailed, "Internal failed"},
	{ErrDealFailed, "Deal failed"},
	{ErrPath, "Invalid url path"},

	{ErrCheckTx, "CheckTx failed"},

	//ErrCodeNoAuthorization
	{ErrNoAuthorization, "No authorization"},

	{ErrDeliverTx, "DeliverTx failed"},
	{ErrMarshal, "Json marshal error"},
	{ErrCallRPC, "Call rpc error"},
	{ErrDealFailed, "The deal failed"},
	{ErrAccountLocked, "Account is locked"},

	{ErrCheckTx, "CheckTx failed"},

	{ErrDeliverTx, "DeliverTx failed"},

	// Err describe of EVM
	{ErrCodeEVMInvoke, "EVM invokeTx failed"},
	{ErrRlpDecode, "EVM rlp decode failed"},
	{ErrCallState, "EVM of error calling state"},
	{ErrFeeNotEnough, "EVM Insufficient balance to pay fee"},
}

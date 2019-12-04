package types

//Error define type for error of sdk
type Error struct {
	ErrorCode uint32 // Error code
	ErrorDesc string // Error description
}

// Error() gets error description with error code
func (err *Error) Error() string {
	if err.ErrorDesc != "" {
		return err.ErrorDesc
	}

	errStr, found := errStrings[err.ErrorCode]
	if found {
		return errStr
	}
	return "Undefined error description"
}

// For smart contracts custom errors
const (
	CodeOK = 200 + iota
)
const (
	// minimum of stub code
	ErrStubDefined = 51000 + iota
	ErrGasNotEnough
	ErrFeeNotEnough
)
const (
	//standard code for sdk
	ErrAddSupplyNotEnabled = 52000 + iota
	ErrBurnNotEnabled
	ErrInvalidAddress
)
const (
	//standard code for contract
	ErrNoAuthorization = 53000 + iota
	ErrInvalidParameter
	ErrInsufficientBalance
	ErrInvalidMethod
)
const (
	// minimum of user code
	ErrUserDefined = 55000 + iota
	ErrExpireContract
)

var errStrings map[uint32]string

func init() {
	errStrings = make(map[uint32]string)

	errStrings[CodeOK] = ""

	//for stub
	errStrings[ErrStubDefined] = "Error stub defined"
	errStrings[ErrGasNotEnough] = "Gas limit is not enough"
	errStrings[ErrFeeNotEnough] = "Insufficient balance to pay fee"

	//for sdk
	errStrings[ErrAddSupplyNotEnabled] = "Add supply is not enabled"
	errStrings[ErrBurnNotEnabled] = "Burn supply is not enabled"
	errStrings[ErrInvalidAddress] = "Invalid address"

	//for contract
	errStrings[ErrNoAuthorization] = "No authorization to execute contract"
	errStrings[ErrInvalidParameter] = "Invalid parameter"
	errStrings[ErrInsufficientBalance] = "Insufficient balance"
	errStrings[ErrInvalidMethod] = "Invalid method"
	errStrings[ErrUserDefined] = "Error user defined"

	// user code
	errStrings[ErrExpireContract] = "The contract has expired"
}

package std

// GetResult - get callback function return result struct
type GetResult struct {
	Code int32  `json:"code"` // result code, types.CodeOk - success
	Msg  string `json:"log"`  // result message
	Data []byte `json:"data"` // result data
}

package parsecode

// GoLang can't declare const array, so ... var

// WhiteListPKG - these packages can be imported
var WhiteListPKG = map[string]struct{}{
	// SYS packages
	"\"bytes\"":             {},
	"\"container/heap\"":    {},
	"\"container/list\"":    {},
	"\"container/ring\"":    {},
	"\"crypto\"":            {},
	"\"crypto/aes\"":        {},
	"\"crypto/cipher\"":     {},
	"\"crypto/des\"":        {},
	"\"crypto/dsa\"":        {},
	"\"crypto/ecdsa\"":      {},
	"\"crypto/elliptic\"":   {},
	"\"crypto/hmac\"":       {},
	"\"crypto/md5\"":        {},
	"\"crypto/rc4\"":        {},
	"\"crypto/rsa\"":        {},
	"\"crypto/sha1\"":       {},
	"\"crypto/sha256\"":     {},
	"\"crypto/sha512\"":     {},
	"\"encoding\"":          {},
	"\"encoding/ascii85\"":  {},
	"\"encoding/asn1\"":     {},
	"\"encoding/base32\"":   {},
	"\"encoding/base64\"":   {},
	"\"encoding/binary\"":   {},
	"\"encoding/csv\"":      {},
	"\"encoding/gob\"":      {},
	"\"encoding/hex\"":      {},
	"\"encoding/json\"":     {},
	"\"encoding/pem\"":      {},
	"\"encoding/xml\"":      {},
	"\"errors\"":            {},
	"\"fmt\"":               {},
	"\"hash\"":              {},
	"\"hash/adler32\"":      {},
	"\"hash/crc32\"":        {},
	"\"hash/crc64\"":        {},
	"\"hash/fnv\"":          {},
	"\"index/suffixarray\"": {},
	"\"math\"":              {},
	"\"math/big\"":          {},
	"\"math/bits\"":         {},
	"\"math/cmplx\"":        {},
	"\"reflect\"":           {},
	"\"regexp\"":            {},
	"\"regexp/syntax\"":     {},
	"\"sort\"":              {},
	"\"strconv\"":           {},
	"\"strings\"":           {},
	"\"unicode\"":           {},
	"\"unicode/utf8\"":      {},
	"\"unicode/utf16\"":     {},

	// SDK packages
}

// WhiteListPkgPrefix - packages start with these paths are all allowed
var WhiteListPkgPrefix = []string{
	"\"blockchain/smcsdk/sdk",
	"\"github.com/bcbchain/sdk/sdk",
}

const (
	contractNameExpr  = `^[a-zA-Z]+[a-zA-Z0-9_\-\.]*$`
	organizationExpr  = "^org[1-9a-km-zA-HJ-NP-Z]*$"
	authorExpr        = "^[0-9a-fA-f]{64}$"
	versionExpr       = `^\d+(\.\d+){0,3}$`
	contractClassExpr = "^[A-Z][a-zA-Z0-9_]*$"
)

// LiteralTypes - primitive type
var LiteralTypes = map[string]struct{}{
	"int":      {},
	"int8":     {},
	"int16":    {},
	"int32":    {},
	"int64":    {},
	"uint":     {},
	"uint8":    {},
	"uint16":   {},
	"uint32":   {},
	"uint64":   {},
	"float32":  {},
	"float64":  {},
	"bool":     {},
	"string":   {},
	"byte":     {},
	"Address":  {},
	"Hash":     {},
	"HexBytes": {},
	"PubKey":   {},
}

var NumberType = "Number"

var baseTypes = map[string]struct{}{
	"int":      {},
	"int8":     {},
	"int16":    {},
	"int32":    {},
	"int64":    {},
	"uint":     {},
	"uint8":    {},
	"uint16":   {},
	"uint32":   {},
	"uint64":   {},
	"bool":     {},
	"Address":  {},
	"Number":   {},
	"HexBytes": {},
	"Hash":     {},
	"PubKey":   {},
	"string":   {},
	"byte":     {}}

var receipts = []string{
	"std.Transfer",
	"std.SetOwner",
	"std.Fee",
	"std.SetGasPrice",
	"std.Burn",
	"std.AddSupply",
	"std.NewToken",
	"ibc.Packet",
	"ibc.State",
	"ibc.Final"}

var basicContracts = map[string]struct{}{
	"token-basic":       {},
	"token-issue":       {},
	"organization":      {},
	"smartcontract":     {},
	"mining":            {},
	"ibc":               {},
	"netgovernance":     {},
	"governance":        {},
	"brc30-token-issue": {},
	"black-list":        {}}

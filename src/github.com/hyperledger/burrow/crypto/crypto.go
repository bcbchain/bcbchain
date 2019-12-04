package crypto

type CurveType uint32

const (
	CurveTypeUnset CurveType = iota
	CurveTypeEd25519
	CurveTypeSecp256k1
)

func (k CurveType) String() string {
	switch k {
	case CurveTypeSecp256k1:
		return "secp256k1"
	case CurveTypeEd25519:
		return "ed25519"
	case CurveTypeUnset:
		return ""
	default:
		return "unknown"
	}
}
func (k CurveType) ABCIType() string {
	switch k {
	case CurveTypeSecp256k1:
		return "secp256k1"
	case CurveTypeEd25519:
		return "ed25519"
	case CurveTypeUnset:
		return ""
	default:
		return "unknown"
	}
}

// Get this CurveType's 8 bit identifier as a byte
func (k CurveType) Byte() byte {
	return byte(k)
}

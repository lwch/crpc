package codec

// DataType data type
type DataType byte

const (
	// TypeUnknown unknown type
	TypeUnknown DataType = iota
	// TypeRaw raw data
	TypeRaw
	// TypeHTTPRequest http request data
	TypeHTTPRequest
	// TypeHTTPResponse http response data
	TypeHTTPResponse
)

type header struct {
	Type DataType
}

func (h *header) Marshal() []byte {
	return []byte{byte(h.Type)}
}

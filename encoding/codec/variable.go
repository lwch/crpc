package codec

import (
	"bufio"
	"bytes"
	"net/http"
)

// Variable variable data
type Variable struct {
	t         DataType
	raw       bytes.Buffer
	bufReader *bufio.Reader
}

// Type data type
func (v *Variable) Type() DataType {
	return v.t
}

// Bytes get raw bytes
func (v *Variable) Bytes() []byte {
	return v.raw.Bytes()
}

// ToRequest convert to http request
func (v *Variable) ToRequest() (*http.Request, error) {
	v.bufReader.Reset(&v.raw)
	return http.ReadRequest(v.bufReader)
}

// ToResponse convert to http response
func (v *Variable) ToResponse() (*http.Response, error) {
	v.bufReader.Reset(&v.raw)
	return http.ReadResponse(v.bufReader, nil)
}

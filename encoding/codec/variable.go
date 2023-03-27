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

// NewVariable create variable
func NewVariable() *Variable {
	v := &Variable{}
	v.bufReader = bufio.NewReader(&v.raw)
	return v
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
	return http.ReadRequest(v.bufReader)
}

// ToResponse convert to http response
func (v *Variable) ToResponse() (*http.Response, error) {
	return http.ReadResponse(v.bufReader, nil)
}

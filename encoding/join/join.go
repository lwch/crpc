package join

import "bytes"

// Marshaler marshaler
type Marshaler interface {
	Marshal() []byte
}

// Joiner joiner
type Joiner struct {
	buf     bytes.Buffer
	header  Marshaler
	payload Marshaler
}

// New create joiner
func New() *Joiner {
	return &Joiner{}
}

// SetHeader set header
func (j *Joiner) SetHeader(header Marshaler) {
	j.header = header
}

// SetPayload set payload
func (j *Joiner) SetPayload(body Marshaler) {
	j.payload = body
}

// Marshal marshal
func (j *Joiner) Marshal() ([]byte, error) {
	j.buf.Reset()
	if _, err := j.buf.Write(j.header.Marshal()); err != nil {
		return nil, err
	}
	if _, err := j.buf.Write(j.payload.Marshal()); err != nil {
		return nil, err
	}
	cpy := make([]byte, j.buf.Len())
	copy(cpy, j.buf.Bytes())
	return cpy, nil
}

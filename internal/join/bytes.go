package join

import "bytes"

// BytesBuffer bytes buffer
type BytesBuffer struct {
	bytes.Buffer
}

// Marshal marshal bytes
func (b *BytesBuffer) Marshal() []byte {
	return b.Bytes()
}

// Bytes bytes
type Bytes []byte

// Marshal marshal bytes
func (b Bytes) Marshal() []byte {
	return b
}

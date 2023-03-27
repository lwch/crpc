package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"sync"

	"github.com/lwch/crpc/encoding"
	"github.com/lwch/crpc/internal/join"
)

var errUnsupportedType = errors.New("codec: unsupported type")
var errInvalidOutputValueType = errors.New("codec: output value type is not codec.Variable")

// Codec serializer
type Codec struct {
	bufPool  sync.Pool
	joinPool sync.Pool
}

// New create codec
func New() encoding.Codec {
	c := &Codec{}
	c.bufPool.New = func() any {
		return new(join.BytesBuffer)
	}
	c.joinPool.New = func() any {
		return join.New()
	}
	return c
}

// Marshal serialize data
func (c *Codec) Marshal(v any) ([]byte, error) {
	type writer interface {
		Write(io.Writer) error
	}
	var hdr header
	payload := c.bufPool.Get().(*join.BytesBuffer)
	defer c.bufPool.Put(payload)
	payload.Reset()
	switch value := v.(type) {
	case http.Request, *http.Request:
		hdr.Type = TypeHTTPRequest
		if err := v.(writer).Write(payload); err != nil {
			return nil, err
		}
	case http.Response, *http.Response:
		hdr.Type = TypeHTTPResponse
		if err := v.(writer).Write(payload); err != nil {
			return nil, err
		}
	case []byte:
		hdr.Type = TypeRaw
		if _, err := io.Copy(payload, bytes.NewReader(value)); err != nil {
			return nil, err
		}
	default:
		return nil, errUnsupportedType
	}
	joiner := c.joinPool.Get().(*join.Joiner)
	defer c.joinPool.Put(joiner)
	joiner.SetHeader(&hdr)
	joiner.SetPayload(payload)
	return joiner.Marshal()
}

// Unmarshal deserialize data
func (c *Codec) Unmarshal(data []byte, v any) error {
	vb, ok := v.(*Variable)
	if !ok {
		return errInvalidOutputValueType
	}
	r := bytes.NewReader(data)
	var hdr header
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return err
	}
	_, err = io.Copy(&vb.raw, r)
	if err != nil {
		return err
	}
	vb.t = hdr.Type
	return nil
}

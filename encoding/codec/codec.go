package codec

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"net/http"
	"reflect"
	"sync"

	"github.com/lwch/crpc/encoding"
	"github.com/lwch/crpc/internal/join"
)

var errUnsupportedType = errors.New("codec: unsupported type")
var errIsNotPointer = errors.New("codec: the specify variable is not pointer")

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
func (c *Codec) Unmarshal(data []byte, v any) (int, error) {
	vv := reflect.ValueOf(v)
	if vv.Kind() != reflect.Ptr {
		return 0, errIsNotPointer
	}
	r := bytes.NewReader(data)
	var hdr header
	err := binary.Read(r, binary.BigEndian, &hdr)
	if err != nil {
		return 0, err
	}
	switch hdr.Type {
	case TypeRaw:
		if vv, ok := v.(*[]byte); ok {
			return copy(*vv, data[1:]), nil
		}
		buf := make([]byte, len(data)-1)
		n := copy(buf, data[1:])
		vv.Elem().Set(reflect.ValueOf(buf))
		return n, nil
	case TypeHTTPRequest:
		if req, ok := v.(*http.Request); ok {
			v, err := http.ReadRequest(bufio.NewReader(r))
			if err != nil {
				return 0, err
			}
			*req = *v
			return 0, nil
		}
		req, err := http.ReadRequest(bufio.NewReader(r))
		if err != nil {
			return 0, err
		}
		vv.Elem().Set(reflect.ValueOf(req))
		return 0, nil
	case TypeHTTPResponse:
		if rep, ok := v.(*http.Response); ok {
			v, err := http.ReadResponse(bufio.NewReader(r), nil)
			if err != nil {
				return 0, err
			}
			*rep = *v
			return 0, nil
		}
		rep, err := http.ReadResponse(bufio.NewReader(r), nil)
		if err != nil {
			return 0, err
		}
		vv.Elem().Set(reflect.ValueOf(rep))
		return 0, nil
	}
	return 0, errUnsupportedType
}

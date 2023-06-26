package codec

import (
	"bytes"
	"encoding/binary"
	"errors"
	"net/http"
	"reflect"
	"sync"

	"github.com/lwch/crpc/encoding"
	"github.com/lwch/crpc/internal/join"
	"google.golang.org/protobuf/proto"
)

var errUnsupportedType = errors.New("codec: unsupported type")
var errIsNotPointer = errors.New("codec: the specify variable is not pointer")
var errProtoMessage = errors.New("codec: the specify variable is not proto.Message")

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
	switch value := v.(type) {
	case []byte:
		return c.marshalRaw(value)
	case http.Request, *http.Request:
		return c.marshalHTTPRequest(value)
	case http.Response, *http.Response:
		return c.marshalHTTPResponse(value)
	case proto.Message:
		return c.marshalProtoMessage(value)
	default:
		return nil, errUnsupportedType
	}
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
		return c.unmarshalRaw(r, v, len(data)-1)
	case TypeHTTPRequest:
		return c.unmarshalHttpRequest(r, v)
	case TypeHTTPResponse:
		return c.unmarshalHttpResponse(r, v)
	case TypeProtobuf:
		return c.unmarshalProtoMessage(r, v)
	default:
		return 0, errUnsupportedType
	}
}

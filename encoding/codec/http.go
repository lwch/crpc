package codec

import (
	"bufio"
	"io"
	"net/http"
	"reflect"

	"github.com/lwch/crpc/internal/join"
)

func (c *Codec) marshalHttpRequest(v any) ([]byte, error) {
	type writer interface {
		Write(io.Writer) error
	}
	var hdr header
	payload := c.bufPool.Get().(*join.BytesBuffer)
	defer c.bufPool.Put(payload)
	payload.Reset()
	hdr.Type = TypeHTTPRequest
	if err := v.(writer).Write(payload); err != nil {
		return nil, err
	}
	joiner := c.joinPool.Get().(*join.Joiner)
	defer c.joinPool.Put(joiner)
	joiner.SetHeader(&hdr)
	joiner.SetPayload(payload)
	return joiner.Marshal()
}

func (c *Codec) marshalHttpResponse(v any) ([]byte, error) {
	type writer interface {
		Write(io.Writer) error
	}
	var hdr header
	payload := c.bufPool.Get().(*join.BytesBuffer)
	defer c.bufPool.Put(payload)
	payload.Reset()
	hdr.Type = TypeHTTPResponse
	if err := v.(writer).Write(payload); err != nil {
		return nil, err
	}
	joiner := c.joinPool.Get().(*join.Joiner)
	defer c.joinPool.Put(joiner)
	joiner.SetHeader(&hdr)
	joiner.SetPayload(payload)
	return joiner.Marshal()
}

func (c *Codec) unmarshalHttpRequest(r io.Reader, v any) (int, error) {
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
	reflect.ValueOf(v).Elem().Set(reflect.ValueOf(req))
	return 0, nil
}

func (c *Codec) unmarshalHttpResponse(r io.Reader, v any) (int, error) {
	if resp, ok := v.(*http.Response); ok {
		v, err := http.ReadResponse(bufio.NewReader(r), nil)
		if err != nil {
			return 0, err
		}
		*resp = *v
		return 0, nil
	}
	resp, err := http.ReadResponse(bufio.NewReader(r), nil)
	if err != nil {
		return 0, err
	}
	reflect.ValueOf(v).Elem().Set(reflect.ValueOf(resp))
	return 0, nil
}

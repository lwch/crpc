package codec

import (
	"bytes"
	"io"
	"reflect"

	"github.com/lwch/crpc/internal/join"
	"github.com/lwch/crpc/internal/utils"
)

func (c *Codec) marshalRaw(v any) ([]byte, error) {
	var hdr header
	payload := c.bufPool.Get().(*join.BytesBuffer)
	defer c.bufPool.Put(payload)
	payload.Reset()
	hdr.Type = TypeRaw
	if _, err := io.Copy(payload, bytes.NewReader(v.([]byte))); err != nil {
		return nil, err
	}
	joiner := c.joinPool.Get().(*join.Joiner)
	defer c.joinPool.Put(joiner)
	joiner.SetHeader(&hdr)
	joiner.SetPayload(payload)
	return joiner.Marshal()
}

func (*Codec) unmarshalRaw(r io.Reader, v any, len int) (int, error) {
	if vv, ok := v.(*[]byte); ok {
		if cap(*vv) == 0 {
			*vv = make([]byte, len)
		}
		n, err := io.Copy(utils.BytesWriter(*vv), r)
		return int(n), err
	}
	var buf bytes.Buffer
	n, err := io.Copy(&buf, r)
	if err != nil {
		return 0, err
	}
	reflect.ValueOf(v).Elem().Set(reflect.ValueOf(buf.Bytes()))
	return int(n), nil
}

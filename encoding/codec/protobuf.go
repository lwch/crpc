package codec

import (
	"bytes"
	"io"

	"github.com/lwch/crpc/internal/join"
	"google.golang.org/protobuf/proto"
)

func (c *Codec) marshalProtoMessage(v any) ([]byte, error) {
	var hdr header
	payload := c.bufPool.Get().(*join.BytesBuffer)
	defer c.bufPool.Put(payload)
	payload.Reset()
	hdr.Type = TypeProtobuf
	enc, err := proto.Marshal(v.(proto.Message))
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(payload, bytes.NewReader(enc))
	if err != nil {
		return nil, err
	}
	joiner := c.joinPool.Get().(*join.Joiner)
	defer c.joinPool.Put(joiner)
	joiner.SetHeader(&hdr)
	joiner.SetPayload(payload)
	return joiner.Marshal()
}

func (c *Codec) unmarshalProtoMessage(r io.Reader, v any) (int, error) {
	msg, ok := v.(proto.Message)
	if !ok {
		return 0, errProtoMessage
	}
	data, err := io.ReadAll(r)
	if err != nil {
		return 0, err
	}
	if err := proto.Unmarshal(data, msg); err != nil {
		return 0, err
	}
	return 0, nil
}

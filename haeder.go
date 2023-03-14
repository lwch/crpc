package crpc

import (
	"fmt"
	"net/http"

	"github.com/lwch/crpc/encoding/codec"
)

const keyRequestID = "X-Request-Id"

func (tp *transport) buildRequest(req *http.Request) ([]byte, uint64, error) {
	if req.Header == nil {
		req.Header = make(http.Header)
	}
	seq := tp.sequence.Add(1)
	req.Header.Set(keyRequestID, fmt.Sprintf("%d", seq))
	payload, err := tp.codec.Marshal(req)
	if err != nil {
		return nil, 0, err
	}
	if tp.encoder != nil {
		payload = tp.encoder.Encrypt(payload)
	}
	return payload, seq, nil
}

func (tp *transport) buildResponse(rep *http.Response, reqID uint64) ([]byte, error) {
	if rep.Header == nil {
		rep.Header = make(http.Header)
	}
	rep.Header.Set(keyRequestID, fmt.Sprintf("%d", reqID))
	payload, err := tp.codec.Marshal(rep)
	if err != nil {
		return nil, err
	}
	if tp.encoder != nil {
		payload = tp.encoder.Encrypt(payload)
	}
	return payload, nil
}

func (tp *transport) decode(data []byte) (*codec.Variable, error) {
	if tp.encoder != nil {
		var err error
		data, err = tp.encoder.Decrypt(data)
		if err != nil {
			return nil, err
		}
	}
	var value codec.Variable
	err := tp.codec.Unmarshal(data, &value)
	if err != nil {
		return nil, err
	}
	return &value, nil
}

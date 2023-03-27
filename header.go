package crpc

import (
	"fmt"
	"net/http"

	"github.com/lwch/crpc/encoding/codec"
)

const keyRequestID = "X-Crpc-Request-Id"

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
	if tp.compresser != nil {
		payload, err = tp.compresser.Compress(payload)
		if err != nil {
			return nil, 0, err
		}
	}
	if tp.encrypter != nil {
		payload, err = tp.encrypter.Encrypt(payload)
		if err != nil {
			return nil, 0, err
		}
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
	if tp.compresser != nil {
		payload, err = tp.compresser.Compress(payload)
		if err != nil {
			return nil, err
		}
	}
	if tp.encrypter != nil {
		payload, err = tp.encrypter.Encrypt(payload)
		if err != nil {
			return nil, err
		}
	}
	return payload, nil
}

func (tp *transport) decode(data []byte, value *codec.Variable) error {
	if tp.encrypter != nil {
		var err error
		data, err = tp.encrypter.Decrypt(data)
		if err != nil {
			return err
		}
	}
	if tp.compresser != nil {
		var err error
		data, err = tp.compresser.Decompress(data)
		if err != nil {
			return err
		}
	}
	err := tp.codec.Unmarshal(data, value)
	if err != nil {
		return err
	}
	return nil
}

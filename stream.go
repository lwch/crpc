package crpc

import (
	"github.com/lwch/crpc/encoding/codec"
	"github.com/lwch/crpc/network"
)

// Stream stream
type Stream struct {
	parent *transport
	s      *network.Stream
}

// Close close stream
func (s *Stream) Close() error {
	return s.s.Close()
}

// Write write data in stream
func (s *Stream) Write(p []byte) (int, error) {
	data, err := s.parent.codec.Marshal(p)
	if err != nil {
		return 0, err
	}
	if s.parent.compresser != nil {
		data, err = s.parent.compresser.Compress(data)
		if err != nil {
			return 0, err
		}
	}
	if s.parent.encrypter != nil {
		data, err = s.parent.encrypter.Encrypt(data)
		if err != nil {
			return 0, err
		}
	}
	_, err = s.s.Write(data)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

// Read read data from stream
func (s *Stream) Read(p []byte) (int, error) {
	buf := make([]byte, 65535)
	n, err := s.s.Read(buf)
	if err != nil {
		return 0, err
	}
	if n == 0 {
		return 0, nil
	}
	buf = buf[:n]
	if s.parent.encrypter != nil {
		buf, err = s.parent.encrypter.Decrypt(buf)
		if err != nil {
			return 0, err
		}
	}
	if s.parent.compresser != nil {
		buf, err = s.parent.compresser.Decompress(buf)
		if err != nil {
			return 0, err
		}
	}
	var vb codec.Variable
	err = s.parent.codec.Unmarshal(buf, &vb)
	if err != nil {
		return 0, err
	}
	return copy(p, vb.Bytes()), nil
}

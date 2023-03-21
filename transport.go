package crpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lwch/crpc/encoding"
	"github.com/lwch/crpc/encoding/codec"
	"github.com/lwch/crpc/network"
	"github.com/lwch/logging"
)

// ErrTimeout timeout error
var ErrTimeout = errors.New("transport: timeout")

// RequestHandlerFunc request handler
type RequestHandlerFunc func(*http.Request) (*http.Response, error)

type transport struct {
	conn       *network.Conn
	codec      encoding.Codec
	encrypter  encoding.Encrypter
	compresser encoding.Compresser
	sequence   atomic.Uint64
	onResponse map[uint64]chan *http.Response
	mResponse  sync.RWMutex
	onRequest  RequestHandlerFunc
	// runtime
	err    error
	ctx    context.Context
	cancel context.CancelFunc
}

func new(conn net.Conn) *transport {
	conn.SetDeadline(time.Time{}) // no timeout
	ctx, cancel := context.WithCancel(context.Background())
	t := &transport{
		conn:       network.New(conn),
		codec:      codec.New(),
		onResponse: make(map[uint64]chan *http.Response),
		onRequest: func(r *http.Request) (*http.Response, error) {
			return &http.Response{}, nil
		},
		ctx:    ctx,
		cancel: cancel,
	}
	go t.keepalive()
	return t
}

func (tp *transport) SetEncrypter(encrypter encoding.Encrypter) {
	tp.encrypter = encrypter
}

func (tp *transport) SetCompresser(compresser encoding.Compresser) {
	tp.compresser = compresser
}

func (tp *transport) AcceptStream() (*Stream, error) {
	s, err := tp.conn.AcceptStream()
	if err != nil {
		return nil, err
	}
	return &Stream{
		parent: tp,
		s:      s,
	}, nil
}

func (tp *transport) OpenStream(timeout time.Duration) (*Stream, error) {
	s, err := tp.conn.OpenStream(timeout)
	if err != nil {
		return nil, err
	}
	return &Stream{
		parent: tp,
		s:      s,
	}, nil
}

func (tp *transport) SetOnRequest(fn RequestHandlerFunc) {
	tp.onRequest = fn
}

func (tp *transport) Close() error {
	return tp.conn.Close()
}

func (tp *transport) Call(req *http.Request, timeout time.Duration) (*http.Response, error) {
	data, reqID, err := tp.buildRequest(req)
	if err != nil {
		return nil, err
	}
	ch := make(chan *http.Response, 1)
	tp.mResponse.Lock()
	tp.onResponse[reqID] = ch
	tp.mResponse.Unlock()
	defer func() {
		close(ch)
		tp.mResponse.Lock()
		defer tp.mResponse.Unlock()
		delete(tp.onResponse, reqID)
	}()
	_, err = tp.conn.Write(data)
	if err != nil {
		return nil, err
	}
	select {
	case <-tp.ctx.Done():
		return nil, tp.err
	case <-time.After(timeout):
		return nil, ErrTimeout
	case resp := <-ch:
		return resp, nil
	}
}

func (tp *transport) Serve() error {
	var err error
	defer func() {
		tp.err = err
		tp.cancel()
	}()
	buf := make([]byte, 65535)
	for {
		var n int
		n, err = tp.conn.Read(buf)
		if err != nil {
			logging.Error("serve: %v", err)
			return err
		}
		payload, err := tp.decode(buf[:n])
		if err != nil {
			logging.Error("decode: %v", err)
			return err
		}
		switch payload.Type() {
		case codec.TypeHTTPRequest:
			req, err := payload.ToRequest()
			if err != nil {
				logging.Error("to request: %v", err)
				return err
			}
			str := req.Header.Get(keyRequestID)
			seq, _ := strconv.ParseUint(str, 10, 64)
			go tp.handleRequest(req, seq)
		case codec.TypeHTTPResponse:
			rep, err := payload.ToResponse()
			if err != nil {
				logging.Error("to response: %v", err)
				return err
			}
			str := rep.Header.Get(keyRequestID)
			seq, _ := strconv.ParseUint(str, 10, 64)
			tp.mResponse.RLock()
			ch := tp.onResponse[seq]
			tp.mResponse.RUnlock()
			if ch == nil {
				continue
			}
			// recover on closed
			send := func(ch chan *http.Response, rep *http.Response) {
				defer func() {
					recover()
				}()
				ch <- rep
			}
			send(ch, rep)
		}
	}
}

func (tp *transport) handleRequest(req *http.Request, reqID uint64) {
	if tp.onRequest == nil {
		return
	}
	resp, err := tp.onRequest(req)
	if err != nil {
		resp = &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader(err.Error())),
		}
	}
	resp.ProtoMajor = req.ProtoMajor
	resp.ProtoMinor = req.ProtoMinor
	if resp.Body != nil {
		var buf bytes.Buffer
		if _, err := io.Copy(&buf, resp.Body); err != nil {
			logging.Error("read body(%d): %v", reqID, err)
			return
		}
		resp.ContentLength = int64(buf.Len())
		resp.Body = io.NopCloser(&buf)
	}
	data, err := tp.buildResponse(resp, reqID)
	if err != nil {
		logging.Error("build response(%d): %v", reqID, err)
		return
	}
	_, err = tp.conn.Write(data)
	if err != nil {
		logging.Error("write response(%d): %v", reqID, err)
		return
	}
}

func (tp *transport) keepalive() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-tp.ctx.Done():
			return
		case <-ticker.C:
			err := tp.conn.SendKeepalive()
			if err != nil {
				logging.Error("keepalive: %v", err)
				continue
			}
		}
	}
}

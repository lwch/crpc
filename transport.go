package crpc

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
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

// ErrDone done error
var ErrDone = errors.New("transport: done")

var errDataType = errors.New("transport: data type error")

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

func (tp *transport) OpenStream(ctx context.Context) (*Stream, error) {
	s, err := tp.conn.OpenStream(ctx)
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

func (tp *transport) Call(ctx context.Context, req *http.Request) (*http.Response, error) {
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
	hdr, _ := httputil.DumpRequest(req, false)
	logging.Debug("< http call(%d):\n%s", reqID, string(hdr))
	_, err = tp.conn.Write(data)
	if err != nil {
		return nil, err
	}
	select {
	case <-tp.ctx.Done():
		return nil, tp.err
	case <-ctx.Done():
		return nil, ErrDone
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
		err := tp.parse(buf[:n])
		if err != nil {
			logging.Error("parse: %v", err)
			return err
		}
	}
}

func (tp *transport) parse(data []byte) error {
	payload, err := tp.decode(data)
	if err != nil {
		logging.Error("decode: %v", err)
		return err
	}
	switch v := payload.(type) {
	case *http.Request:
		str := v.Header.Get(keyRequestID)
		seq, _ := strconv.ParseUint(str, 10, 64)
		go tp.handleRequest(v, seq)
	case *http.Response:
		str := v.Header.Get(keyRequestID)
		seq, _ := strconv.ParseUint(str, 10, 64)
		tp.mResponse.RLock()
		ch := tp.onResponse[seq]
		tp.mResponse.RUnlock()
		if ch == nil {
			return nil
		}
		// recover on closed
		send := func(ch chan *http.Response, rep *http.Response) {
			defer func() {
				recover()
			}()
			ch <- rep
		}
		send(ch, v)
	default:
		return errDataType
	}
	return nil
}

func (tp *transport) handleRequest(req *http.Request, reqID uint64) {
	hdr, _ := httputil.DumpRequest(req, false)
	logging.Debug("> received http call(%d):\n%s", reqID, string(hdr))
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
	hdr, _ = httputil.DumpResponse(resp, false)
	logging.Debug("< http response(%d):\n%s", reqID, string(hdr))
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

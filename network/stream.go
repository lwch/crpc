package network

import (
	"context"
	"errors"
	"math"
	"sync/atomic"
)

// ErrStreamClosed stream closed error
var ErrStreamClosed = errors.New("network: stream closed")

// ErrClosedByRemote closed by remote error
var ErrClosedByRemote = errors.New("network: closed by remote")

// Stream stream
type Stream struct {
	parent *Conn
	id     uint32
	closed atomic.Bool
	chRead chan []byte
	// runtime
	err    error
	ctx    context.Context
	cancel context.CancelFunc
}

func newStream(parent *Conn, id uint32) *Stream {
	ctx, cancel := context.WithCancel(context.Background())
	return &Stream{
		parent: parent,
		id:     id & 0xffffff,
		chRead: make(chan []byte, 1000),
		ctx:    ctx,
		cancel: cancel,
	}
}

// ID get stream id
func (s *Stream) ID() uint32 {
	return s.id & 0xffffff
}

// Close close stream
func (s *Stream) Close() error {
	s.onClose(nil)
	return nil
}

func (s *Stream) onClose(err error) {
	s.closed.Store(true)
	s.err = err
	s.cancel()
	s.parent.chWriteControl <- writeControlArgs{
		id:   s.ID(),
		flag: flagStreamClose,
	}
	s.parent.mStreams.Lock()
	delete(s.parent.streams, s.ID())
	s.parent.mStreams.Unlock()
}

// Read read data
func (s *Stream) Read(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, ErrStreamClosed
	}
	select {
	case data := <-s.chRead:
		if len(data) > len(p) {
			return 0, errBufferTooShort
		}
		return copy(p, data), nil
	case <-s.ctx.Done():
		return 0, s.err
	}
}

// Write write data
func (s *Stream) Write(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, ErrStreamClosed
	}
	if len(p) > math.MaxUint16 {
		return 0, errTooLarge
	}
	data := make([]byte, len(p))
	copy(data, p)
	s.parent.chWrite <- writeArgs{
		flag: s.ID() | flagStreamData,
		data: data,
	}
	return len(p), nil
}

func (c *Conn) handleOpenStream() error {
	stream := newStream(c, c.streamID.Add(1))
	c.chWriteControl <- writeControlArgs{
		id:   stream.id,
		flag: flagStreamOpenAck,
	}
	c.mStreams.Lock()
	c.streams[stream.id] = stream
	c.mStreams.Unlock()
	c.chStreamOpened <- stream
	return nil
}

func (c *Conn) handleOpenStreamAck(flag uint32) error {
	s := newStream(c, flag)
	c.mStreams.Lock()
	c.streams[s.id] = s
	c.mStreams.Unlock()
	c.chStreamOpened <- s
	return nil
}

func (c *Conn) handleCloseStream(flag uint32) error {
	s := c.getStream(flag)
	if s == nil {
		return errStreamNotFound
	}
	s.onClose(ErrClosedByRemote)
	return nil
}

func (c *Conn) handleStreamData(flag uint32, data []byte) error {
	s := c.getStream(flag)
	if s == nil {
		return errStreamNotFound
	}
	s.chRead <- dup(data)
	return nil
}

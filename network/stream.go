package network

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"hash/crc32"
	"io"
	"math"
	"sync/atomic"
)

var errStreamClosed = errors.New("network: stream closed")
var errClosedByRemote = errors.New("network: closed by remote")

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
	sequence := s.parent.sequence.Add(1)
	binary.Write(s.parent.conn, binary.BigEndian, header{
		Size:     0,
		Crc32:    0,
		Sequence: sequence,
		Flag:     s.ID() | flagStreamClose,
	})
	s.parent.mStreams.Lock()
	delete(s.parent.streams, s.ID())
	s.parent.mStreams.Unlock()
}

// Read read data
func (s *Stream) Read(p []byte) (int, error) {
	if s.closed.Load() {
		return 0, errStreamClosed
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
		return 0, errStreamClosed
	}
	if len(p) > math.MaxUint16 {
		return 0, errTooLarge
	}
	var buf bytes.Buffer
	sequence := s.parent.sequence.Add(1)
	err := binary.Write(&buf, binary.BigEndian, header{
		Size:     uint16(len(p)),
		Crc32:    crc32.ChecksumIEEE(p),
		Sequence: sequence,
		Flag:     s.ID() | flagStreamData,
	})
	if err != nil {
		return 0, fmt.Errorf("network: build packet header[%d]: %v", sequence, err)
	}
	_, err = io.Copy(&buf, bytes.NewReader(p))
	if err != nil {
		return 0, fmt.Errorf("network: build packet payload[%d]: %v", sequence, err)
	}
	_, err = s.parent.conn.Write(buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("network: write packet[%d]: %v", sequence, err)
	}
	return len(p), nil
}

func (c *Conn) handleOpenStream() error {
	stream := newStream(c, c.streamID.Add(1))
	sequence := c.sequence.Add(1)
	err := binary.Write(c.conn, binary.BigEndian, header{
		Size:     0,
		Crc32:    0,
		Sequence: sequence,
		Flag:     stream.id | flagStreamOpenAck,
	})
	if err != nil {
		return err
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
	s.onClose(errClosedByRemote)
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

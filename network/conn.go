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
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/lwch/logging"
)

var errTooLarge = errors.New("network: too large")
var errBufferTooShort = errors.New("network: buffer too short")
var errInvalidPacketChecksum = errors.New("network: invalid packet checksum")
var errOpenStreamTimeout = errors.New("network: open stream timeout")
var errStreamNotFound = errors.New("network: stream not found")

const (
	flagStreamOpen    = 1 << 31 // 32位表示open请求
	flagStreamOpenAck = 1 << 30 // 31位表示open成功
	flagStreamClose   = 1 << 29 // 30位表示关闭请求
	flagStreamData    = 1 << 28 // 29位表示数据传输
	flagPing          = 1 << 27 // 28位表示ping请求
	flagPong          = 1 << 26 // 27位表示pong响应
)

// Conn connection
type Conn struct {
	conn     net.Conn
	mRead    sync.Mutex
	sequence atomic.Uint64
	streamID atomic.Uint32
	chRead   chan []byte
	// stream
	streams        map[uint32]*Stream
	mStreams       sync.RWMutex
	chStreamOpened chan *Stream
	// runtime
	err error
	ctx context.Context
}

// 封包格式
// +-------------+---------+----------+---------+---------+
// | Sequence(4) | Size(2) | Crc32(4) | Flag(4) | Payload |
// +-------------+---------+----------+---------+---------+
// Flag字段格式
// +---------+------------+----------+---------+---------+---------+-----------+---------------+
// | Open(1) | OpenAck(1) | Close(1) | Data(1) | Ping(1) | Pong(1) | Unused(2) | Stream ID(24) |
// +---------+------------+----------+---------+---------+---------+-----------+---------------+
// 高6位为标志位，后2位暂未使用，低24位为stream id
// Stream ID由Accept方进行分配，在Open请求中Stream ID为0

type header struct {
	Sequence uint64
	Size     uint16
	Crc32    uint32
	Flag     uint32
}

// New new connection
func New(conn net.Conn) *Conn {
	ctx, cancel := context.WithCancel(context.Background())
	ret := &Conn{
		conn:           conn,
		chRead:         make(chan []byte, 10000),
		streams:        make(map[uint32]*Stream),
		chStreamOpened: make(chan *Stream),
		ctx:            ctx,
	}
	go ret.loopRead(cancel)
	return ret
}

// Close close connection
func (c *Conn) Close() error {
	logging.Error("connection closed: %s", c.conn.RemoteAddr().String())
	return c.conn.Close()
}

// AcceptStream accept stream
func (c *Conn) AcceptStream() (*Stream, error) {
	select {
	case <-c.ctx.Done():
		return nil, c.err
	case stream := <-c.chStreamOpened:
		return stream, nil
	}
}

// OpenStream open stream
func (c *Conn) OpenStream(timeout time.Duration) (*Stream, error) {
	sequence := c.sequence.Add(1)
	err := binary.Write(c.conn, binary.BigEndian, header{
		Size:     0,
		Crc32:    0,
		Sequence: sequence,
		Flag:     flagStreamOpen,
	})
	if err != nil {
		return nil, fmt.Errorf("network: send openstream: %v", err)
	}
	after := time.After(timeout)
	select {
	case <-after:
		return nil, errOpenStreamTimeout
	case <-c.ctx.Done():
		return nil, c.err
	case stream := <-c.chStreamOpened:
		return stream, nil
	}
}

// Write send data
func (c *Conn) Write(p []byte) (int, error) {
	if len(p) > math.MaxUint16 {
		return 0, errTooLarge
	}
	var buf bytes.Buffer
	sequence := c.sequence.Add(1)
	err := binary.Write(&buf, binary.BigEndian, header{
		Size:     uint16(len(p)),
		Crc32:    crc32.ChecksumIEEE(p),
		Sequence: sequence,
		Flag:     0,
	})
	if err != nil {
		return 0, fmt.Errorf("network: build packet header[%d]: %v", sequence, err)
	}
	_, err = io.Copy(&buf, bytes.NewReader(p))
	if err != nil {
		return 0, fmt.Errorf("network: build packet payload[%d]: %v", sequence, err)
	}
	_, err = c.conn.Write(buf.Bytes())
	if err != nil {
		return 0, fmt.Errorf("network: write packet[%d]: %v", sequence, err)
	}
	return len(p), nil
}

// Read read data
func (c *Conn) Read(p []byte) (int, error) {
	select {
	case <-c.ctx.Done():
		return 0, c.err
	case data := <-c.chRead:
		if len(p) < len(data) {
			return 0, errBufferTooShort
		}
		return copy(p, data), nil
	}
}

func (c *Conn) read(p []byte) (*header, int, error) {
	c.mRead.Lock()
	defer c.mRead.Unlock()
	var hdr header
	err := binary.Read(c.conn, binary.BigEndian, &hdr)
	if err != nil {
		return nil, 0, fmt.Errorf("network: read packet header: %v", err)
	}
	if len(p) < int(hdr.Size) {
		return nil, 0, errBufferTooShort
	}
	if hdr.Size == 0 {
		return &hdr, 0, nil
	}
	n, err := io.ReadFull(c.conn, p[:hdr.Size])
	if err != nil {
		return nil, 0, fmt.Errorf("network: read packet payload[%d]: %v", hdr.Sequence, err)
	}
	if crc32.ChecksumIEEE(p[:hdr.Size]) != hdr.Crc32 {
		return nil, 0, errInvalidPacketChecksum
	}
	return &hdr, n, nil
}

func dup(data []byte) []byte {
	ret := make([]byte, len(data))
	copy(ret, data)
	return ret
}

func (c *Conn) onClose(err error) {
	logging.Error("connection closed: %s", c.conn.RemoteAddr().String())
	c.conn.Close()
	var streams []*Stream
	c.mStreams.RLock()
	for _, stream := range c.streams {
		streams = append(streams, stream)
	}
	c.mStreams.RUnlock()
	for _, stream := range streams {
		stream.onClose(err)
	}
}

func (c *Conn) loopRead(cancel context.CancelFunc) {
	var err error
	defer func() {
		c.err = err
		cancel()
	}()
	defer c.onClose(err)
	buf := make([]byte, math.MaxUint16)
	for {
		var hdr *header
		var n int
		hdr, n, err = c.read(buf)
		if err != nil {
			logging.Error("loop read => %s: %v", c.conn.RemoteAddr().String(), err)
			return
		}
		if hdr.Flag&flagPing != 0 {
			err = c.handlePing()
			if err != nil {
				logging.Error("handle ping => %s: %v", c.conn.RemoteAddr().String(), err)
				return
			}
			continue
		}
		if hdr.Flag&flagStreamOpen != 0 {
			err = c.handleOpenStream()
			if err != nil {
				logging.Error("handle open stream => %s: %v", c.conn.RemoteAddr().String(), err)
				return
			}
			continue
		}
		if hdr.Flag&flagStreamOpenAck != 0 {
			err = c.handleOpenStreamAck(hdr.Flag)
			if err != nil {
				logging.Error("handle open stream ack => %s: %v", c.conn.RemoteAddr().String(), err)
				return
			}
			continue
		}
		if hdr.Flag&flagStreamClose != 0 {
			err = c.handleCloseStream(hdr.Flag)
			if err != nil {
				if err == errStreamNotFound {
					continue
				}
				logging.Error("handle close stream => %s: %v", c.conn.RemoteAddr().String(), err)
				return
			}
			continue
		}
		if hdr.Flag&flagStreamData != 0 {
			err = c.handleStreamData(hdr.Flag, buf[:n])
			if err != nil {
				if err == errStreamNotFound {
					continue
				}
				logging.Error("handle data => %s: %v", c.conn.RemoteAddr().String(), err)
				return
			}
			continue
		}
		if hdr.Size == 0 {
			continue
		}
		c.chRead <- dup(buf[:n])
	}
}

func (c *Conn) getStream(stream uint32) *Stream {
	stream = stream & 0xffffff
	c.mStreams.RLock()
	defer c.mStreams.RUnlock()
	return c.streams[stream]
}

// SendKeepalive send keepalive packet
func (c *Conn) SendKeepalive() error {
	var buf bytes.Buffer
	sequence := c.sequence.Add(1)
	err := binary.Write(&buf, binary.BigEndian, header{
		Size:     0,
		Crc32:    0,
		Sequence: sequence,
		Flag:     flagPing,
	})
	if err != nil {
		return fmt.Errorf("network: build ping packet header[%d]: %v", sequence, err)
	}
	_, err = c.conn.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("network: write ping packet[%d]: %v", sequence, err)
	}
	return nil
}

func (c *Conn) handlePing() error {
	var buf bytes.Buffer
	sequence := c.sequence.Add(1)
	err := binary.Write(&buf, binary.BigEndian, header{
		Size:     0,
		Crc32:    0,
		Sequence: sequence,
		Flag:     flagPong,
	})
	if err != nil {
		return fmt.Errorf("network: build pong packet header[%d]: %v", sequence, err)
	}
	_, err = c.conn.Write(buf.Bytes())
	if err != nil {
		return fmt.Errorf("network: write pong packet[%d]: %v", sequence, err)
	}
	return nil
}

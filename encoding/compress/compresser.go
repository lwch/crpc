package compress

import (
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"io"
	"sync"

	"github.com/klauspost/compress/zstd"
)

var errNoDecompresser = errors.New("compress: no decompresser")
var errInvalidChecksum = errors.New("compress: invalid checksum")

// Method compress method
type Method byte

const (
	// Gzip gzip method
	Gzip Method = 1 << 0
	// Zstd zstd method
	Zstd Method = 1 << 1
)

type compresser interface {
	io.Writer
	Reset(io.Writer)
}

type decompresser interface {
	io.Reader
	Reset(io.Reader) error
}

// Compresser compresser
type Compresser struct {
	nc               func(int) (compresser, error)
	level            int
	poolCompresser   map[int]*sync.Pool
	mPoolCompresser  sync.RWMutex
	poolDecompresser sync.Pool
}

// New create new compresser
func New(m Method) *Compresser {
	switch m {
	case Gzip:
		cp := &Compresser{
			nc:             newGzipCompresser,
			level:          gzip.DefaultCompression,
			poolCompresser: make(map[int]*sync.Pool),
		}
		cp.poolDecompresser.New = func() any {
			decompresser, err := newGzipDecompresser()
			if err != nil {
				return nil
			}
			return decompresser
		}
		return cp
	case Zstd:
		cp := &Compresser{
			nc:             newZstdCompresser,
			level:          int(zstd.SpeedDefault),
			poolCompresser: make(map[int]*sync.Pool),
		}
		cp.poolDecompresser.New = func() any {
			decompresser, err := newZstdDecompresser()
			if err != nil {
				return nil
			}
			return decompresser
		}
		return cp
	default:
		return nil
	}
}

// Compress compress func
func (cp *Compresser) Compress(data []byte) ([]byte, error) {
	cp.mPoolCompresser.RLock()
	pool := cp.poolCompresser[cp.level]
	cp.mPoolCompresser.RUnlock()
	if pool == nil {
		pool = new(sync.Pool)
		pool.New = func() any {
			compresser, err := cp.nc(cp.level)
			if err != nil {
				return nil
			}
			return compresser
		}
		cp.mPoolCompresser.Lock()
		cp.poolCompresser[cp.level] = pool
		cp.mPoolCompresser.Unlock()
	}
	obj := pool.Get()
	var w compresser
	if obj == nil {
		var err error
		w, err = cp.nc(cp.level)
		if err != nil {
			return nil, err
		}
	} else {
		w = obj.(compresser)
	}
	defer pool.Put(w)
	data = binary.BigEndian.AppendUint32(data, crc32.ChecksumIEEE(data))
	var buf bytes.Buffer
	w.Reset(&buf)
	_, err := io.Copy(w, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Decompress decompress func
func (cp *Compresser) Decompress(data []byte) ([]byte, error) {
	obj := cp.poolDecompresser.Get()
	if obj == nil {
		return nil, errNoDecompresser
	}
	r := obj.(decompresser)
	err := r.Reset(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	_, err = io.Copy(&buf, r)
	if err != nil {
		return nil, err
	}
	data = buf.Bytes()
	sum := binary.BigEndian.Uint32(data[len(data)-4:])
	data = data[:len(data)-4]
	if crc32.ChecksumIEEE(data) != sum {
		return nil, errInvalidChecksum
	}
	return data, nil
}

// SetLevel set compress level
func (cp *Compresser) SetLevel(level int) {
	cp.level = level
}

package compress

import (
	"io"

	"github.com/klauspost/compress/zstd"
)

func newZstdCompresser(level int) (compresser, error) {
	return zstd.NewWriter(io.Discard, zstd.WithEncoderLevel(zstd.EncoderLevel(level)))
}

func newZstdDecompresser() (io.Reader, error) {
	return zstd.NewReader(nil)
}

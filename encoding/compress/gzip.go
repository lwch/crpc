package compress

import (
	"compress/gzip"
	"io"
)

func newGzipCompresser(level int) (compresser, error) {
	return gzip.NewWriterLevel(io.Discard, level)
}

func newGzipDecompresser(r io.Reader) (io.Reader, error) {
	return gzip.NewReader(r)
}

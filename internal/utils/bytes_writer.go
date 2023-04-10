package utils

// BytesWriter is a simple io.Writer implementation that writes to a []byte.
type BytesWriter []byte

// Write implements io.Writer.
func (w BytesWriter) Write(p []byte) (int, error) {
	return copy(w, p), nil
}

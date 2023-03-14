package encoding

// Codec serializer
type Codec interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) error
}

package encoding

// Codec serializer
type Codec interface {
	Marshal(any) ([]byte, error)
	Unmarshal([]byte, any) (int, error)
}

// Encrypter encrypter
type Encrypter interface {
	Encrypt([]byte) ([]byte, error)
	Decrypt([]byte) ([]byte, error)
}

// Compresser compresser
type Compresser interface {
	Compress([]byte) ([]byte, error)
	Decompress([]byte) ([]byte, error)
}

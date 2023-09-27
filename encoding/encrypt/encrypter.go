package encrypt

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/des"
	"encoding/binary"
	"errors"
	"hash/crc32"
)

var errInvalidChecksum = errors.New("encrypt: invalid checksum")
var errInvalidBlockSize = errors.New("encrypt: invalid block size")

// Method encrypt method
type Method byte

const (
	// Aes aes method
	Aes Method = iota
	// Des des method
	Des
)

type padFunc func([]byte) []byte

// Encrypter encrypter
type Encrypter struct {
	block cipher.Block
	iv    []byte
	pad   padFunc
	unpad padFunc
}

func makePad(size int) padFunc {
	return func(p []byte) []byte {
		if len(p) == 0 {
			return p
		}
		padSize := size - (len(p) % size)
		pad := bytes.Repeat([]byte{byte(padSize)}, padSize)
		return append(p, pad...)
	}
}

func unpad(p []byte) []byte {
	padSize := int(p[len(p)-1])
	return p[:len(p)-padSize]
}

func repeat(str string, limit int) string {
	for len(str) < limit {
		str += str
	}
	return str
}

// New create new encrypter
func New(m Method, key string) *Encrypter {
	var block cipher.Block
	var iv []byte
	var err error
	var pad padFunc
	switch m {
	case Aes:
		key = repeat(key, 32+aes.BlockSize)
		block, err = aes.NewCipher([]byte(key[:32]))
		if err != nil {
			return nil
		}
		iv = []byte(key[32 : 32+aes.BlockSize])
		pad = makePad(aes.BlockSize)
	case Des:
		key = repeat(key, 24+des.BlockSize)
		block, err = des.NewTripleDESCipher([]byte(key[:24]))
		if err != nil {
			return nil
		}
		iv = []byte(key[24 : 24+des.BlockSize])
		pad = makePad(des.BlockSize)
	}
	return &Encrypter{
		block: block,
		iv:    iv,
		pad:   pad,
		unpad: unpad,
	}
}

// Encrypt encrypt data
func (enc *Encrypter) Encrypt(src []byte) ([]byte, error) {
	bm := cipher.NewCBCEncrypter(enc.block, enc.iv)
	src = binary.BigEndian.AppendUint32(src, crc32.ChecksumIEEE(src))
	src = enc.pad(src)
	dst := make([]byte, len(src))
	bm.CryptBlocks(dst, src)
	return dst, nil
}

// Decrypt decrypt data
func (enc *Encrypter) Decrypt(src []byte) ([]byte, error) {
	if len(src) == 0 {
		return src, nil
	}
	bm := cipher.NewCBCDecrypter(enc.block, enc.iv)
	if len(src)%bm.BlockSize() != 0 {
		return nil, errInvalidBlockSize
	}
	dst := make([]byte, len(src))
	bm.CryptBlocks(dst, src)
	dst = enc.unpad(dst)
	sum := binary.BigEndian.Uint32(dst[len(dst)-4:])
	dst = dst[:len(dst)-4]
	if crc32.ChecksumIEEE(dst) != sum {
		return nil, errInvalidChecksum
	}
	return dst, nil
}

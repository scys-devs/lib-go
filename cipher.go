package lib

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"errors"
	"fmt"
	"github.com/jxskiss/base62"
	"io"
)

func MD5(s string) string {
	h := md5.New()
	io.WriteString(h, s)
	return fmt.Sprintf("%x", h.Sum(nil))
}

func PKCS7Padding(plaintext string, blockSize int) []byte {
	padding := blockSize - (len(plaintext) % blockSize)
	padtext := bytes.Repeat([]byte{byte(padding)}, padding)
	var buffer bytes.Buffer
	buffer.WriteString(plaintext)
	buffer.Write(padtext)
	return buffer.Bytes()
}

func PKCS7UnPadding(plaintext []byte, blockSize int) ([]byte, error) {
	plaintextLength := len(plaintext)
	if nil == plaintext || plaintextLength == 0 {
		return nil, errors.New("pKCS7Unpadding error nil or zero")
	}
	if plaintextLength%blockSize != 0 {
		return nil, errors.New("pKCS7Unpadding text not a multiple of the block size")
	}
	paddingLength := int(plaintext[plaintextLength-1])
	return plaintext[:plaintextLength-paddingLength], nil
}

func AesCbcEncrypt(src, key []byte) []byte {
	block, _ := aes.NewCipher(key)
	src = PKCS7Padding(string(src), block.BlockSize())

	cipherText := make([]byte, len(src))
	mode := cipher.NewCBCEncrypter(block, key)
	mode.CryptBlocks(cipherText, src)

	return cipherText
}

func AesCbcDecrypt(src, key []byte) []byte {
	block, _ := aes.NewCipher(key)

	dst := make([]byte, len(src))
	mode := cipher.NewCBCDecrypter(block, key)
	mode.CryptBlocks(dst, src)

	b, _ := PKCS7UnPadding(dst, block.BlockSize())
	return b
}

//----------------------------------------------------------------------------------------------------------
// 用于数字转短串
type B62Encoder struct {
	encoding *base62.Encoding
}

func (do *B62Encoder) Encode(id int64) string {
	b := do.encoding.FormatInt(id)
	b = PKCS7Padding(BytesToStr(b), 6)
	return do.encoding.EncodeToString(b)
}

func (do *B62Encoder) Decode(s string) int64 {
	b, _ := do.encoding.DecodeString(s)
	raw, _ := PKCS7UnPadding(b, 6)
	id, _ := do.encoding.ParseInt(raw)
	return id
}

func NewB62Encoder(s string) *B62Encoder {
	return &B62Encoder{encoding: base62.NewEncoding(s)}
}

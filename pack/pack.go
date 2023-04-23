package pack

// ********************************************************************************
// WARNING: DO NOT MODIFY THIS FILE. IT WILL BE REPLACED BY THE APPLICATION CODE.
// *********************************************************************************

import (
	"io"

	"github.com/yaoapp/gou/application/yaz/ciphers"
)

// Pack the yao app package
type Pack struct{ aes ciphers.AES }

// Cipher the cipher
var Cipher *Pack

// SetCipher set the cipher
func SetCipher(license string) {
	Cipher = &Pack{aes: ciphers.NewAES([]byte(license))}
}

// Encrypt encrypt
func (pack *Pack) Encrypt(reader io.Reader, writer io.Writer) error {
	return pack.aes.Encrypt(reader, writer)
}

// Decrypt decrypt
func (pack *Pack) Decrypt(reader io.Reader, writer io.Writer) error {
	return pack.aes.Decrypt(reader, writer)
}

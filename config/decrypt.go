package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/sha256"
	"encoding/base64"
	"strings"
)

const encPrefix = "enc:"

// DecryptValue decrypts a value encrypted by cloud settings.
// Returns the original string if not encrypted (no "enc:" prefix)
// or if no AES key is configured.
func DecryptValue(s string) string {
	if !strings.HasPrefix(s, encPrefix) {
		return s
	}
	secret := Conf.DB.AESKey
	if secret == "" {
		return strings.TrimPrefix(s, encPrefix)
	}
	dec, err := aesGCMDecrypt(strings.TrimPrefix(s, encPrefix), secret)
	if err != nil {
		return s
	}
	return dec
}

func aesGCMDecrypt(encoded, secret string) (string, error) {
	key := sha256.Sum256([]byte(secret))
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return "", err
	}
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", aes.KeySizeError(len(data))
	}
	plaintext, err := gcm.Open(nil, data[:nonceSize], data[nonceSize:], nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

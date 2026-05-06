package setting

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"io"
	"strings"

	"github.com/yaoapp/yao/config"
)

const encPrefix = "enc:"

// Encrypt encrypts a plaintext string using AES-256-GCM with the configured AES key.
// Returns the original string if no AES key is configured.
func Encrypt(plaintext string) string {
	secret := config.Conf.DB.AESKey
	if secret == "" {
		return plaintext
	}
	enc, err := aesGCMEncrypt(plaintext, secret)
	if err != nil {
		return plaintext
	}
	return encPrefix + enc
}

// Decrypt decrypts a value previously encrypted by Encrypt.
// Returns the original string if not encrypted or if decryption fails.
func Decrypt(value string) string {
	return config.DecryptValue(value)
}

// IsEncrypted returns true if the value has the encryption prefix.
func IsEncrypted(value string) bool {
	return strings.HasPrefix(value, encPrefix)
}

func aesGCMEncrypt(plaintext, secret string) (string, error) {
	h := sha256.Sum256([]byte(secret))
	block, err := aes.NewCipher(h[:])
	if err != nil {
		return "", err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

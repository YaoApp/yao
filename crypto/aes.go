package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/hex"
	"fmt"
)

// AES256Encrypt AES Encrypt
func AES256Encrypt(key string, algorithm string, nonce string, text string, additionalData string, encoding ...string) (string, error) {
	switch algorithm {
	case "GCM":
		var add []byte
		if additionalData != "" {
			add = []byte(additionalData)
		}
		ciphertext, err := aes256GCMEncrypt([]byte(key), []byte(nonce), []byte(text), add)
		if err != nil {
			return "", err
		}
		if len(encoding) > 0 && encoding[0] == "base64" {
			return base64.StdEncoding.EncodeToString(ciphertext), nil
		}
		return hex.EncodeToString(ciphertext), nil
	}
	return "", fmt.Errorf("algorithm %s not support", algorithm)
}

// AES256Decrypt AES Decrypt
func AES256Decrypt(key string, algorithm string, nonce string, ciphertext string, additionalData string, encoding ...string) (string, error) {
	switch algorithm {
	case "GCM":
		var bytes []byte
		var err error
		if len(encoding) > 0 && encoding[0] == "base64" {
			bytes, err = base64.StdEncoding.DecodeString(ciphertext)
			if err != nil {
				return "", err
			}
		} else {
			bytes, err = hex.DecodeString(ciphertext)
			if err != nil {
				return "", err
			}
		}

		var add []byte
		if additionalData != "" {
			add = []byte(additionalData)
		}
		text, err := aes256GCMDecrypt([]byte(key), []byte(nonce), bytes, add)
		if err != nil {
			return "", err
		}

		return string(text), nil
	}
	return "", fmt.Errorf("algorithm %s not support", algorithm)
}

func aes256GCMDecrypt(key, nonce, ciphertext, additionalData []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, fmt.Errorf("key length must be 32")
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, err
	}

	decrypted, err := gcm.Open(nil, nonce, ciphertext, []byte(additionalData))
	if err != nil {
		return nil, fmt.Errorf("gcm open error: %s", err)
	}

	return decrypted, nil
}

func aes256GCMEncrypt(key, nonce, text, additionalData []byte) ([]byte, error) {

	if len(key) != 32 {
		return nil, fmt.Errorf("key length must be 32")
	}

	c, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	// Create a GCM block mode instance
	gcm, err := cipher.NewGCM(c)
	if err != nil {
		return nil, fmt.Errorf("gcm error: %s", err)
	}

	ciphertext := gcm.Seal(nil, nonce, text, additionalData)
	return ciphertext, nil
}

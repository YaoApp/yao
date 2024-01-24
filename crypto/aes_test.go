package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou/process"
)

func TestAES256GCM(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	crypted, err := AES256Encrypt(key, "GCM", nonce, text, additionalData)
	if err != nil {
		t.Errorf("AES256Encrypt error: %s", err)
	}

	decrypted, err := AES256Decrypt(key, "GCM", nonce, crypted, additionalData)
	if err != nil {
		t.Errorf("AES256Decrypt error: %s", err)
	}

	assert.Equal(t, text, decrypted)
}

func TestAES256GCMBase64(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	crypted, err := AES256Encrypt(key, "GCM", nonce, text, additionalData, "base64")
	if err != nil {
		t.Errorf("AES256Encrypt error: %s", err)
	}

	decrypted, err := AES256Decrypt(key, "GCM", nonce, crypted, additionalData, "base64")
	if err != nil {
		t.Errorf("AES256Decrypt error: %s", err)
	}
	assert.Equal(t, text, decrypted)

}

func TestAES256ProcessGCM(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	args := []interface{}{"GCM", key, nonce, text, additionalData}
	crypted, err := process.New("crypto.Aes256Encrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	args = []interface{}{"GCM", key, nonce, crypted, additionalData}
	decrypted, err := process.New("crypto.Aes256Decrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, text, decrypted)
}

func TestAES256ProcessGCMBase64(t *testing.T) {
	key := `oxxxyXVBqwqUjmbgKlwuHV2mgxxxcfOa`
	nonce := `LJEcFT6QWjkG`
	text := `{"name":"yao"}`
	additionalData := `transaction`

	args := []interface{}{"GCM", key, nonce, text, additionalData, "base64"}
	crypted, err := process.New("crypto.Aes256Encrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	args = []interface{}{"GCM", key, nonce, crypted, additionalData, "base64"}
	decrypted, err := process.New("crypto.Aes256Decrypt", args...).Exec()
	if err != nil {
		t.Fatal(err)
	}

	assert.Equal(t, text, decrypted)
}

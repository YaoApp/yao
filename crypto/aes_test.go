package crypto

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/gou"
)

func TestBase64AES(t *testing.T) {

	originalText := "encrypt this golang"
	key := []byte("example key 1234")

	// encrypt value to base64
	cryptoText, err := Base64AESEncode(key, originalText)
	if err != nil {
		t.Fatal(err)
	}

	// encrypt base64 crypto to original value
	text, err := Base64AESDecode(key, cryptoText)
	if err != nil {
		t.Fatal(err)
	}
	assert.Equal(t, "encrypt this golang", text)
}

func TestBase64AESProcess(t *testing.T) {

	args := []interface{}{"example key 1234", "encrypt this golang"}
	cryptoText := gou.NewProcess("yao.crypto.AESBase64Encode", args...).Run()

	args = []interface{}{"example key 1234", cryptoText}
	text := gou.NewProcess("yao.crypto.AESBase64Decode", args...).Run()
	assert.Equal(t, "encrypt this golang", text)
}

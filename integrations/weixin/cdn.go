package weixin

import (
	"crypto/aes"
	"encoding/base64"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

func buildCDNDownloadURL(cdnBaseURL, encryptedQueryParam string) string {
	return cdnBaseURL + "/download?encrypted_query_param=" + url.QueryEscape(encryptedQueryParam)
}

func parseAesKey(aesKeyBase64 string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(aesKeyBase64)
	if err != nil {
		return nil, fmt.Errorf("parseAesKey: base64 decode: %w", err)
	}
	if len(decoded) == 16 {
		return decoded, nil
	}
	if len(decoded) == 32 {
		hexStr := string(decoded)
		raw := make([]byte, 16)
		for i := 0; i < 16; i++ {
			var b byte
			_, err := fmt.Sscanf(hexStr[i*2:i*2+2], "%02x", &b)
			if err != nil {
				return nil, fmt.Errorf("parseAesKey: hex parse: %w", err)
			}
			raw[i] = b
		}
		return raw, nil
	}
	return nil, fmt.Errorf("parseAesKey: unexpected decoded len=%d", len(decoded))
}

func decryptAES128ECB(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	bs := block.BlockSize()
	if len(ciphertext)%bs != 0 {
		return nil, fmt.Errorf("decryptAES128ECB: ciphertext len %d not multiple of block size", len(ciphertext))
	}
	dst := make([]byte, len(ciphertext))
	for i := 0; i < len(ciphertext); i += bs {
		block.Decrypt(dst[i:i+bs], ciphertext[i:i+bs])
	}
	if len(dst) == 0 {
		return dst, nil
	}
	padLen := int(dst[len(dst)-1])
	if padLen == 0 || padLen > bs {
		return nil, fmt.Errorf("decryptAES128ECB: invalid PKCS7 padding %d", padLen)
	}
	return dst[:len(dst)-padLen], nil
}

func downloadCDNBytes(cdnBaseURL, encryptedQueryParam string) ([]byte, error) {
	u := buildCDNDownloadURL(cdnBaseURL, encryptedQueryParam)
	resp, err := http.Get(u) //nolint:gosec
	if err != nil {
		return nil, fmt.Errorf("CDN download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("CDN download HTTP %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

func DownloadAndDecrypt(cdnBaseURL, encryptedQueryParam, aesKeyBase64 string) ([]byte, error) {
	key, err := parseAesKey(aesKeyBase64)
	if err != nil {
		return nil, err
	}
	data, err := downloadCDNBytes(cdnBaseURL, encryptedQueryParam)
	if err != nil {
		return nil, err
	}
	return decryptAES128ECB(data, key)
}

func DecryptFromRaw(cdnBaseURL, encryptedQueryParam string, rawKey []byte) ([]byte, error) {
	data, err := downloadCDNBytes(cdnBaseURL, encryptedQueryParam)
	if err != nil {
		return nil, err
	}
	return decryptAES128ECB(data, rawKey)
}

func DownloadPlain(cdnBaseURL, encryptedQueryParam string) ([]byte, error) {
	return downloadCDNBytes(cdnBaseURL, encryptedQueryParam)
}

func encryptAES128ECB(plaintext, key []byte) []byte {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil
	}
	bs := block.BlockSize()
	padLen := bs - (len(plaintext) % bs)
	padded := make([]byte, len(plaintext)+padLen)
	copy(padded, plaintext)
	for i := len(plaintext); i < len(padded); i++ {
		padded[i] = byte(padLen)
	}
	dst := make([]byte, len(padded))
	for i := 0; i < len(padded); i += bs {
		block.Encrypt(dst[i:i+bs], padded[i:i+bs])
	}
	return dst
}

func aesEcbPaddedSize(plaintextSize int) int {
	return ((plaintextSize + 1 + 15) / 16) * 16
}

func buildCDNUploadURL(cdnBaseURL, uploadParam, filekey string) string {
	return cdnBaseURL + "/upload?encrypted_query_param=" +
		url.QueryEscape(uploadParam) + "&filekey=" + url.QueryEscape(filekey)
}

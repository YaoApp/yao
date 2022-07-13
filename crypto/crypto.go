package crypto

import (
	"crypto"
	"crypto/hmac"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/md4"
)

func init() {
	crypto.RegisterHash(crypto.MD4, md4.New)
}

var hashTypes = map[string]crypto.Hash{
	"MD4":         crypto.MD4,
	"MD5":         crypto.MD5,
	"SHA1":        crypto.SHA1,
	"SHA224":      crypto.SHA224,
	"SHA256":      crypto.SHA256,
	"SHA384":      crypto.SHA384,
	"SHA512":      crypto.SHA512,
	"MD5SHA1":     crypto.MD5SHA1,
	"RIPEMD160":   crypto.RIPEMD160,
	"SHA3_224":    crypto.SHA3_224,
	"SHA3_256":    crypto.SHA3_256,
	"SHA3_384":    crypto.SHA3_384,
	"SHA3_512":    crypto.SHA3_512,
	"SHA512_224":  crypto.SHA512_224,
	"SHA512_256":  crypto.SHA512_256,
	"BLAKE2s_256": crypto.BLAKE2s_256,
	"BLAKE2b_256": crypto.BLAKE2b_256,
	"BLAKE2b_384": crypto.BLAKE2b_384,
	"BLAKE2b_512": crypto.BLAKE2b_512,
}

// Hash string
func Hash(hash crypto.Hash, value string) (string, error) {
	h := hash.New()
	_, err := h.Write([]byte(value))
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}

// Hmac the Keyed-Hash Message Authentication Code (HMAC)
func Hmac(hash crypto.Hash, value string, key string, encoding ...string) (string, error) {
	mac := hmac.New(hash.New, []byte(key))
	_, err := mac.Write([]byte(value))
	if err != nil {
		return "", err
	}

	if len(encoding) > 0 && encoding[0] == "base64" {
		return base64.StdEncoding.EncodeToString(mac.Sum(nil)), nil
	}

	return fmt.Sprintf("%x", mac.Sum(nil)), nil
}

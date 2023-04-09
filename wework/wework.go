package wework

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"encoding/base64"
	"encoding/binary"
	"strings"
)

// Decrypt wework msg Decrypt
func Decrypt(encodingAESKey string, msgEncrypt string, parse bool) (map[string]interface{}, error) {

	var err error
	aseKey, err := base64.StdEncoding.DecodeString(encodingAESKey + "=")
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(msgEncrypt)
	if err != nil {
		return nil, err
	}

	randMsg, err := aesDecrypt(ciphertext, aseKey)
	if err != nil {
		return nil, err
	}

	content := randMsg[16:]
	buf := bytes.NewBuffer(content[0:4])
	var len int32
	binary.Read(buf, binary.BigEndian, &len)
	msg := content[4 : len+4]
	receiveid := content[len+4:]

	data := map[string]interface{}{}
	if parse {
		data, err = parseXML(string(msg))
		if err != nil {
			return nil, err
		}
	}

	return map[string]interface{}{
		"message":   string(msg),
		"data":      data,
		"receiveid": string(receiveid),
	}, nil
}

func parseXML(data string) (map[string]interface{}, error) {

	decoder := NewDecoder(strings.NewReader(data))
	result, err := decoder.Decode()
	if err != nil {
		return nil, err
	}
	return result, nil
}

func aesDecrypt(crypted, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	blockSize := block.BlockSize()
	blockMode := cipher.NewCBCDecrypter(block, key[:blockSize])
	origData := make([]byte, len(crypted))
	blockMode.CryptBlocks(origData, crypted)
	origData = pckS5UnPadding(origData)
	return origData, nil
}

func pckS5UnPadding(origData []byte) []byte {
	length := len(origData)
	unpadding := int(origData[length-1])
	return origData[:(length - unpadding)]
}

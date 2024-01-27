package crypto

import (
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
)

func init() {
	process.Register("yao.crypto.hash", ProcessHash) // deprecated → crypto.Hash
	process.Register("yao.crypto.hmac", ProcessHmac) // deprecated → crypto.Hash

	process.Alias("yao.crypto.hash", "crypto.Hash")
	process.Alias("yao.crypto.hmac", "crypto.Hmac")

	process.Register("crypto.hmacwith", ProcessHmacWith)
	process.Register("crypto.rsa2sign", ProcessRsa2Sign)
	process.Register("crypto.rsa2verify", ProcessRsa2Verify)
	process.Register("crypto.aes256encrypt", ProcessAes256Encrypt)
	process.Register("crypto.aes256decrypt", ProcessAes256Decrypt)
}

// ProcessRSA2 yao.crypto.rsa Crypto RSA
func ProcessRSA2(process *process.Process) interface{} {
	process.ValidateArgNums(3)

	return nil
}

// ProcessHash yao.crypto.hash Crypto Hash
// Args[0] string: the hash function name. MD4/MD5/SHA1/SHA224/SHA256/SHA384/SHA512/MD5SHA1/RIPEMD160/SHA3_224/SHA3_256/SHA3_384/SHA3_512/SHA512_224/SHA512_256/BLAKE2s_256/BLAKE2b_256/BLAKE2b_384/BLAKE2b_512
// Args[1] string: value
func ProcessHash(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	typ := process.ArgsString(0)
	value := process.ArgsString(1)

	h, has := HashTypes[typ]
	if !has {
		exception.New("%s does not support", 400, typ).Throw()
	}

	res, err := Hash(h, value)
	if err != nil {
		exception.New("%s error: %s value: %s", 400, typ, err, value).Throw()
	}
	return res
}

// ProcessHmac yao.crypto.hmac Crypto the Keyed-Hash Message Authentication Code (HMAC) Hash
// Args[0] string: the hash function name. MD4/MD5/SHA1/SHA224/SHA256/SHA384/SHA512/MD5SHA1/RIPEMD160/SHA3_224/SHA3_256/SHA3_384/SHA3_512/SHA512_224/SHA512_256/BLAKE2s_256/BLAKE2b_256/BLAKE2b_384/BLAKE2b_512
// Args[1] string: value
// Args[2] string: key
// Args[3] string: base64 (optional)
func ProcessHmac(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	typ := process.ArgsString(0)
	value := process.ArgsString(1)
	key := process.ArgsString(2)

	h, has := HashTypes[typ]
	if !has {
		exception.New("%s does not support", 400, typ).Throw()
	}

	encoding := ""
	if process.NumOfArgs() > 3 {
		encoding = process.ArgsString(3)
	}

	res, err := Hmac(h, value, key, encoding)
	if err != nil {
		exception.New("%s error: %s value: %s", 400, typ, err, value).Throw()
	}
	return res
}

// ProcessHmacWith yao.crypto.hmac Crypto the Keyed-Hash Message Authentication Code (HMAC) Hash
// Args[0] map: option {"key": "base64", "value": "base64", "output": "base64", "algo": "SHA256"} // hex/base64
// Args[1] string: value
// Args[2] string: key
func ProcessHmacWith(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	option := process.ArgsMap(0)
	value := process.ArgsString(1)
	key := process.ArgsString(2)
	typ := "SHA256"
	o := &hmacOption{}
	if v, has := option["key"].(string); has {
		o.keyEncoding = v
	}
	if v, has := option["value"].(string); has {
		o.valueEncoding = v
	}
	if v, has := option["output"].(string); has {
		o.outputEncoding = v
	}
	if v, has := option["algo"].(string); has && v != "" {
		typ = v
	}
	h, has := HashTypes[typ]
	if !has {
		exception.New("%s does not support", 400, typ).Throw()
	}
	res, err := HmacWith(o, h, value, key)
	if err != nil {
		exception.New("%s error: %s value: %s", 400, typ, err, value).Throw()
	}
	return res
}

// ProcessRsa2Sign crypto.rsa2sign
// Args[0] string: the private key
// Args[1] string: the hash function name. MD4/MD5/SHA1/SHA224/SHA256/SHA384/SHA512/MD5SHA1/RIPEMD160/SHA3_224/SHA3_256/SHA3_384/SHA3_512/SHA512_224/SHA512_256/BLAKE2s_256/BLAKE2b_256/BLAKE2b_384/BLAKE2b_512
// Args[2] string: value
// Args[3] string: "base64" (optional)
func ProcessRsa2Sign(process *process.Process) interface{} {
	process.ValidateArgNums(3)
	pri := process.ArgsString(0)
	typ := process.ArgsString(1)
	value := process.ArgsString(2)
	base64 := process.ArgsString(3)

	h, has := HashTypes[typ]
	if !has {
		exception.New("%s does not support", 400, typ).Throw()
	}

	res, err := RSA2Sign(pri, h, value, base64)
	if err != nil {
		exception.New("%s error: %s value: %s", 400, typ, err, value).Throw()
	}
	return res
}

// ProcessRsa2Verify crypto.rsa2verify
// Args[0] string: the public key
// Args[1] string: the hash function name. MD4/MD5/SHA1/SHA224/SHA256/SHA384/SHA512/MD5SHA1/RIPEMD160/SHA3_224/SHA3_256/SHA3_384/SHA3_512/SHA512_224/SHA512_256/BLAKE2s_256/BLAKE2b_256/BLAKE2b_384/BLAKE2b_512
// Args[2] string: value
// Args[3] string: sign
// Args[4] string: "base64" (optional)
func ProcessRsa2Verify(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	pub := process.ArgsString(0)
	typ := process.ArgsString(1)
	value := process.ArgsString(2)
	sign := process.ArgsString(3)
	base64 := process.ArgsString(4)

	h, has := HashTypes[typ]
	if !has {
		exception.New("%s does not support", 400, typ).Throw()
	}

	res, err := RSA2Verify(pub, h, value, sign, base64)
	if err != nil {
		exception.New("%s error: %s value: %s", 400, typ, err, value).Throw()
	}
	return res
}

// ProcessAes256Encrypt crypto.aes256encrypt
// Args[0] string: the algorithm "GCM"
// Args[1] string: the key
// Args[2] string: the nonce
// Args[3] string: the text
// Args[4] string: the additionalData
// Args[5] string: "base64" (optional)
func ProcessAes256Encrypt(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	algorithm := process.ArgsString(0)
	key := process.ArgsString(1)
	nonce := process.ArgsString(2)
	text := process.ArgsString(3)
	additionalData := process.ArgsString(4)
	encoding := process.ArgsString(5)

	res, err := AES256Encrypt(key, algorithm, nonce, text, additionalData, encoding)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

// ProcessAes256Decrypt crypto.aes256decrypt
// Args[0] string: the algorithm "GCM"
// Args[1] string: the key
// Args[2] string: the nonce
// Args[3] string: the crypted
// Args[4] string: the additionalData
// Args[5] string: "base64" (optional)
func ProcessAes256Decrypt(process *process.Process) interface{} {
	process.ValidateArgNums(4)
	algorithm := process.ArgsString(0)
	key := process.ArgsString(1)
	nonce := process.ArgsString(2)
	crypted := process.ArgsString(3)
	additionalData := process.ArgsString(4)
	encoding := process.ArgsString(5)
	res, err := AES256Decrypt(key, algorithm, nonce, crypted, additionalData, encoding)
	if err != nil {
		exception.Err(err, 500).Throw()
	}
	return res
}

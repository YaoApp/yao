package crypto

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func init() {
	gou.RegisterProcessHandler("yao.crypto.hash", ProcessHash)
	gou.RegisterProcessHandler("yao.crypto.hmac", ProcessHmac)
}

// ProcessHash yao.crypto.hash Crypto Hash
// Args[0] string: the hash function name. MD4/MD5/SHA1/SHA224/SHA256/SHA384/SHA512/MD5SHA1/RIPEMD160/SHA3_224/SHA3_256/SHA3_384/SHA3_512/SHA512_224/SHA512_256/BLAKE2s_256/BLAKE2b_256/BLAKE2b_384/BLAKE2b_512
// Args[1] string: value
func ProcessHash(process *gou.Process) interface{} {
	process.ValidateArgNums(2)
	typ := process.ArgsString(0)
	value := process.ArgsString(1)

	h, has := hashTypes[typ]
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
// Args[3] string: base64
func ProcessHmac(process *gou.Process) interface{} {
	process.ValidateArgNums(3)
	typ := process.ArgsString(0)
	value := process.ArgsString(1)
	key := process.ArgsString(2)

	h, has := hashTypes[typ]
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

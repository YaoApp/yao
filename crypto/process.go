package crypto

import (
	"github.com/yaoapp/gou"
	"github.com/yaoapp/kun/exception"
)

func init() {
	gou.RegisterProcessHandler("yao.crypto.hash", ProcessHash)
}

// ProcessHash yao.crypto.hash Crypto Hash
// Args[0] string: the hash function name. MD4/MD5/SHA1/SHA224/SHA256/SHA384/SHA512/MD5SHA1/RIPEMD160/SHA3_224/SHA3_256/SHA3_384/SHA3_512/SHA512_224/SHA512_256/BLAKE2s_256/BLAKE2b_256/BLAKE2b_384/BLAKE2b_512
// Args[1] string: value
func ProcessHash(process *gou.Process) interface{} {
	process.ValidateArgNums(1)
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

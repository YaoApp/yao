package taiid

import (
	"crypto/sha256"
	"fmt"
	"math/big"
)

const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Generate produces a deterministic tai_id from a machine ID and a node ID.
// The result is "tai-" followed by a Base62-encoded truncated SHA-256 hash.
// Both machineID and nodeID must be non-empty.
func Generate(machineID, nodeID string) (string, error) {
	if machineID == "" || nodeID == "" {
		return "", fmt.Errorf("machineID and nodeID are required")
	}
	h := sha256.Sum256([]byte(machineID + ":" + nodeID))
	return "tai-" + base62Encode(h[:16]), nil
}

func base62Encode(data []byte) string {
	num := new(big.Int).SetBytes(data)
	base := big.NewInt(62)
	zero := big.NewInt(0)
	mod := new(big.Int)

	var encoded []byte
	for num.Cmp(zero) > 0 {
		num.DivMod(num, base, mod)
		encoded = append([]byte{base62Chars[mod.Int64()]}, encoded...)
	}
	if len(encoded) == 0 {
		return "0"
	}
	return string(encoded)
}

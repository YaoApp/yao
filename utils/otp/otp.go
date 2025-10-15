package otp

import (
	"crypto/rand"
	"math/big"
	"time"

	"github.com/dchest/captcha"
	"github.com/google/uuid"
)

// OTP store using captcha's MemoryStore
// Stores OTP codes with expiration (default: 10 minutes)
var store = captcha.NewMemoryStore(2048, 10*time.Minute)

// Option OTP configuration
type Option struct {
	Length     int    // Code length (default: 6)
	Expiration int    // Expiration time in seconds (default: 600)
	Type       string // Code type: "numeric" (default), "alphanumeric"
}

// NewOption creates default OTP configuration
func NewOption() Option {
	return Option{
		Length:     6,
		Expiration: 600, // 10 minutes
		Type:       "numeric",
	}
}

// Generate generates a new OTP code and returns id and code
// The id is used to identify the OTP, and the code is sent to user
func Generate(option Option) (string, string) {
	if option.Length <= 0 {
		option.Length = 6
	}

	if option.Type == "" {
		option.Type = "numeric"
	}

	// Generate unique ID for this OTP
	id := uuid.New().String()

	// Generate OTP code
	var code string
	switch option.Type {
	case "alphanumeric":
		code = generateAlphanumericCode(option.Length)
	default:
		code = generateNumericCode(option.Length)
	}

	// Store OTP code as bytes
	store.Set(id, []byte(code))

	return id, code
}

// Validate validates an OTP code against the stored value
// Returns true if valid, false otherwise
// The clear parameter indicates whether to delete the OTP after validation
func Validate(id string, code string, clear bool) bool {
	if id == "" || code == "" {
		return false
	}

	// Get stored OTP code
	storedBytes := store.Get(id, clear)
	if storedBytes == nil {
		return false
	}

	storedCode := string(storedBytes)
	return storedCode == code
}

// Get retrieves the OTP code for testing purposes
// Returns empty string if OTP ID not found or expired
func Get(id string) string {
	storedBytes := store.Get(id, false)
	if storedBytes == nil {
		return ""
	}
	return string(storedBytes)
}

// Delete deletes an OTP code from the store
func Delete(id string) {
	store.Get(id, true)
}

// generateNumericCode generates a random numeric code
func generateNumericCode(length int) string {
	const digits = "0123456789"
	return generateRandomString(length, digits)
}

// generateAlphanumericCode generates a random alphanumeric code
func generateAlphanumericCode(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	return generateRandomString(length, chars)
}

// generateRandomString generates a random string from the given character set
func generateRandomString(length int, charset string) string {
	result := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		num, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			// Fallback to less secure method if crypto/rand fails
			num = big.NewInt(int64(i % len(charset)))
		}
		result[i] = charset[num.Int64()]
	}

	return string(result)
}

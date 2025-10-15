package otp

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	option := NewOption()

	// Test numeric code generation
	id, code := Generate(option)
	assert.NotEmpty(t, id)
	assert.NotEmpty(t, code)
	assert.Equal(t, 6, len(code), "Default numeric code should be 6 digits")

	// Verify code is numeric
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9', "Code should be numeric")
	}
	t.Logf("Generated numeric code: id=%s, code=%s", id, code)

	// Test custom length
	option.Length = 4
	_, code4 := Generate(option)
	assert.Equal(t, 4, len(code4), "Custom length should be respected")

	// Test alphanumeric code
	option.Type = "alphanumeric"
	option.Length = 8
	_, alphaCode := Generate(option)
	assert.Equal(t, 8, len(alphaCode), "Alphanumeric code should match length")

	// Verify alphanumeric
	for _, c := range alphaCode {
		assert.True(t, (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9'),
			"Code should be alphanumeric uppercase")
	}
	t.Logf("Generated alphanumeric code: %s", alphaCode)
}

func TestValidate(t *testing.T) {
	option := NewOption()

	// Generate OTP
	id, code := Generate(option)
	t.Logf("Generated OTP: id=%s, code=%s", id, code)

	// Test valid code without clearing
	valid := Validate(id, code, false)
	assert.True(t, valid, "Valid OTP should pass validation")

	// Verify code is still available
	storedCode := Get(id)
	assert.Equal(t, code, storedCode, "Code should still be available when not cleared")

	// Test valid code with clearing
	valid = Validate(id, code, true)
	assert.True(t, valid, "Valid OTP should pass validation")

	// Verify code is deleted
	storedCode = Get(id)
	assert.Empty(t, storedCode, "OTP should be deleted after validation with clear=true")

	// Test invalid code
	id2, _ := Generate(option)
	valid = Validate(id2, "wrong_code", false)
	assert.False(t, valid, "Invalid OTP should fail validation")

	// Test empty values
	valid = Validate("", "", false)
	assert.False(t, valid, "Empty values should fail validation")

	// Test non-existent ID
	valid = Validate("non-existent-id", "123456", false)
	assert.False(t, valid, "Non-existent ID should fail validation")
}

func TestGet(t *testing.T) {
	option := NewOption()

	// Generate OTP
	id, code := Generate(option)

	// Test get
	retrievedCode := Get(id)
	assert.Equal(t, code, retrievedCode, "Should retrieve correct code")

	// Test get non-existent
	retrievedCode = Get("non-existent-id")
	assert.Empty(t, retrievedCode, "Non-existent ID should return empty string")
}

func TestDelete(t *testing.T) {
	option := NewOption()

	// Generate OTP
	id, code := Generate(option)

	// Verify exists
	retrievedCode := Get(id)
	assert.Equal(t, code, retrievedCode)

	// Delete
	Delete(id)

	// Verify deleted
	retrievedCode = Get(id)
	assert.Empty(t, retrievedCode, "Code should be deleted")
}

func TestNewOption(t *testing.T) {
	option := NewOption()
	assert.Equal(t, 6, option.Length, "Default length should be 6")
	assert.Equal(t, 600, option.Expiration, "Default expiration should be 600 seconds")
	assert.Equal(t, "numeric", option.Type, "Default type should be numeric")
}

func TestOTPExpiration(t *testing.T) {
	// Note: Testing actual expiration requires time manipulation
	// The OTP store has a 10-minute default expiration
	t.Skip("Skipping expiration test - requires time manipulation or long wait")
}

func TestOTPConcurrency(t *testing.T) {
	option := NewOption()

	// Test concurrent OTP generation and validation
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			id, code := Generate(option)
			assert.NotEmpty(t, id)
			assert.NotEmpty(t, code)

			// Validate
			valid := Validate(id, code, true)
			assert.True(t, valid)

			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestOTPMultipleValidations(t *testing.T) {
	option := NewOption()

	// Generate OTP
	id, code := Generate(option)

	// First validation without clearing
	valid := Validate(id, code, false)
	assert.True(t, valid)

	// Second validation without clearing (should still work)
	valid = Validate(id, code, false)
	assert.True(t, valid)

	// Third validation with clearing
	valid = Validate(id, code, true)
	assert.True(t, valid)

	// Fourth validation should fail (OTP was cleared)
	valid = Validate(id, code, false)
	assert.False(t, valid)
}

func TestOTPZeroValues(t *testing.T) {
	// Test with zero/empty option values
	option := Option{}
	id, code := Generate(option)

	assert.NotEmpty(t, id, "Should generate ID even with zero values")
	assert.NotEmpty(t, code, "Should generate code even with zero values")
	assert.Equal(t, 6, len(code), "Should use default length")

	// Should be numeric by default
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9', "Should default to numeric")
	}
}

func TestOTPInvalidType(t *testing.T) {
	option := NewOption()
	option.Type = "invalid_type"

	id, code := Generate(option)
	assert.NotEmpty(t, id)
	assert.NotEmpty(t, code)

	// Should fallback to numeric
	for _, c := range code {
		assert.True(t, c >= '0' && c <= '9', "Invalid type should fallback to numeric")
	}
}

func BenchmarkGenerate(b *testing.B) {
	option := NewOption()
	for i := 0; i < b.N; i++ {
		Generate(option)
	}
}

func BenchmarkValidate(b *testing.B) {
	option := NewOption()
	id, code := Generate(option)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(id, code, false)
	}
}

func BenchmarkGenerateAlphanumeric(b *testing.B) {
	option := NewOption()
	option.Type = "alphanumeric"
	option.Length = 8

	for i := 0; i < b.N; i++ {
		Generate(option)
	}
}

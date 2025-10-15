package captcha

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerate(t *testing.T) {
	// Test image captcha
	option := NewOption()
	option.Type = "image"
	option.Length = 6
	id, content := Generate(option)
	assert.NotEmpty(t, id, "Captcha ID should not be empty")
	assert.NotEmpty(t, content, "Captcha content should not be empty")
	assert.Contains(t, content, "data:image/png;base64,", "Should return base64 encoded image")
	t.Logf("Image captcha: id=%s, content_length=%d", id, len(content))

	// Test audio captcha
	option.Type = "audio"
	option.Length = 4
	option.Lang = "en"
	id2, content2 := Generate(option)
	assert.NotEmpty(t, id2, "Audio captcha ID should not be empty")
	assert.NotEmpty(t, content2, "Audio captcha content should not be empty")
	assert.Contains(t, content2, "data:audio/mp3;base64,", "Should return base64 encoded audio")
	t.Logf("Audio captcha: id=%s, content_length=%d", id2, len(content2))

	// Test math captcha (default)
	option.Type = "math"
	option.Length = 6
	id3, content3 := Generate(option)
	assert.NotEmpty(t, id3)
	assert.NotEmpty(t, content3)
	t.Logf("Math captcha: id=%s", id3)
}

func TestValidate(t *testing.T) {
	option := NewOption()
	option.Type = "math"
	option.Length = 6

	// Generate captcha
	id, _ := Generate(option)
	assert.NotEmpty(t, id)

	// Get the correct answer
	answer := Get(id)
	assert.NotEmpty(t, answer, "Should be able to retrieve captcha answer")
	t.Logf("Captcha answer: %s", answer)

	// Test valid captcha
	valid := Validate(id, answer)
	assert.True(t, valid, "Valid captcha should pass validation")

	// Test invalid captcha
	valid = Validate(id, "wrong_answer")
	assert.False(t, valid, "Invalid captcha should fail validation")

	// Test non-existent ID
	valid = Validate("non_existent_id", answer)
	assert.False(t, valid, "Non-existent ID should fail validation")
}

func TestGet(t *testing.T) {
	option := NewOption()
	option.Length = 6

	// Generate captcha
	id, _ := Generate(option)

	// Get answer
	answer := Get(id)
	assert.NotEmpty(t, answer, "Should retrieve captcha answer")
	assert.Equal(t, 6, len(answer), "Answer length should match configured length")

	// Verify the answer is correct
	valid := Validate(id, answer)
	assert.True(t, valid, "Retrieved answer should be valid")

	// Test non-existent ID
	answer2 := Get("non_existent_id")
	assert.Empty(t, answer2, "Non-existent ID should return empty string")
}

func TestValidateCloudflare(t *testing.T) {
	// Test with empty values
	valid := ValidateCloudflare("", "")
	assert.False(t, valid, "Empty token should fail validation")

	valid = ValidateCloudflare("token", "")
	assert.False(t, valid, "Empty secret should fail validation")

	// Note: Testing actual Cloudflare Turnstile requires real API keys and tokens
	// For real testing, use Cloudflare's test sitekeys:
	// https://developers.cloudflare.com/turnstile/troubleshooting/testing/
	t.Log("Cloudflare Turnstile validation requires real API keys for full testing")
}

func TestNewOption(t *testing.T) {
	option := NewOption()
	assert.Equal(t, 240, option.Width, "Default width should be 240")
	assert.Equal(t, 80, option.Height, "Default height should be 80")
	assert.Equal(t, 6, option.Length, "Default length should be 6")
	assert.Equal(t, "zh", option.Lang, "Default language should be zh")
	assert.Equal(t, "#FFFFFF", option.Background, "Default background should be #FFFFFF")
}

func TestCaptchaExpiration(t *testing.T) {
	option := NewOption()
	id, _ := Generate(option)

	// Verify captcha exists
	answer := Get(id)
	assert.NotEmpty(t, answer)

	// Validate once (this will delete it from store)
	valid := Validate(id, answer)
	assert.True(t, valid)

	// Try to get again - should be gone after validation
	answer2 := Get(id)
	assert.Empty(t, answer2, "Captcha should be deleted after validation")
}

func TestCaptchaConcurrency(t *testing.T) {
	option := NewOption()
	option.Length = 4

	// Test concurrent captcha generation and validation
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			id, _ := Generate(option)
			assert.NotEmpty(t, id)

			answer := Get(id)
			assert.NotEmpty(t, answer)

			valid := Validate(id, answer)
			assert.True(t, valid)

			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
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
	id, _ := Generate(option)
	answer := Get(id)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		Validate(id, answer)
	}
}

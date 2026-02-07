package user_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/openapi/user"
)

// TestMaskEmail tests the MaskEmail utility function
func TestMaskEmail(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		// Basic cases
		{
			"standard email",
			"john.doe@example.com",
			"j***e@example.com",
		},
		{
			"short local part (3 chars)",
			"abc@example.com",
			"a***c@example.com",
		},
		{
			"two char local part",
			"ab@example.com",
			"a***b@example.com",
		},
		{
			"single char local part",
			"a@example.com",
			"a***@example.com",
		},
		{
			"long local part",
			"very.long.email.address@example.com",
			"v***s@example.com",
		},

		// Gmail-style emails
		{
			"gmail address",
			"shadow.iqka@gmail.com",
			"s***a@gmail.com",
		},
		{
			"gmail with numbers",
			"user123@gmail.com",
			"u***3@gmail.com",
		},

		// Edge cases
		{
			"empty string",
			"",
			"",
		},
		{
			"no at sign",
			"not-an-email",
			"",
		},
		{
			"multiple at signs",
			"user@@example.com",
			"",
		},
		{
			"empty local part",
			"@example.com",
			"",
		},
		{
			"empty domain",
			"user@",
			"",
		},
		{
			"only at sign",
			"@",
			"",
		},

		// Special characters in local part
		{
			"dots in local part",
			"first.last@example.com",
			"f***t@example.com",
		},
		{
			"plus sign in local part",
			"user+tag@example.com",
			"u***g@example.com",
		},
		{
			"underscore in local part",
			"first_last@example.com",
			"f***t@example.com",
		},

		// Different domains
		{
			"subdomain email",
			"user@mail.example.com",
			"u***r@mail.example.com",
		},
		{
			"country code domain",
			"user@example.co.jp",
			"u***r@example.co.jp",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := user.MaskEmail(tc.input)
			assert.Equal(t, tc.expected, result, "MaskEmail(%q) should return %q", tc.input, tc.expected)
		})
	}
}

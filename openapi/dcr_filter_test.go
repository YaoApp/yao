package openapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// filterClientCredentials replicates the inline DCR grant filtering logic
// from oauthRegister for isolated testing.
func filterClientCredentials(grants []string) []string {
	if len(grants) == 0 {
		return grants
	}
	filtered := make([]string, 0, len(grants))
	for _, g := range grants {
		if g != "client_credentials" {
			filtered = append(filtered, g)
		}
	}
	return filtered
}

func TestFilterClientCredentials(t *testing.T) {
	tests := []struct {
		name  string
		input []string
		want  []string
	}{
		{
			name:  "nil grants passthrough",
			input: nil,
			want:  nil,
		},
		{
			name:  "empty grants passthrough",
			input: []string{},
			want:  []string{},
		},
		{
			name:  "removes client_credentials",
			input: []string{"authorization_code", "client_credentials", "refresh_token"},
			want:  []string{"authorization_code", "refresh_token"},
		},
		{
			name:  "only client_credentials",
			input: []string{"client_credentials"},
			want:  []string{},
		},
		{
			name:  "no client_credentials",
			input: []string{"authorization_code", "refresh_token"},
			want:  []string{"authorization_code", "refresh_token"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filterClientCredentials(tt.input)
			if tt.input == nil {
				assert.Nil(t, got, "nil input must produce nil output")
			} else {
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

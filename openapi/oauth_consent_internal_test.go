package openapi

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestParsePromptValues(t *testing.T) {
	values, err := parsePromptValues("")
	require.NoError(t, err)
	require.Empty(t, values)
	values, err = parsePromptValues("consent login")
	require.NoError(t, err)
	require.True(t, promptHas(values, "consent"))
	require.True(t, promptHas(values, "login"))
	_, err = parsePromptValues("none consent")
	require.Error(t, err)
	_, err = parsePromptValues("unknown")
	require.Error(t, err)
}

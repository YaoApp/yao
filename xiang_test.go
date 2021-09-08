package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCommand(t *testing.T) {
	assert.NotPanics(t, func() {
		main()
	})
}

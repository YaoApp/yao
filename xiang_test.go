package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainCMD(t *testing.T) {
	assert.NotPanics(t, func() {
		main()
	})
}

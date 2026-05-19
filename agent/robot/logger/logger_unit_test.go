//go:build unit

package logger_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/agent/robot/logger"
)

func TestNew(t *testing.T) {
	l := logger.New("test-tag")
	assert.NotNil(t, l)
}

func TestNewWithVariousTags(t *testing.T) {
	tags := []string{"telegram", "dispatcher", "message", "delivery"}
	for _, tag := range tags {
		l := logger.New(tag)
		assert.NotNil(t, l)
	}
}

func TestLogMethods(t *testing.T) {
	l := logger.New("unit-test")

	assert.NotPanics(t, func() {
		l.Trace("trace message: %s", "hello")
	})

	assert.NotPanics(t, func() {
		l.Debug("debug message: %d", 42)
	})

	assert.NotPanics(t, func() {
		l.Info("info message: %v", true)
	})

	assert.NotPanics(t, func() {
		l.Warn("warn message: %s", "caution")
	})

	assert.NotPanics(t, func() {
		l.Error("error message: %s", "failure")
	})
}

func TestRaw(t *testing.T) {
	assert.NotPanics(t, func() {
		logger.Raw("raw output\n")
	})
}

func TestIsDev(t *testing.T) {
	result := logger.IsDev()
	assert.IsType(t, true, result)
}

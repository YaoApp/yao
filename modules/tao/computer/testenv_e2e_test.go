//go:build e2e

package computer_test

import (
	"os"
	"testing"

	"github.com/yaoapp/yao/e2e/env"
)

var sharedEnvironment *env.Environment
var sharedErr error

func TestMain(m *testing.M) {
	code := m.Run()
	if sharedEnvironment != nil {
		sharedEnvironment.Close()
	}
	os.Exit(code)
}

func testEnvironment(t *testing.T) *env.Environment {
	t.Helper()
	if sharedEnvironment == nil {
		sharedEnvironment, sharedErr = env.NewShared(t)
	}
	if sharedErr != nil {
		t.Fatalf("create e2e environment: %v", sharedErr)
	}
	return sharedEnvironment
}

package kb

import (
	"testing"

	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	_, err := Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}
}

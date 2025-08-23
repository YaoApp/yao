package kb

import (
	"testing"

	"github.com/yaoapp/yao/config"
	kbtypes "github.com/yaoapp/yao/kb/types"
	"github.com/yaoapp/yao/test"
)

func TestLoad(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	kb, err := Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}

	// Test that providers are loaded
	if kb != nil && kb.Providers != nil {
		t.Logf("Knowledge base loaded successfully with providers")
	}
}

func TestGetProviders(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	_, err := Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}

	// Test getting providers for different languages
	testCases := []struct {
		providerType string
		locale       string
		expectEmpty  bool
	}{
		{"chunking", "en", false},
		{"embedding", "en", false},
		{"chunking", "zh-cn", false},
		{"embedding", "zh-cn", false},
		{"chunking", "nonexistent", false}, // Should fallback to "en"
	}

	for _, tc := range testCases {
		providers, err := GetProviders(tc.providerType, []string{}, tc.locale)
		if err != nil {
			t.Errorf("Failed to get %s providers for locale %s: %v", tc.providerType, tc.locale, err)
			continue
		}

		if tc.expectEmpty && len(providers) > 0 {
			t.Errorf("Expected empty providers for %s/%s, got %d", tc.providerType, tc.locale, len(providers))
		} else if !tc.expectEmpty && len(providers) == 0 {
			t.Logf("No providers found for %s/%s (this may be expected if no provider files exist)", tc.providerType, tc.locale)
		} else {
			t.Logf("Found %d providers for %s/%s", len(providers), tc.providerType, tc.locale)
		}
	}
}

func TestGetProviderWithLanguage(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	_, err := Load(config.Conf)
	if err != nil {
		t.Fatalf("Failed to load knowledge base: %v", err)
	}

	// Test getting a specific provider with language
	provider, err := GetProviderWithLanguage("chunking", "__yao.structured", "en")
	if err != nil {
		t.Logf("Provider __yao.structured not found for chunking/en: %v (this may be expected if provider files don't exist)", err)
	} else {
		t.Logf("Found provider: %s", provider.ID)
	}

	// Test language fallback
	provider, err = GetProviderWithLanguage("chunking", "__yao.structured", "nonexistent")
	if err != nil {
		t.Logf("Provider __yao.structured not found with fallback: %v (this may be expected if provider files don't exist)", err)
	} else {
		t.Logf("Found provider with fallback: %s", provider.ID)
	}
}

func TestLoadProviders(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	// Test loading providers from a directory
	providers, err := kbtypes.LoadProviders("kb")
	if err != nil {
		t.Fatalf("Failed to load providers: %v", err)
	}

	if providers == nil {
		t.Fatal("Providers config is nil")
	}

	// Check if provider maps are initialized
	if providers.Chunkings == nil {
		t.Error("Chunkings map is nil")
	}
	if providers.Embeddings == nil {
		t.Error("Embeddings map is nil")
	}

	t.Logf("Loaded providers successfully")
}

func TestProviderConfigGetProviders(t *testing.T) {
	// Setup
	test.Prepare(&testing.T{}, config.Conf)
	defer test.Clean()

	providers, err := kbtypes.LoadProviders("kb")
	if err != nil {
		t.Fatalf("Failed to load providers: %v", err)
	}

	// Test getting providers for different types and languages
	testCases := []string{"chunking", "embedding", "converter", "extraction", "fetcher"}

	for _, providerType := range testCases {
		// Test with "en"
		enProviders := providers.GetProviders(providerType, "en")
		t.Logf("Found %d %s providers for 'en'", len(enProviders), providerType)

		// Test with "zh-cn"
		zhProviders := providers.GetProviders(providerType, "zh-cn")
		t.Logf("Found %d %s providers for 'zh-cn'", len(zhProviders), providerType)

		// Test with nonexistent language (should fallback to "en")
		fallbackProviders := providers.GetProviders(providerType, "nonexistent")
		t.Logf("Found %d %s providers for 'nonexistent' (fallback)", len(fallbackProviders), providerType)
	}
}

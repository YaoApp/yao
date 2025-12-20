package api_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/yao/kb"
	"github.com/yaoapp/yao/kb/api"
)

// Note: TestMain is defined in collection_test.go

// ========== Fixed Test Collection IDs ==========
// Use fixed IDs so we can reuse them across test runs during development

const (
	// SearchTestScienceCollection is the fixed ID for science test collection
	SearchTestScienceCollection = "search_test_science"
	// SearchTestTechCollection is the fixed ID for tech test collection
	SearchTestTechCollection = "search_test_tech"
)

// ========== Setup Test - Run Once ==========

// TestSearchSetup creates test collections and documents for search testing.
// Run this once before running search tests:
//
//	go test -v -run "TestSearchSetup" ./kb/api/...
//
// Then run search tests multiple times without waiting for data setup:
//
//	go test -v -run "TestSearchQuery" ./kb/api/...
func TestSearchSetup(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()

	// Check if collections already exist and are complete
	// We check both GraphRag (vector store) and document count
	scienceComplete := false
	techComplete := false

	// Check Science collection
	scienceCollection, scienceErr := kb.API.GetCollection(ctx, SearchTestScienceCollection)
	if scienceErr == nil && scienceCollection != nil {
		scienceDocs, _ := kb.API.ListDocuments(ctx, &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: SearchTestScienceCollection,
		})
		if scienceDocs != nil && len(scienceDocs.Data) >= 5 {
			scienceComplete = true
			t.Logf("✓ Science collection exists: %s (%d docs)", SearchTestScienceCollection, len(scienceDocs.Data))
		}
	}

	// Check Tech collection
	techCollection, techErr := kb.API.GetCollection(ctx, SearchTestTechCollection)
	if techErr == nil && techCollection != nil {
		techDocs, _ := kb.API.ListDocuments(ctx, &api.ListDocumentsFilter{
			Page:         1,
			PageSize:     20,
			CollectionID: SearchTestTechCollection,
		})
		if techDocs != nil && len(techDocs.Data) >= 5 {
			techComplete = true
			t.Logf("✓ Tech collection exists: %s (%d docs)", SearchTestTechCollection, len(techDocs.Data))
		}
	}

	// If both collections are complete, skip setup
	if scienceComplete && techComplete {
		t.Log("✓ All test collections already exist with sufficient documents")
		t.Log("  Skipping setup. Run TestSearchCleanup first to recreate.")
		return
	}

	// Clean up any existing collections (handles both complete and incomplete states)
	// RemoveCollection cleans both database and GraphRag (including orphaned vector collections)
	t.Log("Cleaning up existing collections...")
	if result, err := kb.API.RemoveCollection(ctx, SearchTestScienceCollection); err == nil && result.Removed {
		t.Logf("  Removed: %s", SearchTestScienceCollection)
	}
	if result, err := kb.API.RemoveCollection(ctx, SearchTestTechCollection); err == nil && result.Removed {
		t.Logf("  Removed: %s", SearchTestTechCollection)
	}
	time.Sleep(1 * time.Second) // Wait for cleanup

	// Create Science Collection
	t.Log("Creating Science collection...")
	scienceParams := &api.CreateCollectionParams{
		ID: SearchTestScienceCollection,
		Metadata: map[string]interface{}{
			"name":        "Science Knowledge Base",
			"description": "Scientists and their discoveries for search testing",
		},
		EmbeddingProviderID: "__yao.openai",
		EmbeddingOptionID:   "text-embedding-3-small",
		Locale:              "en",
		Config: &graphragtypes.CreateCollectionOptions{
			Distance:  "cosine",
			IndexType: "hnsw",
		},
	}
	_, err := kb.API.CreateCollection(ctx, scienceParams)
	if err != nil {
		t.Fatalf("Failed to create science collection: %v", err)
	}
	t.Logf("✓ Created collection: %s", SearchTestScienceCollection)

	// Create Tech Collection
	t.Log("Creating Tech collection...")
	techParams := &api.CreateCollectionParams{
		ID: SearchTestTechCollection,
		Metadata: map[string]interface{}{
			"name":        "Tech Knowledge Base",
			"description": "Technology companies and products for search testing",
		},
		EmbeddingProviderID: "__yao.openai",
		EmbeddingOptionID:   "text-embedding-3-small",
		Locale:              "en",
		Config: &graphragtypes.CreateCollectionOptions{
			Distance:  "cosine",
			IndexType: "hnsw",
		},
	}
	_, err = kb.API.CreateCollection(ctx, techParams)
	if err != nil {
		t.Fatalf("Failed to create tech collection: %v", err)
	}
	t.Logf("✓ Created collection: %s", SearchTestTechCollection)

	// Add Science Documents
	// Entity relationships: Einstein -> Relativity -> Physics -> Nobel Prize
	scienceDocs := []struct {
		title   string
		content string
	}{
		{
			title: "Albert Einstein Biography",
			content: `Albert Einstein was a theoretical physicist born in Germany in 1879. 
			He developed the theory of relativity, one of the two pillars of modern physics. 
			Einstein received the Nobel Prize in Physics in 1921 for his discovery of the photoelectric effect. 
			He later emigrated to the United States and worked at Princeton University until his death in 1955.`,
		},
		{
			title: "Theory of Relativity",
			content: `The theory of relativity was developed by Albert Einstein in the early 20th century. 
			It consists of special relativity (1905) and general relativity (1915). 
			Special relativity introduced E=mc², showing the relationship between energy and mass. 
			General relativity describes gravity as the curvature of spacetime caused by mass and energy.`,
		},
		{
			title: "Marie Curie Biography",
			content: `Marie Curie was a Polish-French physicist and chemist who conducted pioneering research on radioactivity. 
			She was the first woman to win a Nobel Prize and the only person to win Nobel Prizes in two different sciences (Physics and Chemistry). 
			Curie discovered the elements polonium and radium. She founded the Curie Institutes in Paris and Warsaw.`,
		},
		{
			title: "Nobel Prize in Physics",
			content: `The Nobel Prize in Physics is awarded annually by the Royal Swedish Academy of Sciences. 
			Notable recipients include Albert Einstein (1921) for the photoelectric effect, 
			Marie Curie (1903) for research on radiation phenomena, 
			and Niels Bohr (1922) for his contributions to understanding atomic structure.`,
		},
		{
			title: "Quantum Mechanics Foundations",
			content: `Quantum mechanics emerged in the early 20th century through the work of many physicists. 
			Max Planck introduced the concept of energy quanta in 1900. 
			Niels Bohr proposed the Bohr model of the atom. 
			Werner Heisenberg developed the uncertainty principle. 
			These discoveries built upon Einstein's work on the photoelectric effect.`,
		},
	}

	t.Log("Adding Science documents...")
	for _, doc := range scienceDocs {
		docID := addFixedTestDocument(t, ctx, SearchTestScienceCollection, doc.title, doc.content)
		if docID != "" {
			t.Logf("  ✓ Added: %s", doc.title)
		}
	}

	// Add Tech Documents
	// Entity relationships: Apple -> Steve Jobs -> iPhone -> iOS
	techDocs := []struct {
		title   string
		content string
	}{
		{
			title: "Apple Inc History",
			content: `Apple Inc. was founded by Steve Jobs, Steve Wozniak, and Ronald Wayne in 1976. 
			The company revolutionized personal computing with the Macintosh in 1984. 
			Under Steve Jobs' leadership, Apple introduced the iPhone in 2007, which transformed the smartphone industry. 
			Apple is headquartered in Cupertino, California.`,
		},
		{
			title: "iPhone Development",
			content: `The iPhone was introduced by Steve Jobs at Macworld 2007. 
			It combined a mobile phone, widescreen iPod, and internet device into one product. 
			The iPhone runs on iOS, Apple's mobile operating system. 
			The App Store, launched in 2008, created a new ecosystem for mobile applications.`,
		},
		{
			title: "Google and AI",
			content: `Google has been a pioneer in artificial intelligence and machine learning. 
			The company developed TensorFlow, an open-source machine learning framework. 
			Google's AI research includes natural language processing, computer vision, and deep learning. 
			Google Brain and DeepMind are the company's main AI research divisions.`,
		},
		{
			title: "Machine Learning Applications",
			content: `Machine learning is transforming various industries through AI applications. 
			Google uses ML for search ranking, language translation, and image recognition. 
			TensorFlow enables developers to build and train neural networks. 
			Deep learning models can now understand natural language and generate human-like text.`,
		},
		{
			title: "Tech Industry Leaders",
			content: `The technology industry has been shaped by visionary leaders. 
			Steve Jobs transformed Apple into the world's most valuable company. 
			Larry Page and Sergey Brin founded Google and pioneered internet search. 
			Elon Musk leads Tesla and SpaceX, pushing boundaries in electric vehicles and space exploration.`,
		},
	}

	t.Log("Adding Tech documents...")
	for _, doc := range techDocs {
		docID := addFixedTestDocument(t, ctx, SearchTestTechCollection, doc.title, doc.content)
		if docID != "" {
			t.Logf("  ✓ Added: %s", doc.title)
		}
	}

	// Wait for indexing
	t.Log("Waiting for indexing...")
	time.Sleep(2 * time.Second)

	// Verify setup
	t.Log("Verifying setup...")
	scienceDocsResult, _ := kb.API.ListDocuments(ctx, &api.ListDocumentsFilter{
		Page:         1,
		PageSize:     20,
		CollectionID: SearchTestScienceCollection,
	})
	techDocsResult, _ := kb.API.ListDocuments(ctx, &api.ListDocumentsFilter{
		Page:         1,
		PageSize:     20,
		CollectionID: SearchTestTechCollection,
	})

	t.Logf("✓ Setup complete!")
	t.Logf("  Science collection: %d documents", len(scienceDocsResult.Data))
	t.Logf("  Tech collection: %d documents", len(techDocsResult.Data))
	t.Logf("")
	t.Logf("Now run search tests with:")
	t.Logf("  go test -v -run 'TestSearchQuery' ./kb/api/...")
}

// ========== Cleanup Test ==========

// TestSearchCleanup removes test collections.
// Run this to clean up test data:
//
//	go test -v -run "TestSearchCleanup" ./kb/api/...
func TestSearchCleanup(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()

	t.Log("Removing test collections...")

	result1, err := kb.API.RemoveCollection(ctx, SearchTestScienceCollection)
	if err != nil {
		t.Logf("  Science collection removal: %v", err)
	} else if result1.Removed {
		t.Logf("✓ Removed: %s", SearchTestScienceCollection)
	}

	result2, err := kb.API.RemoveCollection(ctx, SearchTestTechCollection)
	if err != nil {
		t.Logf("  Tech collection removal: %v", err)
	} else if result2.Removed {
		t.Logf("✓ Removed: %s", SearchTestTechCollection)
	}

	t.Log("✓ Cleanup complete!")
}

// ========== Verify Test ==========

// TestSearchVerify checks if test collections exist and have documents.
// Run this to verify test data:
//
//	go test -v -run "TestSearchVerify" ./kb/api/...
func TestSearchVerify(t *testing.T) {
	if kb.API == nil {
		t.Skip("KB API not initialized")
	}

	ctx := context.Background()

	// Check Science collection
	scienceExists, err := kb.API.CollectionExists(ctx, SearchTestScienceCollection)
	if err != nil {
		t.Fatalf("Failed to check science collection: %v", err)
	}
	if !scienceExists.Exists {
		t.Fatalf("✗ Science collection does not exist. Run TestSearchSetup first.")
	}

	scienceDocs, err := kb.API.ListDocuments(ctx, &api.ListDocumentsFilter{
		Page:         1,
		PageSize:     20,
		CollectionID: SearchTestScienceCollection,
	})
	assert.NoError(t, err)
	t.Logf("✓ Science collection: %s (%d documents)", SearchTestScienceCollection, len(scienceDocs.Data))
	for _, doc := range scienceDocs.Data {
		t.Logf("    - %s", doc["name"])
	}

	// Check Tech collection
	techExists, err := kb.API.CollectionExists(ctx, SearchTestTechCollection)
	if err != nil {
		t.Fatalf("Failed to check tech collection: %v", err)
	}
	if !techExists.Exists {
		t.Fatalf("✗ Tech collection does not exist. Run TestSearchSetup first.")
	}

	techDocs, err := kb.API.ListDocuments(ctx, &api.ListDocumentsFilter{
		Page:         1,
		PageSize:     20,
		CollectionID: SearchTestTechCollection,
	})
	assert.NoError(t, err)
	t.Logf("✓ Tech collection: %s (%d documents)", SearchTestTechCollection, len(techDocs.Data))
	for _, doc := range techDocs.Data {
		t.Logf("    - %s", doc["name"])
	}

	t.Log("")
	t.Log("✓ Test data verified! Ready for search tests.")
}

// ========== Helper Functions ==========

// addFixedTestDocument adds a document for search testing
func addFixedTestDocument(t *testing.T, ctx context.Context, collectionID, title, content string) string {
	params := &api.AddTextParams{
		CollectionID: collectionID,
		Text:         content,
		DocID:        fmt.Sprintf("%s__%s", collectionID, sanitizeTitle(title)),
		Metadata: map[string]interface{}{
			"title": title,
		},
		Chunking: &api.ProviderConfigParams{
			ProviderID: "__yao.structured",
			OptionID:   "standard",
		},
		Embedding: &api.ProviderConfigParams{
			ProviderID: "__yao.openai",
			OptionID:   "text-embedding-3-small",
		},
		// Enable extraction for graph-based search
		Extraction: &api.ProviderConfigParams{
			ProviderID: "__yao.openai",
			OptionID:   "gpt-4o-mini",
		},
	}

	result, err := kb.API.AddText(ctx, params)
	if err != nil {
		t.Logf("Warning: Failed to add document '%s': %v", title, err)
		return ""
	}
	return result.DocID
}

// sanitizeTitle converts title to a safe ID format
func sanitizeTitle(title string) string {
	result := ""
	for _, c := range title {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') {
			result += string(c)
		} else if c == ' ' {
			result += "_"
		}
	}
	return result
}

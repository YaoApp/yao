package api

import (
	"context"
	"fmt"

	graphragtypes "github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/kun/maps"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// updateDocumentAfterProcessing updates document status and segment count after processing
func (instance *KBInstance) updateDocumentAfterProcessing(ctx context.Context, docID, collectionID string) {
	// Update status to completed
	if err := instance.Config.UpdateDocument(docID, maps.MapStrAny{"status": "completed"}); err != nil {
		log.Error("Failed to update document status to completed: %v", err)
	}

	// Update segment count
	if segmentCount, err := instance.GraphRag.SegmentCount(ctx, docID); err != nil {
		log.Error("Failed to get segment count for document %s: %v", docID, err)
	} else {
		if err := instance.Config.UpdateSegmentCount(docID, segmentCount); err != nil {
			log.Error("Failed to update segment count for document %s: %v", docID, err)
		}
	}

	// Update document count for collection
	if err := instance.updateDocumentCountWithSync(ctx, collectionID); err != nil {
		log.Error("Failed to update document count for collection %s: %v", collectionID, err)
	}
}

// updateDocumentCountWithSync updates document count and syncs to GraphRag
func (instance *KBInstance) updateDocumentCountWithSync(ctx context.Context, collectionID string) error {
	// Get document count
	count, err := instance.Config.DocumentCount(collectionID)
	if err != nil {
		return fmt.Errorf("failed to get document count: %w", err)
	}

	// Update collection in database
	if err := instance.Config.UpdateCollection(collectionID, maps.MapStrAny{"document_count": count}); err != nil {
		return fmt.Errorf("failed to update collection document count: %w", err)
	}

	// Sync to GraphRag
	metadata := map[string]interface{}{"document_count": count}
	if err := instance.GraphRag.UpdateCollectionMetadata(ctx, collectionID, metadata); err != nil {
		return fmt.Errorf("failed to sync document count to GraphRag: %w", err)
	}

	return nil
}

// toUpsertOptions converts provider config params to UpsertOptions
func (instance *KBInstance) toUpsertOptions(docID, collectionID, locale, filename, contentType string, chunking, embedding, extraction, fetcher, converter *ProviderConfigParams) (*graphragtypes.UpsertOptions, error) {
	if locale == "" {
		locale = DefaultLocale
	}

	options := &graphragtypes.UpsertOptions{
		CollectionID: collectionID,
		DocID:        docID,
	}

	// Create chunking provider
	if chunking != nil {
		chunkingOption, err := instance.getProviderOption("chunking", chunking.ProviderID, chunking.OptionID, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve chunking provider: %w", err)
		}

		chunkingProvider, err := factory.MakeChunking(chunking.ProviderID, chunkingOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create chunking provider: %w", err)
		}
		options.Chunking = chunkingProvider

		chunkingOpts, err := factory.ChunkingOptions(chunking.ProviderID, chunkingOption)
		if err != nil {
			return nil, fmt.Errorf("failed to get chunking options: %w", err)
		}
		options.ChunkingOptions = chunkingOpts
	}

	// Create embedding provider
	if embedding != nil {
		embeddingOption, err := instance.getProviderOption("embedding", embedding.ProviderID, embedding.OptionID, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve embedding provider: %w", err)
		}

		embeddingProvider, err := factory.MakeEmbedding(embedding.ProviderID, embeddingOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create embedding provider: %w", err)
		}
		options.Embedding = embeddingProvider
	}

	// Create extraction provider (optional, but required if graph is enabled)
	// If extraction is not provided, try to use the default extraction provider
	if extraction != nil {
		extractionOption, err := instance.getProviderOption("extraction", extraction.ProviderID, extraction.OptionID, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve extraction provider: %w", err)
		}

		extractionProvider, err := factory.MakeExtraction(extraction.ProviderID, extractionOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create extraction provider: %w", err)
		}
		options.Extraction = extractionProvider
	} else {
		// Try to get default extraction provider to avoid gou's DetectExtractor with hardcoded connector
		defaultExtraction := instance.getDefaultProvider("extraction", locale)
		if defaultExtraction != nil {
			extractionOption, err := instance.getProviderOption("extraction", defaultExtraction.ID, "", locale)
			if err == nil {
				extractionProvider, err := factory.MakeExtraction(defaultExtraction.ID, extractionOption)
				if err == nil {
					options.Extraction = extractionProvider
				}
			}
		}
	}

	// Create fetcher provider (optional)
	if fetcher != nil {
		fetcherOption, err := instance.getProviderOption("fetcher", fetcher.ProviderID, fetcher.OptionID, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve fetcher provider: %w", err)
		}

		fetcherProvider, err := factory.MakeFetcher(fetcher.ProviderID, fetcherOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create fetcher provider: %w", err)
		}
		options.Fetcher = fetcherProvider
	}

	// Create converter provider (optional or auto-detect)
	if converter != nil {
		converterOption, err := instance.getProviderOption("converter", converter.ProviderID, converter.OptionID, locale)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve converter provider: %w", err)
		}

		converterProvider, err := factory.MakeConverter(converter.ProviderID, converterOption)
		if err != nil {
			return nil, fmt.Errorf("failed to create converter provider: %w", err)
		}
		options.Converter = converterProvider
	} else if filename != "" || contentType != "" {
		// Auto-detect converter
		matched, converterID, err := factory.AutoDetectConverter(filename, contentType)
		if err != nil {
			return nil, fmt.Errorf("failed to auto-detect converter: %w", err)
		}

		if matched {
			converterOption, err := instance.getProviderOption("converter", converterID, "", locale)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve auto-detected converter provider: %w", err)
			}

			converterProvider, err := factory.MakeConverter(converterID, converterOption)
			if err != nil {
				return nil, fmt.Errorf("failed to create auto-detected converter provider: %w", err)
			}
			options.Converter = converterProvider
		}
	}

	return options, nil
}

// getProviderOption gets a provider option by provider type, ID and option ID
func (instance *KBInstance) getProviderOption(providerType, providerID, optionID, locale string) (*kbtypes.ProviderOption, error) {
	provider, err := instance.Providers.GetProvider(providerType, providerID, locale)
	if err != nil {
		return nil, fmt.Errorf("provider %s not found for locale %s: %w", providerID, locale, err)
	}

	if optionID != "" {
		option, exists := provider.GetOption(optionID)
		if !exists {
			return nil, fmt.Errorf("option %s not found in provider %s", optionID, providerID)
		}
		return option, nil
	}

	// Return default option
	if provider.Options != nil {
		for _, option := range provider.Options {
			if option.Default {
				return option, nil
			}
		}
		if len(provider.Options) > 0 {
			return provider.Options[0], nil
		}
	}

	return nil, fmt.Errorf("no option specified and no default option found for provider %s", providerID)
}

// getDefaultProvider returns the default provider for a given type and locale
func (instance *KBInstance) getDefaultProvider(providerType, locale string) *kbtypes.Provider {
	if instance.Providers == nil {
		return nil
	}

	providers := instance.Providers.GetProviders(providerType, locale)
	if len(providers) == 0 {
		return nil
	}

	// Find provider with default=true
	for _, provider := range providers {
		if provider.Default {
			return provider
		}
	}

	// Return first provider if no default is set
	return providers[0]
}

// getJobOptions returns job options with defaults
func getJobOptions(job *JobOptionsParams, defaultName, defaultDescription, defaultIcon, defaultCategory string) (string, string, string, string) {
	name := defaultName
	description := defaultDescription
	icon := defaultIcon
	category := defaultCategory

	if job != nil {
		if job.Name != "" {
			name = job.Name
		}
		if job.Description != "" {
			description = job.Description
		}
		if job.Icon != "" {
			icon = job.Icon
		}
		if job.Category != "" {
			category = job.Category
		}
	}

	return name, description, icon, category
}

// addBaseFieldsFromParams adds base fields from parameters to document data
func addBaseFieldsFromParams(data map[string]interface{}, locale string, metadata map[string]interface{}, chunking, embedding, extraction, fetcher, converter *ProviderConfigParams) {
	if locale != "" {
		data["locale"] = locale
	}

	// Extract fields from metadata
	if metadata != nil {
		if description, ok := metadata["description"]; ok && description != nil {
			data["description"] = description
		}
		if cover, ok := metadata["cover"]; ok && cover != nil {
			data["cover"] = cover
		}
		if tags, ok := metadata["tags"]; ok && tags != nil {
			data["tags"] = tags
		}
		if name, ok := metadata["name"]; ok && name != nil {
			data["name"] = name
		}
	}

	// Add provider configurations
	if converter != nil {
		data["converter_provider_id"] = converter.ProviderID
		if converter.OptionID != "" {
			data["converter_option_id"] = converter.OptionID
		}
		if converter.Properties != nil {
			data["converter_properties"] = converter.Properties
		}
	}
	if fetcher != nil {
		data["fetcher_provider_id"] = fetcher.ProviderID
		if fetcher.OptionID != "" {
			data["fetcher_option_id"] = fetcher.OptionID
		}
		if fetcher.Properties != nil {
			data["fetcher_properties"] = fetcher.Properties
		}
	}
	if chunking != nil {
		data["chunking_provider_id"] = chunking.ProviderID
		if chunking.OptionID != "" {
			data["chunking_option_id"] = chunking.OptionID
		}
		if chunking.Properties != nil {
			data["chunking_properties"] = chunking.Properties
		}
	}
	if embedding != nil {
		data["embedding_provider_id"] = embedding.ProviderID
		if embedding.OptionID != "" {
			data["embedding_option_id"] = embedding.OptionID
		}
		if embedding.Properties != nil {
			data["embedding_properties"] = embedding.Properties
		}
	}
	if extraction != nil {
		data["extraction_provider_id"] = extraction.ProviderID
		if extraction.OptionID != "" {
			data["extraction_option_id"] = extraction.OptionID
		}
		if extraction.Properties != nil {
			data["extraction_properties"] = extraction.Properties
		}
	}
}

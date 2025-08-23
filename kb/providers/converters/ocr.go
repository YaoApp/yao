package converters

import (
	"fmt"
	"strings"

	"github.com/yaoapp/gou/graphrag/converter"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/gou/pdf"
	"github.com/yaoapp/yao/kb/providers/factory"
	kbtypes "github.com/yaoapp/yao/kb/types"
)

// OCR is a converter provider for ocr files, support pdf, image.
type OCR struct {
	Autodetect    []string `json:"autodetect" yaml:"autodetect"`         // Optional, default is empty, if not set, will not use autodetect
	MatchPriority int      `json:"match_priority" yaml:"match_priority"` // Optional, default is 0, the higher the number, the higher the priority
}

// Make creates a new OCR converter
func (ocr *OCR) Make(option *kbtypes.ProviderOption) (types.Converter, error) {
	// Start with default values
	ocrOption := converter.OCROption{
		Vision:         nil,                    // Will be set from option
		Mode:           converter.OCRModeQueue, // Default to queue mode
		MaxConcurrency: 4,                      // Default 4 concurrent processes
		CompressSize:   512,                    // Default compression size
		ForceImageMode: false,                  // Default don't force image mode
		PDFTool:        pdf.ToolPdftoppm,       // Default PDF tool
		PDFToolPath:    "",                     // Use system default
		PDFDPI:         150,                    // Default DPI
		PDFFormat:      "png",                  // Default format
		PDFQuality:     90,                     // Default JPEG quality
	}

	// Use global PDF configuration as defaults if available
	if globalPDF := kbtypes.GetGlobalPDF(); globalPDF != nil {
		// Map PDF configuration to OCR options
		if globalPDF.ConvertTool != "" {
			switch globalPDF.ConvertTool {
			case "pdftoppm":
				ocrOption.PDFTool = pdf.ToolPdftoppm
			case "mutool":
				ocrOption.PDFTool = pdf.ToolMutool
			case "imagemagick", "convert":
				ocrOption.PDFTool = pdf.ToolImageMagick
			}
		}

		if globalPDF.ToolPath != "" {
			ocrOption.PDFToolPath = globalPDF.ToolPath
		}
	}

	// Extract values from Properties map to override defaults
	if option != nil && option.Properties != nil {
		if mode, ok := option.Properties["mode"]; ok {
			if modeStr, ok := mode.(string); ok {
				switch modeStr {
				case "queue":
					ocrOption.Mode = converter.OCRModeQueue
				case "concurrent":
					ocrOption.Mode = converter.OCRModeConcurrent
				}
			}
		}

		if maxConcurrency, ok := option.Properties["max_concurrency"]; ok {
			if maxInt, ok := maxConcurrency.(int); ok {
				ocrOption.MaxConcurrency = maxInt
			} else if maxFloat, ok := maxConcurrency.(float64); ok {
				ocrOption.MaxConcurrency = int(maxFloat)
			}
		}

		if compressSize, ok := option.Properties["compress_size"]; ok {
			if sizeInt, ok := compressSize.(int); ok {
				ocrOption.CompressSize = int64(sizeInt)
			} else if sizeFloat, ok := compressSize.(float64); ok {
				ocrOption.CompressSize = int64(sizeFloat)
			}
		}

		if forceImageMode, ok := option.Properties["force_image_mode"]; ok {
			if forceBool, ok := forceImageMode.(bool); ok {
				ocrOption.ForceImageMode = forceBool
			}
		}

		if pdfTool, ok := option.Properties["pdf_tool"]; ok {
			if pdfToolStr, ok := pdfTool.(string); ok {
				switch pdfToolStr {
				case "pdftoppm":
					ocrOption.PDFTool = pdf.ToolPdftoppm
				case "mutool":
					ocrOption.PDFTool = pdf.ToolMutool
				case "imagemagick", "convert":
					ocrOption.PDFTool = pdf.ToolImageMagick
				}
			}
		}

		if pdfToolPath, ok := option.Properties["pdf_tool_path"]; ok {
			if pathStr, ok := pdfToolPath.(string); ok {
				ocrOption.PDFToolPath = pathStr
			}
		}

		if pdfDPI, ok := option.Properties["pdf_dpi"]; ok {
			if dpiInt, ok := pdfDPI.(int); ok {
				ocrOption.PDFDPI = dpiInt
			} else if dpiFloat, ok := pdfDPI.(float64); ok {
				ocrOption.PDFDPI = int(dpiFloat)
			}
		}

		if pdfFormat, ok := option.Properties["pdf_format"]; ok {
			if formatStr, ok := pdfFormat.(string); ok {
				ocrOption.PDFFormat = formatStr
			}
		}

		if pdfQuality, ok := option.Properties["pdf_quality"]; ok {
			if qualityInt, ok := pdfQuality.(int); ok {
				ocrOption.PDFQuality = qualityInt
			} else if qualityFloat, ok := pdfQuality.(float64); ok {
				ocrOption.PDFQuality = int(qualityFloat)
			}
		}

		// Handle nested vision converter
		if vision, ok := option.Properties["vision"]; ok {
			visionConverter, err := parseNestedConverter(vision)
			if err != nil {
				return nil, fmt.Errorf("failed to parse vision converter: %w", err)
			}
			ocrOption.Vision = visionConverter
		}
	}

	// Vision converter is required
	if ocrOption.Vision == nil {
		return nil, fmt.Errorf("vision converter is required for OCR processing")
	}

	return converter.NewOCR(ocrOption)
}

// AutoDetect detects the converter based on the filename and content types
func (ocr *OCR) AutoDetect(filename, contentTypes string) (bool, int, error) {
	// If autodetect is empty, return false
	if ocr.Autodetect == nil {
		return false, 0, nil
	}

	// Check if the filename matches the autodetect
	for _, autodetect := range ocr.Autodetect {
		if strings.HasSuffix(filename, autodetect) {
			return true, ocr.MatchPriority, nil
		}

		// Check if the content types matches the autodetect
		if strings.Contains(contentTypes, autodetect) {
			return true, ocr.MatchPriority, nil
		}
	}

	return false, 0, nil
}

// Schema returns the schema for the OCR converter
func (ocr *OCR) Schema(provider *kbtypes.Provider, locale string) (*kbtypes.ProviderSchema, error) {
	return factory.GetSchemaFromBindata(factory.ProviderTypeConverter, "ocr", locale)
}

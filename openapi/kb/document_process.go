package kb

import (
	"context"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/kb"
)

// ProcessGetDocumentsContent retrieves content for documents by IDs via Yao process
// Process: kb.documents.getcontents
//
// Args[0]: document_ids (string | []string) - Document ID or list of document IDs
//
// Returns: []map containing document_id, name, content, content_type, etc. for each document
//
// Example:
//
//	// Single document
//	Process("kb.documents.getcontents", "doc_id_123")
//
//	// Multiple documents
//	Process("kb.documents.getcontents", ["doc_id_1", "doc_id_2", "doc_id_3"])
func ProcessGetDocumentsContent(process *process.Process) interface{} {
	process.ValidateArgNums(1)

	if kb.API == nil {
		exception.New("Knowledge base not initialized", 500).Throw()
	}

	// Support both single string and array of strings
	var docIDs []string
	arg := process.Args[0]

	switch v := arg.(type) {
	case string:
		if v == "" {
			exception.New("Document ID is required", 400).Throw()
		}
		docIDs = []string{v}
	case []string:
		docIDs = v
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok && s != "" {
				docIDs = append(docIDs, s)
			}
		}
	default:
		exception.New("Document IDs must be a string or array of strings", 400).Throw()
	}

	if len(docIDs) == 0 {
		exception.New("Document IDs are required", 400).Throw()
	}

	ctx := process.Context
	if ctx == nil {
		ctx = context.Background()
	}

	// Call KB API to get documents content
	results, err := kb.API.GetDocumentsContent(ctx, docIDs)
	if err != nil {
		exception.New("Failed to get documents content: "+err.Error(), 500).Throw()
	}

	return results
}

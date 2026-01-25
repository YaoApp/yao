package attachment

import (
	"context"
	"encoding/base64"
	"fmt"
	"mime"
	"mime/multipart"
	"net/textproto"
	"path/filepath"
	"strings"

	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

// Init registers all attachment processes
func Init() {
	process.RegisterGroup("attachment", map[string]process.Handler{
		"Save":     processSave,
		"Read":     processRead,
		"Info":     processInfo,
		"List":     processList,
		"Delete":   processDelete,
		"Exists":   processExists,
		"URL":      processURL,
		"SaveText": processSaveText,
		"GetText":  processGetText,
	})
}

// processSave saves a file from base64 data URI
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - content: string - base64 data URI (e.g., "data:image/png;base64,xxxx") or plain base64
//   - filename: string (optional) - original filename
//   - option: map (optional) - upload options (groups, gzip, compress_image, public, share)
//
// Returns: *File - uploaded file info
//
// Example:
//
//	Process("attachment.Save", "default", "data:image/png;base64,iVBORw0KGgo...", "photo.png")
//	Process("attachment.Save", "default", "data:text/plain;base64,SGVsbG8=", "hello.txt", {"share": "team"})
func processSave(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	uploaderID := p.ArgsString(0)
	content := p.ArgsString(1)

	// Get manager
	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	// Parse data URI and decode content
	contentType, data, err := parseDataURI(content)
	if err != nil {
		return fmt.Errorf("failed to parse content: %v", err)
	}

	// Get filename from args or generate from content type
	filename := ""
	if p.NumOfArgs() > 2 {
		filename = p.ArgsString(2)
	}
	if filename == "" {
		filename = generateFilename(contentType)
	}

	// Create file header
	header := createFileHeader(filename, contentType, int64(len(data)))

	// Create upload options
	option := createUploadOption(p, filename)

	// Upload
	ctx := context.Background()
	file, err := manager.Upload(ctx, header, strings.NewReader(string(data)), option)
	if err != nil {
		return fmt.Errorf("failed to save file: %v", err)
	}

	return file
}

// processRead reads file content as base64 data URI
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - fileID: string - the file ID
//
// Returns: string - base64 data URI (e.g., "data:image/png;base64,xxxx")
//
// Example:
//
//	const dataURI = Process("attachment.Read", "default", "abc123")
func processRead(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	uploaderID := p.ArgsString(0)
	fileID := p.ArgsString(1)

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	ctx := context.Background()

	// Get file info for content type and permission check
	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// Check permission
	if err := checkFilePermission(p, fileInfo, true); err != nil {
		return err
	}

	// Read content as base64
	base64Data, err := manager.ReadBase64(ctx, fileID)
	if err != nil {
		return fmt.Errorf("failed to read file: %v", err)
	}

	// Return as data URI
	return fmt.Sprintf("data:%s;base64,%s", fileInfo.ContentType, base64Data)
}

// processInfo gets file information
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - fileID: string - the file ID
//
// Returns: *File - file info
func processInfo(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	uploaderID := p.ArgsString(0)
	fileID := p.ArgsString(1)

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	ctx := context.Background()
	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// Check permission
	if err := checkFilePermission(p, fileInfo, true); err != nil {
		return err
	}

	return fileInfo
}

// processList lists files with pagination and filtering
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - option: map (optional) - list options (page, page_size, filters, order_by, select)
//
// Returns: *ListResult - paginated file list
func processList(p *process.Process) interface{} {
	p.ValidateArgNums(1)

	uploaderID := p.ArgsString(0)

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	// Parse list options
	listOption := ListOption{
		Page:     1,
		PageSize: 20,
	}

	if p.NumOfArgs() > 1 {
		optionRaw := p.ArgsMap(1)
		option := maps.MapOf(optionRaw).Dot()

		if page := any.Of(option.Get("page")).CInt(); page > 0 {
			listOption.Page = page
		}
		if pageSize := any.Of(option.Get("page_size")).CInt(); pageSize > 0 && pageSize <= 100 {
			listOption.PageSize = pageSize
		}
		if filters, ok := option.Get("filters").(map[string]interface{}); ok {
			listOption.Filters = filters
		}
		if orderBy, ok := option.Get("order_by").(string); ok {
			listOption.OrderBy = orderBy
		}
		if selectFields, ok := option.Get("select").([]interface{}); ok {
			for _, field := range selectFields {
				if f, ok := field.(string); ok {
					listOption.Select = append(listOption.Select, f)
				}
			}
		}
	}

	// Always filter by uploader
	if listOption.Filters == nil {
		listOption.Filters = make(map[string]interface{})
	}
	listOption.Filters["uploader"] = uploaderID

	// Add permission-based filtering
	listOption.Wheres = append(listOption.Wheres, model.QueryWhere{
		Column: "uploader",
		Value:  uploaderID,
	})
	listOption.Wheres = append(listOption.Wheres, buildPermissionWheres(p)...)

	ctx := context.Background()
	result, err := manager.List(ctx, listOption)
	if err != nil {
		return fmt.Errorf("failed to list files: %v", err)
	}

	return result
}

// processDelete deletes a file
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - fileID: string - the file ID
//
// Returns: bool - success
func processDelete(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	uploaderID := p.ArgsString(0)
	fileID := p.ArgsString(1)

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	ctx := context.Background()

	// Get file info first
	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// Check write permission
	if err := checkFilePermission(p, fileInfo, false); err != nil {
		return err
	}

	// Delete file
	if err := manager.Delete(ctx, fileID); err != nil {
		return fmt.Errorf("failed to delete file: %v", err)
	}

	return true
}

// processExists checks if file exists
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - fileID: string - the file ID
//
// Returns: bool
func processExists(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	uploaderID := p.ArgsString(0)
	fileID := p.ArgsString(1)

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	ctx := context.Background()
	return manager.Exists(ctx, fileID)
}

// processURL gets file URL
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - fileID: string - the file ID
//
// Returns: string - file URL
func processURL(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	uploaderID := p.ArgsString(0)
	fileID := p.ArgsString(1)

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	ctx := context.Background()

	// Get file info for permission check
	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// Check permission
	if err := checkFilePermission(p, fileInfo, true); err != nil {
		return err
	}

	return manager.storage.URL(ctx, fileID)
}

// processSaveText saves parsed text content for a file
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - fileID: string - the file ID
//   - text: string - the text content to save
//
// Returns: bool - success
func processSaveText(p *process.Process) interface{} {
	p.ValidateArgNums(3)

	uploaderID := p.ArgsString(0)
	fileID := p.ArgsString(1)
	text := p.ArgsString(2)

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	ctx := context.Background()

	// Get file info first to check write permission
	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// Check write permission
	if err := checkFilePermission(p, fileInfo, false); err != nil {
		return err
	}

	if err := manager.SaveText(ctx, fileID, text); err != nil {
		return fmt.Errorf("failed to save text: %v", err)
	}

	return true
}

// processGetText gets parsed text content for a file
// Args:
//   - uploaderID: string - the uploader/manager ID
//   - fileID: string - the file ID
//   - fullContent: bool (optional) - whether to get full content (default: false, returns preview)
//
// Returns: string - text content
func processGetText(p *process.Process) interface{} {
	p.ValidateArgNums(2)

	uploaderID := p.ArgsString(0)
	fileID := p.ArgsString(1)

	fullContent := false
	if p.NumOfArgs() > 2 {
		fullContent = p.ArgsBool(2)
	}

	manager, exists := Managers[uploaderID]
	if !exists {
		return fmt.Errorf("uploader not found: %s", uploaderID)
	}

	ctx := context.Background()

	// Get file info for permission check
	fileInfo, err := manager.Info(ctx, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// Check permission
	if err := checkFilePermission(p, fileInfo, true); err != nil {
		return err
	}

	text, err := manager.GetText(ctx, fileID, fullContent)
	if err != nil {
		return fmt.Errorf("failed to get text: %v", err)
	}

	return text
}

// ============ Helper Functions ============

// parseDataURI parses content as either:
// 1. Data URI format: data:image/png;base64,xxxxx (decoded from base64)
// 2. Plain text: stored as-is with text/plain content type
//
// Returns content type, data bytes, and error
func parseDataURI(content string) (string, []byte, error) {
	// Handle data URI format: data:image/png;base64,xxxxx
	if strings.HasPrefix(content, "data:") {
		// Split by comma to get the data part
		parts := strings.SplitN(content, ",", 2)
		if len(parts) != 2 {
			return "", nil, fmt.Errorf("invalid data URI format")
		}

		// Parse the header: data:image/png;base64
		header := parts[0]
		base64Content := parts[1]

		// Extract content type from header
		contentType := "application/octet-stream"
		header = strings.TrimPrefix(header, "data:")
		headerParts := strings.Split(header, ";")
		if len(headerParts) > 0 && headerParts[0] != "" {
			contentType = headerParts[0]
		}

		// Decode base64
		data, err := base64.StdEncoding.DecodeString(base64Content)
		if err != nil {
			return "", nil, fmt.Errorf("failed to decode base64: %v", err)
		}

		return contentType, data, nil
	}

	// Plain text content - store as-is
	return "text/plain", []byte(content), nil
}

// generateFilename generates a filename based on content type
func generateFilename(contentType string) string {
	// Get extension from content type
	exts, err := mime.ExtensionsByType(contentType)
	if err == nil && len(exts) > 0 {
		return "file" + exts[0]
	}

	// Fallback for common types
	switch contentType {
	case "image/png":
		return "file.png"
	case "image/jpeg":
		return "file.jpg"
	case "image/gif":
		return "file.gif"
	case "image/webp":
		return "file.webp"
	case "text/plain":
		return "file.txt"
	case "application/pdf":
		return "file.pdf"
	case "application/json":
		return "file.json"
	default:
		return "file.bin"
	}
}

// createUploadOption creates UploadOption from process args
func createUploadOption(p *process.Process, filename string) UploadOption {
	option := UploadOption{
		OriginalFilename: filename,
	}

	// Parse option from fourth argument if provided
	if p.NumOfArgs() > 3 {
		optionRaw := p.ArgsMap(3)
		optionMap := maps.MapOf(optionRaw).Dot()

		// Groups
		if groups, ok := optionMap.Get("groups").([]interface{}); ok {
			for _, g := range groups {
				if gs, ok := g.(string); ok {
					option.Groups = append(option.Groups, gs)
				}
			}
		} else if groupsStr, ok := optionMap.Get("groups").(string); ok {
			option.Groups = strings.Split(groupsStr, ",")
			for i := range option.Groups {
				option.Groups[i] = strings.TrimSpace(option.Groups[i])
			}
		}

		// Gzip
		if gzip, ok := optionMap.Get("gzip").(bool); ok {
			option.Gzip = gzip
		}

		// Compress image
		if compress, ok := optionMap.Get("compress_image").(bool); ok {
			option.CompressImage = compress
		}
		if size := any.Of(optionMap.Get("compress_size")).CInt(); size > 0 {
			option.CompressSize = size
		}

		// Public/Share
		if public, ok := optionMap.Get("public").(bool); ok {
			option.Public = public
		}
		if share, ok := optionMap.Get("share").(string); ok {
			option.Share = share
		}
	}

	// Set permission fields from process.Authorized
	if p.Authorized != nil {
		option.YaoCreatedBy = p.Authorized.UserID
		option.YaoTeamID = p.Authorized.TeamID
		option.YaoTenantID = p.Authorized.TenantID
	}

	return option
}

// createFileHeader creates a FileHeader from parameters
func createFileHeader(filename, contentType string, size int64) *FileHeader {
	header := &multipart.FileHeader{
		Filename: filename,
		Size:     size,
		Header:   make(textproto.MIMEHeader),
	}
	header.Header.Set("Content-Type", contentType)

	// Set extension from filename
	if ext := filepath.Ext(filename); ext != "" {
		header.Header.Set("Content-Extension", ext)
	}

	return &FileHeader{FileHeader: header}
}

// checkFilePermission checks if user has permission to access the file
// readable: true for read permission, false for write permission
func checkFilePermission(p *process.Process, fileInfo *File, readable bool) error {
	auth := p.Authorized

	// No auth info - allow access (for non-authenticated operations)
	if auth == nil {
		return nil
	}

	// No constraints - allow access
	if !auth.Constraints.TeamOnly && !auth.Constraints.OwnerOnly {
		return nil
	}

	// Public files are readable by everyone
	if readable && fileInfo.Public {
		return nil
	}

	// Combined Team and Owner permission validation
	if auth.Constraints.TeamOnly && auth.Constraints.OwnerOnly {
		if fileInfo.YaoCreatedBy == auth.UserID && fileInfo.YaoTeamID == auth.TeamID {
			return nil
		}
	}

	// Owner only permission validation
	if auth.Constraints.OwnerOnly {
		if fileInfo.YaoCreatedBy != "" && fileInfo.YaoCreatedBy == auth.UserID {
			return nil
		}
	}

	// Team only permission validation
	if auth.Constraints.TeamOnly {
		switch fileInfo.Share {
		case "team":
			if fileInfo.YaoTeamID == auth.TeamID {
				return nil
			}
		case "private":
			if fileInfo.YaoCreatedBy == auth.UserID {
				return nil
			}
		}
	}

	return fmt.Errorf("forbidden: no permission to access file")
}

// buildPermissionWheres builds where clauses for permission filtering
func buildPermissionWheres(p *process.Process) []model.QueryWhere {
	auth := p.Authorized
	if auth == nil {
		return nil
	}

	// No constraints - no additional filtering needed
	if !auth.Constraints.TeamOnly && !auth.Constraints.OwnerOnly {
		return nil
	}

	var wheres []model.QueryWhere

	// Team only - User can access:
	// 1. Public files (public = true)
	// 2. Files in their team where:
	//    - They uploaded the file (__yao_created_by matches)
	//    - OR the file is shared with team (share = "team")
	if auth.Constraints.TeamOnly && auth.TeamID != "" {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{Column: "public", Value: true, Method: "orwhere"},
				{Wheres: []model.QueryWhere{
					{Column: "__yao_team_id", Value: auth.TeamID},
					{Wheres: []model.QueryWhere{
						{Column: "__yao_created_by", Value: auth.UserID},
						{Column: "share", Value: "team", Method: "orwhere"},
					}},
				}, Method: "orwhere"},
			},
		})
		return wheres
	}

	// Owner only - User can access:
	// 1. Public files (public = true)
	// 2. Files they uploaded where:
	//    - __yao_team_id is null (not team files)
	//    - __yao_created_by matches their user ID
	if auth.Constraints.OwnerOnly && auth.UserID != "" {
		wheres = append(wheres, model.QueryWhere{
			Wheres: []model.QueryWhere{
				{Column: "public", Value: true, Method: "orwhere"},
				{Wheres: []model.QueryWhere{
					{Column: "__yao_team_id", OP: "null"},
					{Column: "__yao_created_by", Value: auth.UserID},
				}, Method: "orwhere"},
			},
		})
		return wheres
	}

	return wheres
}

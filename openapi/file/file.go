package file

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	"github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
)

// Attach attaches the file management handlers to the router
func Attach(group *gin.RouterGroup, oauth types.OAuth) {
	// https://api.openai.com/v1/files
	// Protect all endpoints with OAuth
	group.Use(oauth.Guard)

	// Upload a file (supports chunked upload)
	group.POST("/:uploaderID", upload)

	// List files
	group.GET("/:uploaderID", list)

	// Retrieve file
	group.GET("/:uploaderID/:fileID", retrieve)

	// Delete file
	group.DELETE("/:uploaderID/:fileID", delete)

	// Retrieve file content
	group.GET("/:uploaderID/:fileID/content", content)

	// Check if file exists
	group.GET("/:uploaderID/:fileID/exists", exists)
}

// upload handles file upload
func upload(c *gin.Context) {
	// Get the uploader ID from the URL path
	uploaderID := c.Param("uploaderID")
	if uploaderID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get the attachment manager
	manager, exists := attachment.Managers[uploaderID]
	if !exists {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader not found: " + uploaderID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Parse multipart form
	err := c.Request.ParseMultipartForm(32 << 20) // 32 MB max memory
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Failed to parse multipart form: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get the file from the form
	file, fileHeader, err := c.Request.FormFile("file")
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}
	defer file.Close()

	// Create upload header from request
	header := attachment.GetHeader(c.Request.Header, fileHeader.Header, fileHeader.Size)

	// Create upload options with all parameters parsed from form data
	uploadOption := createUploadOption(c, fileHeader.Filename)

	// Upload the file
	uploadedFile, err := manager.Upload(c.Request.Context(), header, file, uploadOption)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to upload file: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return the uploaded file info
	response.RespondWithSuccess(c, response.StatusOK, uploadedFile)
}

// list handles file listing with pagination and filtering
func list(c *gin.Context) {
	uploaderID := c.Param("uploaderID")
	if uploaderID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get the attachment manager
	manager, ok := attachment.Managers[uploaderID]
	if !ok {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader not found: " + uploaderID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Parse query parameters
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	// Get auth info for permission filtering
	authInfo := authorized.GetInfo(c)

	// Parse filters
	filters := make(map[string]interface{})
	filters["uploader"] = uploaderID // Always filter by current uploader

	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if contentType := c.Query("content_type"); contentType != "" {
		filters["content_type"] = contentType
	}
	if name := c.Query("name"); name != "" {
		filters["name"] = name + "*" // Wildcard search
	}

	// Build where clauses for permission-based filtering
	var wheres []model.QueryWhere

	// Add basic filters as where clauses
	wheres = append(wheres, model.QueryWhere{
		Column: "uploader",
		Value:  uploaderID,
	})

	// Apply permission-based filtering
	wheres = append(wheres, AuthFilter(c, authInfo)...)

	// Parse order by
	orderBy := c.Query("order_by")
	if orderBy == "" {
		orderBy = "created_at desc"
	}

	// Parse select fields
	var selectFields []string
	if selectStr := c.Query("select"); selectStr != "" {
		selectFields = strings.Split(selectStr, ",")
		for i, field := range selectFields {
			selectFields[i] = strings.TrimSpace(field)
		}
	}

	// Create list option with where clauses
	listOption := attachment.ListOption{
		Page:     page,
		PageSize: pageSize,
		Filters:  filters,
		Wheres:   wheres,
		OrderBy:  orderBy,
		Select:   selectFields,
	}

	// Get file list
	result, err := manager.List(c.Request.Context(), listOption)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to list files: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Return the list result
	response.RespondWithSuccess(c, response.StatusOK, result)
}

// retrieve handles file metadata retrieval
func retrieve(c *gin.Context) {
	uploaderID := c.Param("uploaderID")
	fileID, _ := url.QueryUnescape(c.Param("fileID"))

	if uploaderID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if fileID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get the attachment manager
	manager, ok := attachment.Managers[uploaderID]
	if !ok {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader not found: " + uploaderID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Get file info (includes permission fields)
	fileInfo, err := manager.Info(c.Request.Context(), fileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Check read permission using file info
	authInfo := authorized.GetInfo(c)
	hasPermission, err := checkFilePermission(authInfo, fileInfo, true) // true = readable mode
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access file",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Return the file info
	response.RespondWithSuccess(c, response.StatusOK, fileInfo)
}

// delete handles file deletion
func delete(c *gin.Context) {
	uploaderID := c.Param("uploaderID")
	fileID, _ := url.QueryUnescape(c.Param("fileID"))

	if uploaderID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if fileID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get the attachment manager
	manager, ok := attachment.Managers[uploaderID]
	if !ok {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader not found: " + uploaderID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Get file info first (includes permission fields)
	fileInfo, err := manager.Info(c.Request.Context(), fileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Check delete permission using file info (false = write permission required)
	authInfo := authorized.GetInfo(c)
	hasPermission, err := checkFilePermission(authInfo, fileInfo, false) // false = write permission required
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to delete file",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Delete the file (permission already checked)
	err = manager.Delete(c.Request.Context(), fileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to delete file: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	successData := gin.H{
		"message": "File deleted successfully",
		"file_id": fileID,
	}
	response.RespondWithSuccess(c, response.StatusOK, successData)
}

// content handles file content retrieval
func content(c *gin.Context) {
	uploaderID := c.Param("uploaderID")
	fileID, _ := url.QueryUnescape(c.Param("fileID"))

	if uploaderID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if fileID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get the attachment manager
	manager, ok := attachment.Managers[uploaderID]
	if !ok {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader not found: " + uploaderID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Get file info (includes permission fields)
	fileInfo, err := manager.Info(c.Request.Context(), fileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Check read permission using file info
	authInfo := authorized.GetInfo(c)
	hasPermission, err := checkFilePermission(authInfo, fileInfo, true) // true = readable mode
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: err.Error(),
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	if !hasPermission {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrAccessDenied.Code,
			ErrorDescription: "Forbidden: No permission to access file content",
		}
		response.RespondWithError(c, response.StatusForbidden, errorResp)
		return
	}

	// Read the file content
	fileContent, err := manager.Read(c.Request.Context(), fileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrServerError.Code,
			ErrorDescription: "Failed to read file: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusInternalServerError, errorResp)
		return
	}

	// Set headers based on file info
	c.Header("Content-Type", fileInfo.ContentType)
	if fileInfo.Filename != "" {
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", fileInfo.Filename))
	}
	c.Header("Content-Length", fmt.Sprintf("%d", len(fileContent)))

	// Return file content directly
	c.Data(http.StatusOK, fileInfo.ContentType, fileContent)
}

// exists checks if a file exists
func exists(c *gin.Context) {
	uploaderID := c.Param("uploaderID")
	fileID, _ := url.QueryUnescape(c.Param("fileID"))

	if uploaderID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	if fileID == "" {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File ID is required",
		}
		response.RespondWithError(c, response.StatusBadRequest, errorResp)
		return
	}

	// Get the attachment manager
	manager, ok := attachment.Managers[uploaderID]
	if !ok {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "Uploader not found: " + uploaderID,
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Check if file exists
	exists := manager.Exists(c.Request.Context(), fileID)

	successData := gin.H{
		"exists":  exists,
		"file_id": fileID,
	}
	response.RespondWithSuccess(c, response.StatusOK, successData)
}

// createUploadOption creates an UploadOption from request context and form data
// Parses all upload parameters including auth info, permission fields, and upload options
func createUploadOption(c *gin.Context, defaultFilename string) attachment.UploadOption {
	option := attachment.UploadOption{}

	// Parse original filename from form data
	originalFilename := c.PostForm("original_filename")
	if originalFilename == "" {
		originalFilename = defaultFilename
	}
	option.OriginalFilename = originalFilename

	// Parse groups from form data
	if groupsStr := c.PostForm("groups"); groupsStr != "" {
		groups := strings.Split(groupsStr, ",")
		// Trim spaces from each group
		for i, group := range groups {
			groups[i] = strings.TrimSpace(group)
		}
		option.Groups = groups
	}

	// Parse gzip option
	if gzipStr := c.PostForm("gzip"); gzipStr == "true" || gzipStr == "1" {
		option.Gzip = true
	}

	// Parse compress image options
	if compressImageStr := c.PostForm("compress_image"); compressImageStr == "true" || compressImageStr == "1" {
		option.CompressImage = true
	}

	// Parse compress size
	if compressSizeStr := c.PostForm("compress_size"); compressSizeStr != "" {
		if size, err := strconv.Atoi(compressSizeStr); err == nil && size > 0 {
			option.CompressSize = size
		}
	}

	// Extract auth info from context (set by OAuth guard middleware)
	authInfo := authorized.GetInfo(c)
	if authInfo != nil {
		// Set Yao permission fields from authenticated user info
		// Note: YaoUpdatedBy is not set on upload (creation), only on update
		if authInfo.UserID != "" {
			option.YaoCreatedBy = authInfo.UserID
		}
		if authInfo.TeamID != "" {
			option.YaoTeamID = authInfo.TeamID
		}
		if authInfo.TenantID != "" {
			option.YaoTenantID = authInfo.TenantID
		}
	}

	// Parse public field from form data (user can override)
	if publicStr := c.PostForm("public"); publicStr != "" {
		if publicStr == "true" || publicStr == "1" {
			option.Public = true
		} else {
			option.Public = false
		}
	}

	// Parse share field from form data (user can override)
	// Valid values: "private", "team"
	if shareStr := c.PostForm("share"); shareStr != "" {
		shareStr = strings.TrimSpace(strings.ToLower(shareStr))
		if shareStr == "private" || shareStr == "team" {
			option.Share = shareStr
		}
	}

	return option
}

// checkFilePermission checks if the user has permission to access the file
func checkFilePermission(authInfo *types.AuthorizedInfo, fileInfo *attachment.File, readable ...bool) (bool, error) {
	// No auth info, allow access
	if authInfo == nil {
		return true, nil
	}

	// No constraints, allow access
	if !authInfo.Constraints.TeamOnly && !authInfo.Constraints.OwnerOnly {
		return true, nil
	}

	// If readable mode and file is public, allow access
	if len(readable) > 0 && readable[0] {
		if fileInfo.Public {
			return true, nil
		}

		// Combined Team and Owner permission validation
		if authInfo.Constraints.TeamOnly && authInfo.Constraints.OwnerOnly {
			if fileInfo.YaoCreatedBy == authInfo.UserID && fileInfo.YaoTeamID == authInfo.TeamID {
				return true, nil
			}
		}

		// Owner only permission validation
		if authInfo.Constraints.OwnerOnly {
			if fileInfo.YaoCreatedBy == authInfo.UserID {
				return true, nil
			}
		}

		// Team only permission validation
		if authInfo.Constraints.TeamOnly {

			switch fileInfo.Share {
			case "team":
				if fileInfo.YaoTeamID == authInfo.TeamID {
					return true, nil
				}
			case "private":
				if fileInfo.YaoCreatedBy == authInfo.UserID {
					return true, nil
				}
			}
		}

	}

	// Combined Team and Owner permission validation
	if authInfo.Constraints.TeamOnly && authInfo.Constraints.OwnerOnly {
		if fileInfo.YaoCreatedBy == authInfo.UserID && fileInfo.YaoTeamID == authInfo.TeamID {
			return true, nil
		}
	}

	// Owner only permission validation
	if authInfo.Constraints.OwnerOnly && fileInfo.YaoCreatedBy == authInfo.UserID {
		return true, nil
	}

	// Team only permission validation
	if authInfo.Constraints.TeamOnly && fileInfo.YaoTeamID == authInfo.TeamID && fileInfo.Share == "team" {
		return true, nil
	}

	return false, nil
}

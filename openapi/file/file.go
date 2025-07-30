package file

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/attachment"
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

	// Get original filename from form data
	originalFilename := c.PostForm("original_filename")
	if originalFilename == "" {
		originalFilename = fileHeader.Filename
	}

	// Get path from form data for user_path
	userPath := c.PostForm("path")
	if userPath == "" {
		userPath = originalFilename
	}

	// Parse groups from form data
	var groups []string
	groupsStr := c.PostForm("groups")
	if groupsStr != "" {
		groups = strings.Split(groupsStr, ",")
		// Trim spaces
		for i, group := range groups {
			groups[i] = strings.TrimSpace(group)
		}
	}

	// Create upload header from request
	header := attachment.GetHeader(c.Request.Header, fileHeader.Header, fileHeader.Size)

	// Parse gzip option
	gzip := false
	if gzipStr := c.PostForm("gzip"); gzipStr == "true" {
		gzip = true
	}

	// Parse compress image options
	compressImage := false
	if compressImageStr := c.PostForm("compress_image"); compressImageStr == "true" {
		compressImage = true
	}

	compressSize := 0
	if compressSizeStr := c.PostForm("compress_size"); compressSizeStr != "" {
		if size, err := strconv.Atoi(compressSizeStr); err == nil && size > 0 {
			compressSize = size
		}
	}

	// Create upload options
	uploadOption := attachment.UploadOption{
		OriginalFilename: originalFilename, // Use original filename from form data
		Groups:           groups,           // Groups for directory structure
		ClientID:         c.PostForm("client_id"),
		OpenID:           c.PostForm("openid"),
		Gzip:             gzip,          // Gzip compression
		CompressImage:    compressImage, // Image compression
		CompressSize:     compressSize,  // Compression size
	}

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

	// Create list option
	listOption := attachment.ListOption{
		Page:     page,
		PageSize: pageSize,
		Filters:  filters,
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

	// Get file info using the new Info method
	fileInfo, err := manager.Info(c.Request.Context(), fileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
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

	// Check if file exists first
	if !manager.Exists(c.Request.Context(), fileID) {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found",
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Delete the file
	err := manager.Delete(c.Request.Context(), fileID)
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

	// Get file info first to obtain metadata
	fileInfo, err := manager.Info(c.Request.Context(), fileID)
	if err != nil {
		errorResp := &response.ErrorResponse{
			Code:             response.ErrInvalidRequest.Code,
			ErrorDescription: "File not found: " + err.Error(),
		}
		response.RespondWithError(c, response.StatusNotFound, errorResp)
		return
	}

	// Read the file content
	content, err := manager.Read(c.Request.Context(), fileID)
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
	c.Header("Content-Length", fmt.Sprintf("%d", len(content)))

	// Return file content directly
	c.Data(http.StatusOK, fileInfo.ContentType, content)
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

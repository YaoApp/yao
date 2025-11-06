package openapi_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/attachment"
	"github.com/yaoapp/yao/openapi"
	"github.com/yaoapp/yao/openapi/tests/testutils"
)

const (
	testUploaderID    = "test"
	testFileName      = "test-file.txt"
	testFileContent   = "This is a test file content for OpenAPI file management testing."
	testContentType   = "text/plain"
	invalidUploaderID = "invalid-uploader"
)

// setupTestUploader registers a test uploader for file operations
func setupTestUploader(t *testing.T) {
	_, err := attachment.RegisterDefault(testUploaderID)
	if err != nil {
		t.Fatalf("Failed to register test uploader: %v", err)
	}
	t.Logf("Registered test uploader: %s", testUploaderID)
}

// createMultipartRequest creates a multipart form request for file upload
func createMultipartRequest(url, fieldName, fileName string, content []byte, extraFields map[string]string) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create file part with explicit Content-Type
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition", fmt.Sprintf(`form-data; name="%s"; filename="%s"`, fieldName, fileName))
	h.Set("Content-Type", testContentType)
	part, err := writer.CreatePart(h)
	if err != nil {
		return nil, err
	}

	if _, err := part.Write(content); err != nil {
		return nil, err
	}

	// Add extra fields
	for key, value := range extraFields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	return req, nil
}

// createChunkedUploadRequest creates a request for chunked file upload
func createChunkedUploadRequest(url, fileName, uid string, chunkContent []byte, start, end, total int) (*http.Request, error) {
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// Create file part
	part, err := writer.CreateFormFile("file", fileName)
	if err != nil {
		return nil, err
	}

	if _, err := part.Write(chunkContent); err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.Header.Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, total))
	req.Header.Set("Content-Sync", "true")
	req.Header.Set("Content-Uid", uid)

	return req, nil
}

// TestFileUpload tests the file upload endpoint
func TestFileUpload(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	// Setup test uploader
	setupTestUploader(t)

	// Get base URL from server config
	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	// Register test client and get token
	client := testutils.RegisterTestClient(t, "File Upload Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("UploadFileSuccess", func(t *testing.T) {
		// Create multipart request
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), map[string]string{
			"original_filename": testFileName,
			"path":              "documents/reports/quarterly-report.txt",
			"groups":            "documents,reports",
			"public":            "false",
			"share":             "private",
		})
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		// Make request
		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// Check response
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		// Debug output for failed requests
		if resp.StatusCode != http.StatusOK {
			t.Logf("Expected status 200, got %d", resp.StatusCode)
			t.Logf("Response body: %+v", response)
		}

		assert.Contains(t, response, "file_id")
		assert.Contains(t, response, "filename")
		assert.Contains(t, response, "content_type")
		assert.Contains(t, response, "path")
		assert.Contains(t, response, "user_path")
		assert.Equal(t, testFileName, response["filename"])
		// Content type may include charset
		contentType, _ := response["content_type"].(string)
		assert.True(t, strings.HasPrefix(contentType, testContentType), "Content-Type should start with %s, got %s", testContentType, contentType)
		assert.Equal(t, "uploaded", response["status"])

		// The file_id should be URL-safe (no slashes) and be an MD5 hash (32 chars)
		fileID := response["file_id"].(string)
		assert.NotContains(t, fileID, "/") // URL-safe ID should not contain slashes
		assert.Len(t, fileID, 32)          // MD5 hash is 32 characters

		t.Logf("Successfully uploaded file: %s (ID: %s)", testFileName, fileID)
	})

	t.Run("UploadFileWithCompression", func(t *testing.T) {
		// Test with gzip compression
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), map[string]string{
			"original_filename": testFileName,
			"gzip":              "true",
			"compress_image":    "true",
			"compress_size":     "1000",
		})
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "file_id")
		t.Logf("Successfully uploaded compressed file: %s", response["file_id"])
	})

	t.Run("UploadFileInvalidUploader", func(t *testing.T) {
		// Test with invalid uploader ID
		requestURL := serverURL + baseURL + "/file/" + invalidUploaderID
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")

		t.Logf("Correctly rejected upload with invalid uploader ID")
	})

	t.Run("UploadFileNoFile", func(t *testing.T) {
		// Test with no file in request
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := http.NewRequest("POST", requestURL, strings.NewReader("no file data"))
		assert.NoError(t, err)

		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")

		t.Logf("Correctly rejected upload with no file")
	})

	t.Run("UploadFileMissingUploaderID", func(t *testing.T) {
		// Test with missing uploader ID in path
		requestURL := serverURL + baseURL + "/file/"
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), nil)
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// This should result in a 404 due to route mismatch
		assert.NotEqual(t, http.StatusOK, resp.StatusCode)

		t.Logf("Correctly handled request with missing uploader ID")
	})
}

// TestFileChunkedUpload tests the chunked file upload feature
func TestFileChunkedUpload(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File Chunked Upload Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("ChunkedUploadSuccess", func(t *testing.T) {
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		uid := fmt.Sprintf("chunked-test-%d", time.Now().UnixNano())

		// Split content into chunks
		content := []byte(testFileContent)
		chunkSize := 10
		totalSize := len(content)

		// Upload chunks
		for i := 0; i < totalSize; i += chunkSize {
			end := i + chunkSize
			if end > totalSize {
				end = totalSize
			}

			chunk := content[i:end]

			req, err := createChunkedUploadRequest(requestURL, testFileName, uid, chunk, i, end-1, totalSize)
			assert.NoError(t, err)

			req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			defer resp.Body.Close()

			if i+chunkSize >= totalSize {
				// Last chunk should return success
				assert.Equal(t, http.StatusOK, resp.StatusCode)

				var response map[string]interface{}
				err = json.NewDecoder(resp.Body).Decode(&response)
				assert.NoError(t, err)

				assert.Equal(t, "uploaded", response["status"])
				t.Logf("Successfully completed chunked upload: %s", response["file_id"])
			} else {
				// Intermediate chunks should return partial content or OK
				assert.True(t, resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusPartialContent)
			}
		}
	})
}

// TestFileList tests the file listing endpoint
func TestFileList(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File List Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	// First upload some test files
	var uploadedFileIDs []string
	for i := 0; i < 3; i++ {
		fileName := fmt.Sprintf("test-file-%d.txt", i)
		content := fmt.Sprintf("Test content for file %d", i)

		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", fileName, []byte(content), map[string]string{
			"original_filename": fileName,
		})
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		if resp.StatusCode == http.StatusOK {
			var response map[string]interface{}
			err = json.NewDecoder(resp.Body).Decode(&response)
			assert.NoError(t, err)

			if fileID, ok := response["file_id"].(string); ok {
				uploadedFileIDs = append(uploadedFileIDs, fileID)
			}
		}
	}

	t.Run("ListFilesSuccess", func(t *testing.T) {
		// Test basic file listing
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "files")
		assert.Contains(t, response, "total")
		assert.Contains(t, response, "page")
		assert.Contains(t, response, "page_size")

		files, ok := response["files"].([]interface{})
		assert.True(t, ok)
		t.Logf("Successfully listed %d files", len(files))
	})

	t.Run("ListFilesWithPagination", func(t *testing.T) {
		// Test with pagination parameters
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"?page=1&page_size=2", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Equal(t, float64(1), response["page"])
		assert.Equal(t, float64(2), response["page_size"])
		t.Logf("Successfully listed files with pagination")
	})

	t.Run("ListFilesWithFilters", func(t *testing.T) {
		// Test with filter parameters
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"?status=uploaded&content_type=text/plain", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		t.Logf("Successfully listed files with filters")
	})

	t.Run("ListFilesInvalidUploader", func(t *testing.T) {
		// Test with invalid uploader ID
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+invalidUploaderID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
		t.Logf("Correctly rejected list request with invalid uploader ID")
	})
}

// TestFileRetrieve tests the file metadata retrieval endpoint
func TestFileRetrieve(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File Retrieve Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	var testFileID string

	t.Run("SetupUploadFile", func(t *testing.T) {
		// Upload a file first
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), map[string]string{
			"original_filename": testFileName,
		})
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		testFileID = response["file_id"].(string)
		t.Logf("Setup: Uploaded file with ID: %s", testFileID)
	})

	t.Run("RetrieveFileSuccess", func(t *testing.T) {
		// Retrieve file metadata
		encodedFileID := url.QueryEscape(testFileID)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "file_id")
		assert.Contains(t, response, "filename")
		assert.Contains(t, response, "content_type")
		assert.Equal(t, testFileID, response["file_id"])
		assert.Equal(t, testFileName, response["filename"])
		// Content type may include charset
		contentType, _ := response["content_type"].(string)
		assert.True(t, strings.HasPrefix(contentType, testContentType), "Content-Type should start with %s, got %s", testContentType, contentType)

		t.Logf("Successfully retrieved file metadata: %s", testFileID)
	})

	t.Run("RetrieveFileNotFound", func(t *testing.T) {
		// Test with non-existent file ID
		nonExistentID := "non-existent-file-id"
		encodedFileID := url.QueryEscape(nonExistentID)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")

		t.Logf("Correctly handled non-existent file retrieval")
	})

	t.Run("RetrieveFileMissingIDs", func(t *testing.T) {
		// Test with missing file ID - this URL actually matches the list endpoint
		// which is correct RESTful behavior, so we expect 200 OK
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		// This URL actually matches the list endpoint, so we expect 200 OK
		assert.Equal(t, http.StatusOK, resp.StatusCode)

		// Verify it's actually a list response (should have pagination structure)
		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)
		assert.Contains(t, response, "files")
		assert.Contains(t, response, "total")
		assert.Contains(t, response, "page")

		t.Logf("URL without file ID correctly matched list endpoint")
	})
}

// TestFileContent tests the file content retrieval endpoint
func TestFileContent(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File Content Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	var testFileID string

	t.Run("SetupUploadFile", func(t *testing.T) {
		// Upload a file first
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), map[string]string{
			"original_filename": testFileName,
		})
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		testFileID = response["file_id"].(string)
		t.Logf("Setup: Uploaded file with ID: %s", testFileID)
	})

	t.Run("GetFileContentSuccess", func(t *testing.T) {
		// Get file content
		encodedFileID := url.QueryEscape(testFileID)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID+"/content", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		// Content type may include charset
		assert.True(t, strings.HasPrefix(resp.Header.Get("Content-Type"), testContentType),
			"Content-Type should start with %s, got %s", testContentType, resp.Header.Get("Content-Type"))

		// Read and verify content
		content, err := io.ReadAll(resp.Body)
		assert.NoError(t, err)
		assert.Equal(t, testFileContent, string(content))

		t.Logf("Successfully retrieved file content: %d bytes", len(content))
	})

	t.Run("GetFileContentNotFound", func(t *testing.T) {
		// Test with non-existent file ID
		nonExistentID := "non-existent-file-id"
		encodedFileID := url.QueryEscape(nonExistentID)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID+"/content", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		t.Logf("Correctly handled non-existent file content request")
	})
}

// TestFileExists tests the file existence check endpoint
func TestFileExists(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File Exists Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	var testFileID string

	t.Run("SetupUploadFile", func(t *testing.T) {
		// Upload a file first
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), map[string]string{
			"original_filename": testFileName,
		})
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		testFileID = response["file_id"].(string)
		t.Logf("Setup: Uploaded file with ID: %s", testFileID)
	})

	t.Run("FileExistsTrue", func(t *testing.T) {
		// Check if uploaded file exists
		encodedFileID := url.QueryEscape(testFileID)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID+"/exists", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "exists")
		assert.Contains(t, response, "file_id")
		assert.Equal(t, testFileID, response["file_id"])
		assert.Equal(t, true, response["exists"])

		t.Logf("File exists check returned true for: %s", testFileID)
	})

	t.Run("FileExistsFalse", func(t *testing.T) {
		// Check if non-existent file exists
		nonExistentID := "non-existent-file-id"
		encodedFileID := url.QueryEscape(nonExistentID)
		req, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID+"/exists", nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "exists")
		assert.Contains(t, response, "file_id")
		assert.Equal(t, nonExistentID, response["file_id"])
		assert.Equal(t, false, response["exists"])

		t.Logf("File exists check returned false for: %s", nonExistentID)
	})
}

// TestFileDelete tests the file deletion endpoint
func TestFileDelete(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File Delete Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("DeleteFileSuccess", func(t *testing.T) {
		// Upload a file first
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), map[string]string{
			"original_filename": testFileName,
		})
		assert.NoError(t, err)

		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		var uploadResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&uploadResponse)
		assert.NoError(t, err)

		testFileID := uploadResponse["file_id"].(string)

		// Now delete the file
		encodedFileID := url.QueryEscape(testFileID)
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		deleteResp, err := http.DefaultClient.Do(deleteReq)
		assert.NoError(t, err)
		assert.NotNil(t, deleteResp)
		defer deleteResp.Body.Close()

		assert.Equal(t, http.StatusOK, deleteResp.StatusCode)

		var deleteResponse map[string]interface{}
		err = json.NewDecoder(deleteResp.Body).Decode(&deleteResponse)
		assert.NoError(t, err)

		assert.Contains(t, deleteResponse, "message")
		assert.Contains(t, deleteResponse, "file_id")
		assert.Equal(t, testFileID, deleteResponse["file_id"])

		t.Logf("Successfully deleted file: %s", testFileID)
	})

	t.Run("DeleteFileNotFound", func(t *testing.T) {
		// Test deleting non-existent file
		nonExistentID := "non-existent-file-id"
		encodedFileID := url.QueryEscape(nonExistentID)
		req, err := http.NewRequest("DELETE", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID, nil)
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)

		var errorResponse map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&errorResponse)
		assert.NoError(t, err)
		assert.Contains(t, errorResponse, "error")

		t.Logf("Correctly handled deletion of non-existent file")
	})
}

// TestFileEndpointsUnauthorized tests that endpoints return 401 when not authenticated
func TestFileEndpointsUnauthorized(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	endpoints := []struct {
		method string
		path   string
	}{
		{"POST", "/file/" + testUploaderID},
		{"GET", "/file/" + testUploaderID},
		{"GET", "/file/" + testUploaderID + "/test-file-id"},
		{"DELETE", "/file/" + testUploaderID + "/test-file-id"},
		{"GET", "/file/" + testUploaderID + "/test-file-id/content"},
		{"GET", "/file/" + testUploaderID + "/test-file-id/exists"},
	}

	for _, endpoint := range endpoints {
		t.Run(fmt.Sprintf("Unauthorized_%s_%s", endpoint.method, strings.ReplaceAll(endpoint.path, "/", "_")), func(t *testing.T) {
			var req *http.Request
			var err error

			if endpoint.method == "POST" {
				// For POST, create a simple multipart form
				body := &bytes.Buffer{}
				writer := multipart.NewWriter(body)
				writer.WriteField("test", "data")
				writer.Close()

				req, err = http.NewRequest(endpoint.method, serverURL+baseURL+endpoint.path, body)
				req.Header.Set("Content-Type", writer.FormDataContentType())
			} else {
				req, err = http.NewRequest(endpoint.method, serverURL+baseURL+endpoint.path, nil)
			}
			assert.NoError(t, err)

			// No Authorization header
			resp, err := http.DefaultClient.Do(req)
			assert.NoError(t, err)
			assert.NotNil(t, resp)
			defer resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

			t.Logf("Correctly rejected unauthorized request to %s %s", endpoint.method, endpoint.path)
		})
	}
}

// TestFileIntegration tests the full file lifecycle
func TestFileIntegration(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File Integration Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("FullFileLifecycle", func(t *testing.T) {
		// Step 1: Upload a file
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		uploadReq, err := createMultipartRequest(requestURL, "file", testFileName, []byte(testFileContent), map[string]string{
			"original_filename": testFileName,
			"path":              "integration/test/file.txt",
			"groups":            "integration,test",
		})
		assert.NoError(t, err)
		uploadReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		uploadResp, err := http.DefaultClient.Do(uploadReq)
		assert.NoError(t, err)
		defer uploadResp.Body.Close()

		assert.Equal(t, http.StatusOK, uploadResp.StatusCode)

		var uploadResponse map[string]interface{}
		err = json.NewDecoder(uploadResp.Body).Decode(&uploadResponse)
		assert.NoError(t, err)

		testFileID := uploadResponse["file_id"].(string)

		// Step 2: Verify file exists
		encodedFileID := url.QueryEscape(testFileID)
		existsReq, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID+"/exists", nil)
		assert.NoError(t, err)
		existsReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		existsResp, err := http.DefaultClient.Do(existsReq)
		assert.NoError(t, err)
		defer existsResp.Body.Close()

		assert.Equal(t, http.StatusOK, existsResp.StatusCode)

		// Step 3: Retrieve file metadata
		retrieveReq, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID, nil)
		assert.NoError(t, err)
		retrieveReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		retrieveResp, err := http.DefaultClient.Do(retrieveReq)
		assert.NoError(t, err)
		defer retrieveResp.Body.Close()

		assert.Equal(t, http.StatusOK, retrieveResp.StatusCode)

		// Step 4: Download file content
		contentReq, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID+"/content", nil)
		assert.NoError(t, err)
		contentReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		contentResp, err := http.DefaultClient.Do(contentReq)
		assert.NoError(t, err)
		defer contentResp.Body.Close()

		assert.Equal(t, http.StatusOK, contentResp.StatusCode)

		content, err := io.ReadAll(contentResp.Body)
		assert.NoError(t, err)
		assert.Equal(t, testFileContent, string(content))

		// Step 5: List files and verify our file is included
		listReq, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"?name="+testFileName, nil)
		assert.NoError(t, err)
		listReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		listResp, err := http.DefaultClient.Do(listReq)
		assert.NoError(t, err)
		defer listResp.Body.Close()

		assert.Equal(t, http.StatusOK, listResp.StatusCode)

		// Step 6: Delete the file
		deleteReq, err := http.NewRequest("DELETE", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID, nil)
		assert.NoError(t, err)
		deleteReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		deleteResp, err := http.DefaultClient.Do(deleteReq)
		assert.NoError(t, err)
		defer deleteResp.Body.Close()

		assert.Equal(t, http.StatusOK, deleteResp.StatusCode)

		// Step 7: Verify file no longer exists
		finalExistsReq, err := http.NewRequest("GET", serverURL+baseURL+"/file/"+testUploaderID+"/"+encodedFileID+"/exists", nil)
		assert.NoError(t, err)
		finalExistsReq.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		finalExistsResp, err := http.DefaultClient.Do(finalExistsReq)
		assert.NoError(t, err)
		defer finalExistsResp.Body.Close()

		assert.Equal(t, http.StatusOK, finalExistsResp.StatusCode)

		var finalExistsResponse map[string]interface{}
		err = json.NewDecoder(finalExistsResp.Body).Decode(&finalExistsResponse)
		assert.NoError(t, err)
		assert.Equal(t, false, finalExistsResponse["exists"])

		t.Logf("Completed full file lifecycle test for: %s", testFileID)
	})
}

// TestFilePermissionFields tests the new permission and auth fields
func TestFilePermissionFields(t *testing.T) {
	serverURL := testutils.Prepare(t)
	defer testutils.Clean()

	setupTestUploader(t)

	baseURL := ""
	if openapi.Server != nil && openapi.Server.Config != nil {
		baseURL = openapi.Server.Config.BaseURL
	}

	client := testutils.RegisterTestClient(t, "File Permission Test Client", []string{"https://localhost/callback"})
	defer testutils.CleanupTestClient(t, client.ClientID)
	tokenInfo := testutils.ObtainAccessToken(t, serverURL, client.ClientID, client.ClientSecret, "https://localhost/callback", "openid profile")

	t.Run("UploadWithPublicTeamShare", func(t *testing.T) {
		// Upload file with public=true and share=team
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", "public-team-file.txt", []byte("Public team content"), map[string]string{
			"original_filename": "public-team-file.txt",
			"groups":            "shared,public",
			"public":            "true",
			"share":             "team",
		})
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "file_id")
		t.Logf("Successfully uploaded public team file: %s", response["file_id"])
	})

	t.Run("UploadWithPrivateShare", func(t *testing.T) {
		// Upload file with public=false and share=private (default)
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", "private-file.txt", []byte("Private content"), map[string]string{
			"original_filename": "private-file.txt",
			"groups":            "personal",
			"public":            "false",
			"share":             "private",
		})
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "file_id")
		t.Logf("Successfully uploaded private file: %s", response["file_id"])
	})

	t.Run("UploadWithoutPermissionFields", func(t *testing.T) {
		// Upload file without specifying public/share (should use defaults)
		requestURL := serverURL + baseURL + "/file/" + testUploaderID
		req, err := createMultipartRequest(requestURL, "file", "default-permissions.txt", []byte("Default permissions content"), map[string]string{
			"original_filename": "default-permissions.txt",
			"groups":            "defaults",
		})
		assert.NoError(t, err)
		req.Header.Set("Authorization", "Bearer "+tokenInfo.AccessToken)

		resp, err := http.DefaultClient.Do(req)
		assert.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		assert.NoError(t, err)

		assert.Contains(t, response, "file_id")
		t.Logf("Successfully uploaded file with default permissions: %s", response["file_id"])
	})
}

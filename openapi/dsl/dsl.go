package dsl

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/dsl"
	"github.com/yaoapp/yao/dsl/types"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
)

// Yao DSL Manager API

// Attach attaches the DSL management handlers to the router
func Attach(group *gin.RouterGroup, oauth oauthTypes.OAuth) {

	// Protect all endpoints with OAuth
	group.Handlers = append(group.Handlers, oauth.Guard)

	// DSL Information endpoints
	group.GET("/inspect/:type/:id", inspect)
	group.GET("/source/:type/:id", source)
	group.GET("/path/:type/:id", path)
	group.GET("/list/:type", list)
	group.GET("/exists/:type/:id", exists)

	// DSL CRUD operations
	group.POST("/create/:type", create)
	group.PUT("/update/:type", update)
	group.DELETE("/delete/:type/:id", delete)

	// DSL Load management
	group.POST("/load/:type", load)
	group.POST("/unload/:type", unload)
	group.POST("/reload/:type", reload)

	// DSL Execute and Validate
	group.POST("/execute/:type/:id/:method", execute)
	group.POST("/validate/:type", validate)
}

// Inspect DSL information
func inspect(c *gin.Context) {
	dslType := types.Type(c.Param("type"))
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	info, err := dslManager.Inspect(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, info)
}

// Get DSL source code
func source(c *gin.Context) {
	dslType := types.Type(c.Param("type"))
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	sourceCode, err := dslManager.Source(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"source": sourceCode})
}

// Get DSL file path
func path(c *gin.Context) {
	dslType := types.Type(c.Param("type"))
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	filePath, err := dslManager.Path(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"path": filePath})
}

// List DSLs with optional filters
func list(c *gin.Context) {
	dslType := types.Type(c.Param("type"))

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	// Parse query parameters
	opts := &types.ListOptions{
		Sort:    c.Query("sort"),
		Order:   c.Query("order"),
		Store:   types.StoreType(c.Query("store")),
		Pattern: c.Query("pattern"),
	}

	// Parse source flag
	if sourceStr := c.Query("source"); sourceStr != "" {
		if sourceBool, err := strconv.ParseBool(sourceStr); err == nil {
			opts.Source = sourceBool
		}
	}

	// Parse tags from query parameter (comma-separated)
	if tagsStr := c.Query("tags"); tagsStr != "" {
		c.ShouldBindQuery(&struct {
			Tags []string `form:"tags"`
		}{Tags: opts.Tags})
	}

	infos, err := dslManager.List(c.Request.Context(), opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, infos)
}

// Check if DSL exists
func exists(c *gin.Context) {
	dslType := types.Type(c.Param("type"))
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	exist, err := dslManager.Exists(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"exists": exist})
}

// Create a new DSL
func create(c *gin.Context) {
	dslType := types.Type(c.Param("type"))

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	var options types.CreateOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if options.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	if options.Source == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL source is required"})
		return
	}

	err = dslManager.Create(c.Request.Context(), &options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "DSL created successfully"})
}

// Update an existing DSL
func update(c *gin.Context) {
	dslType := types.Type(c.Param("type"))

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	var options types.UpdateOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if options.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	if options.Source == "" && options.Info == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL source or info is required"})
		return
	}

	err = dslManager.Update(c.Request.Context(), &options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DSL updated successfully"})
}

// Delete a DSL
func delete(c *gin.Context) {
	dslType := types.Type(c.Param("type"))
	id := c.Param("id")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	// Parse optional request body for delete options
	var options types.DeleteOptions
	options.ID = id

	// Try to bind JSON body if provided
	c.ShouldBindJSON(&options)

	err = dslManager.Delete(c.Request.Context(), &options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DSL deleted successfully"})
}

// Load a DSL
func load(c *gin.Context) {
	dslType := types.Type(c.Param("type"))

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	var options types.LoadOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if options.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	err = dslManager.Load(c.Request.Context(), &options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DSL loaded successfully"})
}

// Unload a DSL
func unload(c *gin.Context) {
	dslType := types.Type(c.Param("type"))

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	var options types.UnloadOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if options.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	err = dslManager.Unload(c.Request.Context(), &options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DSL unloaded successfully"})
}

// Reload a DSL
func reload(c *gin.Context) {
	dslType := types.Type(c.Param("type"))

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	var options types.ReloadOptions
	if err := c.ShouldBindJSON(&options); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request body: " + err.Error()})
		return
	}

	if options.ID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	err = dslManager.Reload(c.Request.Context(), &options)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "DSL reloaded successfully"})
}

// Execute a DSL method
func execute(c *gin.Context) {
	dslType := types.Type(c.Param("type"))
	id := c.Param("id")
	method := c.Param("method")

	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL ID is required"})
		return
	}

	if method == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Method name is required"})
		return
	}

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	// Parse arguments from request body
	var requestBody struct {
		Args []interface{} `json:"args"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		// If no body provided, execute without arguments
		requestBody.Args = []interface{}{}
	}

	result, err := dslManager.Execute(c.Request.Context(), id, method, requestBody.Args...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"result": result})
}

// Validate DSL source code
func validate(c *gin.Context) {
	dslType := types.Type(c.Param("type"))

	dslManager, err := dsl.New(dslType)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid DSL type: " + string(dslType)})
		return
	}

	var requestBody struct {
		Source string `json:"source" binding:"required"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "DSL source is required"})
		return
	}

	valid, messages := dslManager.Validate(c.Request.Context(), requestBody.Source)

	c.JSON(http.StatusOK, gin.H{
		"valid":    valid,
		"messages": messages,
	})
}

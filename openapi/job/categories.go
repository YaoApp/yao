package job

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/model"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/job"
)

// ListCategories lists job categories
func ListCategories(c *gin.Context) {
	// Build query parameters
	param := model.QueryParam{}

	// Add enabled filter (default to true)
	enabled := c.DefaultQuery("enabled", "true")
	if enabled == "true" {
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "enabled",
			Value:  true,
		})
	}

	// Add system filter if provided
	if system := c.Query("system"); system != "" {
		systemBool := system == "true"
		param.Wheres = append(param.Wheres, model.QueryWhere{
			Column: "system",
			Value:  systemBool,
		})
	}

	// Order by sort and name
	param.Orders = []model.QueryOrder{
		{Column: "sort", Option: "asc"},
		{Column: "name", Option: "asc"},
	}

	// Get categories
	categories, err := job.GetCategories(param)
	if err != nil {
		log.Error("Failed to list categories: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := gin.H{
		"data":  categories,
		"total": len(categories),
	}

	c.JSON(http.StatusOK, response)
}

// GetCategory gets a specific category by ID
func GetCategory(c *gin.Context) {
	categoryID := c.Param("categoryID")
	if categoryID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category_id is required"})
		return
	}

	// Build query parameters to find by category_id
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "category_id", Value: categoryID},
		},
		Limit: 1,
	}

	// Get categories with filter
	categories, err := job.GetCategories(param)
	if err != nil {
		log.Error("Failed to get category %s: %v", categoryID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(categories) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Category not found"})
		return
	}

	c.JSON(http.StatusOK, categories[0])
}

// ========================
// Process Handlers
// ========================

// ProcessListCategories process handler for listing categories
func ProcessListCategories(process *process.Process) interface{} {
	// TODO: Implement process handler for listing categories
	args := process.Args
	log.Info("ProcessListCategories called with args: %v", args)

	// Build query parameters
	param := model.QueryParam{}
	if len(args) > 0 {
		if queryParam, ok := args[0].(model.QueryParam); ok {
			param = queryParam
		}
	}

	// Call job.GetCategories function
	categories, err := job.GetCategories(param)
	if err != nil {
		log.Error("Failed to list categories: %v", err)
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{
		"categories": categories,
		"count":      len(categories),
	}
}

// ProcessGetCategory process handler for getting a category
func ProcessGetCategory(process *process.Process) interface{} {
	// TODO: Implement process handler for getting a category
	args := process.Args
	if len(args) == 0 {
		return map[string]interface{}{"error": "category_id is required"}
	}

	categoryID, ok := args[0].(string)
	if !ok {
		return map[string]interface{}{"error": "category_id must be a string"}
	}

	log.Info("ProcessGetCategory called for category: %s", categoryID)

	// Build query parameters to find by category_id
	param := model.QueryParam{
		Wheres: []model.QueryWhere{
			{Column: "category_id", Value: categoryID},
		},
		Limit: 1,
	}

	// Call job.GetCategories function with filter
	categories, err := job.GetCategories(param)
	if err != nil {
		log.Error("Failed to get category %s: %v", categoryID, err)
		return map[string]interface{}{"error": err.Error()}
	}

	if len(categories) == 0 {
		return map[string]interface{}{"error": "category not found"}
	}

	return categories[0]
}

// ProcessCountCategories process handler for counting categories
func ProcessCountCategories(process *process.Process) interface{} {
	// TODO: Implement process handler for counting categories
	args := process.Args
	log.Info("ProcessCountCategories called with args: %v", args)

	// Build query parameters
	param := model.QueryParam{}
	if len(args) > 0 {
		if queryParam, ok := args[0].(model.QueryParam); ok {
			param = queryParam
		}
	}

	// Call job.CountCategories function
	count, err := job.CountCategories(param)
	if err != nil {
		log.Error("Failed to count categories: %v", err)
		return map[string]interface{}{"error": err.Error()}
	}

	return map[string]interface{}{"count": count}
}

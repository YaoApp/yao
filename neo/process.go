package neo

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/gou/rag/driver"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/neo/message"
	"github.com/yaoapp/yao/neo/store"
)

// GetNeo returns the Neo instance
func GetNeo() *DSL {
	if Neo == nil {
		exception.New("Neo is not initialized", 500).Throw()
	}
	return Neo
}

func init() {
	process.RegisterGroup("neo", map[string]process.Handler{
		"write":            ProcessWrite,
		"assistant.create": processAssistantCreate,
		"assistant.save":   processAssistantSave,
		"assistant.delete": processAssistantDelete,
		"assistant.search": processAssistantSearch,
		"assistant.find":   processAssistantFind,
		"assistant.match":  processAssistantMatch, // Match assistant by content and params
	})
}

// ProcessWrite process the write request
func ProcessWrite(process *process.Process) interface{} {
	process.ValidateArgNums(2)

	w, ok := process.Args[0].(gin.ResponseWriter)
	if !ok {
		exception.New("The first argument must be a io.Writer", 400).Throw()
		return nil
	}

	data, ok := process.Args[1].([]interface{})
	if !ok {
		exception.New("The second argument must be a Array", 400).Throw()
		return nil
	}

	for _, new := range data {
		if v, ok := new.(map[string]interface{}); ok {
			newMsg := message.New().Map(v)
			newMsg.Write(w)
		}
	}

	return nil
}

// processAssistantCreate process the assistant create request
func processAssistantCreate(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsMap(0)

	neo := GetNeo()
	if neo.Store == nil {
		exception.New("Neo store is not initialized", 500).Throw()
	}

	id, err := neo.Store.SaveAssistant(data)
	if err != nil {
		exception.New("Failed to create assistant: %s", 500, err.Error()).Throw()
	}

	return id
}

// processAssistantSave process the assistant save request
func processAssistantSave(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsMap(0)

	neo := GetNeo()
	if neo.Store == nil {
		exception.New("Neo store is not initialized", 500).Throw()
	}

	id, err := neo.Store.SaveAssistant(data)
	if err != nil {
		exception.New("Failed to save assistant: %s", 500, err.Error()).Throw()
	}

	return id
}

// processAssistantDelete process the assistant delete request
func processAssistantDelete(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	assistantID := process.ArgsString(0)

	neo := GetNeo()
	if neo.Store == nil {
		exception.New("Neo store is not initialized", 500).Throw()
	}

	err := neo.Store.DeleteAssistant(assistantID)
	if err != nil {
		exception.New("Failed to delete assistant: %s", 500, err.Error()).Throw()
	}

	return gin.H{"message": "ok"}
}

// processAssistantMatch process the assistant match request
func processAssistantMatch(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	content := process.Args[0]
	params := map[string]interface{}{}
	if len(process.Args) > 1 {
		params = process.ArgsMap(1)
	}

	// Limit default to 20
	if _, has := params["limit"]; !has {
		params["limit"] = 20
	}

	// Max limit to 100
	if limit, has := params["limit"]; has {
		switch v := limit.(type) {
		case int:
			if v > 100 {
				params["limit"] = 100
			}
		case string:
			limitInt, err := strconv.Atoi(v)
			if err != nil {
				exception.New("Invalid limit type: %T", 500, limit).Throw()
			}

			params["limit"] = limitInt
			if limitInt > 100 {
				params["limit"] = 100
			}

		default:
			exception.New("Invalid limit type: %T", 500, limit).Throw()
		}
	}

	// Force Using sotre
	forceStore := false
	if store, has := params["store"]; has {
		switch v := store.(type) {
		case bool:
			forceStore = v
		case int:
			forceStore = v == 1
		case string:
			forceStore = v == "true" || v == "1"
		}
	}

	// Rag Support match using RAG
	if Neo.RAG != nil && !forceStore {
		return assistantMatchRAG(content, params)
	}

	// Match using Store
	return assistantMatchStore(content, params)
}

func assistantMatchRAG(content interface{}, params map[string]interface{}) interface{} {
	if Neo == nil {
		exception.New("Neo is not initialized", 500).Throw()
	}

	// Convert content to JSON string
	var contentStr string
	switch v := content.(type) {
	case string:
		contentStr = v
	case []byte:
		contentStr = string(v)
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			exception.New("Failed to convert content to JSON: %s", 500, err.Error()).Throw()
		}
		contentStr = string(bytes)
	}

	// Get limit from params
	limit := 20 // default limit
	if v, has := params["limit"]; has {
		switch lv := v.(type) {
		case int:
			limit = lv
		case string:
			limitInt, err := strconv.Atoi(lv)
			if err == nil {
				limit = limitInt
			}
		}
	}

	// Get min_score from params
	minScore := 0.0 // default min_score
	if v, has := params["min_score"]; has {
		switch lv := v.(type) {
		case float64:
			minScore = lv
		case float32:
			minScore = float64(lv)
		case int:
			minScore = float64(lv)
		case string:
			if score, err := strconv.ParseFloat(lv, 64); err == nil {
				minScore = score
			}
		}
	}

	ctx := context.Background()

	// Get vectors using vectorizer
	vectors, err := Neo.RAG.Vectorizer().Vectorize(ctx, contentStr)
	if err != nil {
		exception.New("Failed to encode content: %s", 500, err.Error()).Throw()
	}

	// Search using RAG engine
	opts := driver.VectorSearchOptions{
		TopK:      limit,
		MinScore:  minScore,
		QueryText: contentStr,
	}

	index := fmt.Sprintf("%sassistants", Neo.RAG.Setting().IndexPrefix)
	results, err := Neo.RAG.Engine().Search(ctx, index, vectors, opts)
	if err != nil {
		exception.New("Failed to search with RAG: %s", 500, err.Error()).Throw()
	}

	// Convert results to assistant data array
	ids := []string{}

	// Collect IDs from search results
	for _, result := range results {
		if result.Metadata != nil {
			if id, ok := result.Metadata["assistant_id"].(string); ok {
				ids = append(ids, id)
			}
		}
	}

	// If no IDs found, return empty array
	if len(ids) == 0 {
		return []map[string]interface{}{}
	}

	// Fetch complete assistant data from store using AssistantIDs
	filter := store.AssistantFilter{
		AssistantIDs: ids,
		Page:         1,
		PageSize:     len(ids),
	}
	res, err := Neo.Store.GetAssistants(filter)
	if err != nil {
		exception.New("get assistants error: %s", 500, err).Throw()
	}

	return res.Data
}

// parseAssistantFilter parse common filter parameters
func parseAssistantFilter(params map[string]interface{}) store.AssistantFilter {
	filter := store.AssistantFilter{}

	// Parse page and pagesize
	if page, ok := params["page"]; ok {
		pageStr := fmt.Sprintf("%v", page)
		if pageInt, err := strconv.Atoi(pageStr); err == nil {
			filter.Page = pageInt
		}
	}

	if pagesize, ok := params["pagesize"]; ok {
		pagesizeStr := fmt.Sprintf("%v", pagesize)
		if pagesizeInt, err := strconv.Atoi(pagesizeStr); err == nil {
			filter.PageSize = pagesizeInt
		}
	}

	// Parse tags
	if tags, ok := params["tags"]; ok {
		switch v := tags.(type) {
		case []interface{}:
			filter.Tags = make([]string, len(v))
			for i, tag := range v {
				filter.Tags[i] = fmt.Sprintf("%v", tag)
			}
		case []string:
			filter.Tags = v
		}
	}

	// Parse keywords
	if keywords, ok := params["keywords"].(string); ok {
		filter.Keywords = keywords
	}

	// Parse connector
	if connector, ok := params["connector"].(string); ok {
		filter.Connector = connector
	}

	// Parse mentionable
	if mentionable, ok := params["mentionable"].(bool); ok {
		filter.Mentionable = &mentionable
	}

	// Parse automated
	if automated, ok := params["automated"].(bool); ok {
		filter.Automated = &automated
	}

	return filter
}

func assistantMatchStore(content interface{}, params map[string]interface{}) interface{} {
	neo := GetNeo()
	if neo.Store == nil {
		exception.New("Neo store is not initialized", 500).Throw()
	}

	// Convert limit to pagesize
	if limit, has := params["limit"]; has {
		params["pagesize"] = limit
	}
	params["page"] = 1

	// Parse content to keywords if not empty
	if content != nil {
		contentStr := fmt.Sprintf("%v", content)
		if contentStr != "" {
			params["keywords"] = contentStr
		}
	}

	filter := parseAssistantFilter(params)
	res, err := neo.Store.GetAssistants(filter)
	if err != nil {
		exception.New("get assistants error: %s", 500, err).Throw()
	}

	return res.Data
}

// processAssistantSearch process the assistant search request
func processAssistantSearch(process *process.Process) interface{} {
	params := process.ArgsMap(0)
	filter := parseAssistantFilter(params)

	// Get assistants
	neo := GetNeo()
	if neo.Store == nil {
		exception.New("Neo store is not initialized", 500).Throw()
	}

	res, err := neo.Store.GetAssistants(filter)
	if err != nil {
		exception.New("get assistants error: %s", 500, err).Throw()
	}

	return res
}

// processAssistantFind process the assistant find request
func processAssistantFind(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	assistantID := process.ArgsString(0)

	neo := GetNeo()
	if neo.Store == nil {
		exception.New("Neo store is not initialized", 500).Throw()
	}

	filter := store.AssistantFilter{
		AssistantID: assistantID,
		Page:        1,
		PageSize:    1,
	}

	res, err := neo.Store.GetAssistants(filter)
	if err != nil {
		exception.New("Failed to find assistant: %s", 500, err.Error()).Throw()
	}

	if len(res.Data) == 0 {
		exception.New("Assistant not found: %s", 404, assistantID).Throw()
	}

	return res.Data[0]
}

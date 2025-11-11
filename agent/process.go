package agent

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/exception"
	"github.com/yaoapp/yao/agent/message"
	store "github.com/yaoapp/yao/agent/store/types"
)

func init() {
	process.RegisterGroup("agent", map[string]process.Handler{
		"write":            ProcessWrite,
		"assistant.create": processAssistantCreate,
		"assistant.save":   processAssistantSave,
		"assistant.delete": processAssistantDelete,
		"assistant.search": processAssistantSearch,
		"assistant.find":   processAssistantFind,
		"assistant.match":  processAssistantMatch, // Match assistant by content and params
	})

	// Neo is deprecated, use agent instead (for backward compatibility, It will be removed in the future)
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

	agent := GetAgent()
	if agent.Store == nil {
		exception.New("Agent store is not initialized", 500).Throw()
	}

	// Convert to AssistantModel
	model, err := store.ToAssistantModel(data)
	if err != nil {
		exception.New("Invalid assistant data: %s", 400, err.Error()).Throw()
	}

	id, err := agent.Store.SaveAssistant(model)
	if err != nil {
		exception.New("Failed to create assistant: %s", 500, err.Error()).Throw()
	}

	return id
}

// processAssistantSave process the assistant save request
func processAssistantSave(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	data := process.ArgsMap(0)

	agent := GetAgent()
	if agent.Store == nil {
		exception.New("Agent store is not initialized", 500).Throw()
	}

	// Convert to AssistantModel
	model, err := store.ToAssistantModel(data)
	if err != nil {
		exception.New("Invalid assistant data: %s", 400, err.Error()).Throw()
	}

	id, err := agent.Store.SaveAssistant(model)
	if err != nil {
		exception.New("Failed to save assistant: %s", 500, err.Error()).Throw()
	}

	return id
}

// processAssistantDelete process the assistant delete request
func processAssistantDelete(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	assistantID := process.ArgsString(0)

	agent := GetAgent()
	if agent.Store == nil {
		exception.New("Agent store is not initialized", 500).Throw()
	}

	err := agent.Store.DeleteAssistant(assistantID)
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

	// Match using Store
	return assistantMatchStore(content, params)
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

	// select
	if sel, ok := params["select"]; ok {
		switch v := sel.(type) {
		case []interface{}:
			filter.Select = []string{}
			for _, field := range v {
				switch v := field.(type) {
				case string:
					filter.Select = append(filter.Select, v)
				case interface{}:
					filter.Select = append(filter.Select, fmt.Sprintf("%v", v))
				}
			}

		case []string:
			filter.Select = v

		case string:
			fields := strings.Split(v, ",")
			filter.Select = fields
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
	agent := GetAgent()
	if agent.Store == nil {
		exception.New("Agent store is not initialized", 500).Throw()
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
	res, err := agent.Store.GetAssistants(filter)
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
	agent := GetAgent()
	if agent.Store == nil {
		exception.New("Agent store is not initialized", 500).Throw()
	}

	locale := "en"
	if len(process.Args) > 1 {
		locale = process.ArgsString(1)
	}

	res, err := agent.Store.GetAssistants(filter, locale)
	if err != nil {
		exception.New("get assistants error: %s", 500, err).Throw()
	}

	return res
}

// processAssistantFind process the assistant find request
func processAssistantFind(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	assistantID := process.ArgsString(0)

	agent := GetAgent()
	if agent.Store == nil {
		exception.New("Agent store is not initialized", 500).Throw()
	}

	filter := store.AssistantFilter{
		AssistantID: assistantID,
		Page:        1,
		PageSize:    1,
	}

	locale := "en"
	if len(process.Args) > 1 {
		locale = process.ArgsString(1)
	}
	res, err := agent.Store.GetAssistants(filter, locale)
	if err != nil {
		exception.New("Failed to find assistant: %s", 500, err.Error()).Throw()
	}

	if len(res.Data) == 0 {
		exception.New("Assistant not found: %s", 404, assistantID).Throw()
	}

	return res.Data[0]
}

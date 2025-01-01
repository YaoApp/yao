package neo

import (
	"fmt"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/gou/process"
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

// processAssistantSearch process the assistant search request
func processAssistantSearch(process *process.Process) interface{} {
	params := process.ArgsMap(0)
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

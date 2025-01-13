package neo

import (
	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/neo/assistant"
	"github.com/yaoapp/yao/neo/rag"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/neo/vision"
	"github.com/yaoapp/yao/neo/vision/driver"
)

// DSL AI assistant
type DSL struct {
	ID            string                 `json:"-" yaml:"-"`
	Name          string                 `json:"name,omitempty" yaml:"name,omitempty"`
	Use           string                 `json:"use,omitempty" yaml:"use,omitempty"` // Which assistant to use default
	Guard         string                 `json:"guard,omitempty" yaml:"guard,omitempty"`
	Connector     string                 `json:"connector" yaml:"connector"`
	StoreSetting  store.Setting          `json:"store" yaml:"store"`
	RAGSetting    rag.Setting            `json:"rag" yaml:"rag"`
	VisionSetting VisionSetting          `json:"vision" yaml:"vision"`
	Option        map[string]interface{} `json:"option" yaml:"option"`
	Prepare       string                 `json:"prepare,omitempty" yaml:"prepare,omitempty"`
	Create        string                 `json:"create,omitempty" yaml:"create,omitempty"`
	Write         string                 `json:"write,omitempty" yaml:"write,omitempty"`
	Prompts       []assistant.Prompt     `json:"prompts,omitempty" yaml:"prompts,omitempty"`
	Allows        []string               `json:"allows,omitempty" yaml:"allows,omitempty"`
	Assistant     assistant.API          `json:"-" yaml:"-"` // The default assistant
	Store         store.Store            `json:"-" yaml:"-"`
	RAG           *rag.RAG               `json:"-" yaml:"-"`
	Vision        *vision.Vision         `json:"-" yaml:"-"`
	GuardHandlers []gin.HandlerFunc      `json:"-" yaml:"-"`
}

// VisionSetting the vision setting
type VisionSetting struct {
	Storage driver.StorageConfig `json:"storage" yaml:"storage"`
	Model   driver.ModelConfig   `json:"model" yaml:"model"`
}

// Mention list
type Mention struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Avatar string `json:"avatar,omitempty"`
	Type   string `json:"type,omitempty"`
}

// Field the context field
type Field struct {
	Name string `json:"name,omitempty"`
	Bind string `json:"bind,omitempty"`
}

// FileUpload the file upload info
type FileUpload struct {
	Bytes       int                    `json:"bytes,omitempty"`        // If upload file, the file bytes
	Name        string                 `json:"name,omitempty"`         // If upload
	ContentType string                 `json:"content_type,omitempty"` // If upload file, the file content type
	Option      map[string]interface{} `json:"option,omitempty"`       // If upload file, the upload option
}

// CreateResponse the response of the create hook
type CreateResponse struct {
	AssistantID string `json:"assistant_id,omitempty"`
	ChatID      string `json:"chat_id,omitempty"`
}

package kb

import (
	"path/filepath"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/graphrag"
	"github.com/yaoapp/gou/graphrag/types"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/config"
)

// Instance is the GraphRag instance
var Instance types.GraphRag = nil

// KnowledgeBase is the Knowledge Base instance
type KnowledgeBase struct {
	Config *Config // Knowledge Base configuration
	*graphrag.GraphRag
}

// Load loads the GraphRag instance
func Load(appConfig config.Config) (*KnowledgeBase, error) {

	configPath := filepath.Join("kb", "kb.yao")
	exists, err := application.App.Exists(configPath)
	if err != nil {
		return nil, err
	}
	if !exists {
		log.Warn("[Knowledge Base] kb.yao file not found, skip loading knowledge base")
		return nil, nil
	}

	// Parse the configuration
	var config Config
	raw, err := application.App.Read(filepath.Join("kb", "kb.yao"))
	if err != nil {
		return nil, err
	}

	err = application.Parse("kb.yao", raw, &config)
	if err != nil {
		return nil, err
	}

	// Create the GraphRag config
	graphRagConfig, err := config.GraphRagConfig()
	if err != nil {
		return nil, err
	}

	// Create the GraphRag instance
	graphRag, err := graphrag.New(graphRagConfig)
	if err != nil {
		return nil, err
	}

	// Set the instance
	instance := &KnowledgeBase{Config: &config, GraphRag: graphRag}

	// Set the instance to the global variable
	Instance = instance
	return instance, nil
}

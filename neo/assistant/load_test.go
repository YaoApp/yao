package assistant

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/neo/store"
	"github.com/yaoapp/yao/test"
)

func prepare(t *testing.T) {
	test.Prepare(t, config.Conf)
}

func TestLoad_LoadPath(t *testing.T) {
	prepare(t)
	defer test.Clean()

	assistant, err := LoadPath("/assistants/modi")
	if err != nil {
		t.Fatal(err)
	}

	// Validate basic properties
	assert.NotNil(t, assistant)
	assert.Equal(t, "modi", assistant.ID)
	assert.Equal(t, "Modi", assistant.Name)
	assert.Equal(t, "https://api.dicebear.com/7.x/bottts/svg?seed=Modi", assistant.Avatar)
	assert.Equal(t, "deepseek", assistant.Connector)
	assert.NotNil(t, assistant.Prompts)
	assert.NotNil(t, assistant.Script)

	// Test non-existent assistant
	_, err = LoadPath("/assistants/non-existent")
	assert.Error(t, err)
}

func TestLoad_LoadStore(t *testing.T) {
	prepare(t)
	defer test.Clean()

	// Test with nil storage
	_, err := LoadStore("test-id")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "storage is not set")

	// Setup mock storage
	mockStore := &mockStore{
		data: map[string]map[string]interface{}{
			"test-id": {
				"assistant_id": "test-id",
				"name":         "Test Assistant",
				"avatar":       "test-avatar",
				"connector":    "gpt-3_5-turbo",
			},
		},
	}
	SetStorage(mockStore)
	defer SetStorage(nil)

	// Test loading from store
	assistant, err := LoadStore("test-id")
	assert.NoError(t, err)
	assert.NotNil(t, assistant)
	assert.Equal(t, "test-id", assistant.ID)
	assert.Equal(t, "Test Assistant", assistant.Name)
	assert.Equal(t, "test-avatar", assistant.Avatar)
	assert.Equal(t, "gpt-3_5-turbo", assistant.Connector)

	// Test cache functionality
	assistant2, err := LoadStore("test-id")
	assert.NoError(t, err)
	assert.Equal(t, assistant, assistant2) // Should be the same instance from cache

	// Test non-existent assistant
	_, err = LoadStore("non-existent")
	assert.Error(t, err)
}

func TestLoad_Cache(t *testing.T) {
	prepare(t)
	defer test.Clean()

	// Clear any existing cache first
	ClearCache()

	// Test cache operations
	SetCache(2) // Set small cache size for testing
	assert.Equal(t, 2, loaded.capacity, "Cache capacity should be 2")

	// Create test assistants
	assistant1 := &Assistant{ID: "id1", Name: "Assistant 1"}
	assistant2 := &Assistant{ID: "id2", Name: "Assistant 2"}
	assistant3 := &Assistant{ID: "id3", Name: "Assistant 3"}

	// Test Put and Get
	loaded.Put(assistant1)
	assert.Equal(t, 1, loaded.Len(), "Cache should have 1 item")

	loaded.Put(assistant2)
	assert.Equal(t, 2, loaded.Len(), "Cache should have 2 items")

	// Test cache hit
	cached, exists := loaded.Get("id1")
	assert.True(t, exists)
	assert.Equal(t, assistant1, cached)

	// Test cache eviction (LRU)
	// At this point: assistant1 is most recently used (due to Get), then assistant2
	loaded.Put(assistant3) // This should evict assistant2 since it's least recently used
	assert.Equal(t, 2, loaded.Len(), "Cache should still have 2 items")
	_, exists = loaded.Get("id2")
	assert.False(t, exists, "assistant2 should have been evicted (least recently used)")
	_, exists = loaded.Get("id1")
	assert.True(t, exists, "assistant1 should still be in cache (was accessed recently)")
	_, exists = loaded.Get("id3")
	assert.True(t, exists, "assistant3 should be in cache (most recently added)")

	// Test clear cache
	ClearCache()
	assert.Nil(t, loaded)

	// Test setting new cache capacity
	SetCache(100)
	assert.NotNil(t, loaded)
}

func TestLoad_Validate(t *testing.T) {
	tests := []struct {
		name    string
		ast     *Assistant
		wantErr bool
	}{
		{
			name: "valid assistant",
			ast: &Assistant{
				ID:        "test-id",
				Name:      "Test Assistant",
				Connector: "test-connector",
			},
			wantErr: false,
		},
		{
			name: "missing id",
			ast: &Assistant{
				Name:      "Test Assistant",
				Connector: "test-connector",
			},
			wantErr: true,
		},
		{
			name: "missing name",
			ast: &Assistant{
				ID:        "test-id",
				Connector: "test-connector",
			},
			wantErr: true,
		},
		{
			name: "missing connector",
			ast: &Assistant{
				ID:   "test-id",
				Name: "Test Assistant",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.ast.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Assistant.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestLoad_Clone(t *testing.T) {
	// Create a test assistant with all fields populated
	original := &Assistant{
		ID:          "test-id",
		Type:        "test-type",
		Name:        "Test Assistant",
		Avatar:      "test-avatar",
		Connector:   "test-connector",
		Path:        "test-path",
		BuiltIn:     true,
		Sort:        1,
		Description: "test description",
		Tags:        []string{"tag1", "tag2"},
		Readonly:    true,
		Mentionable: true,
		Automated:   true,
		Options:     map[string]interface{}{"key": "value"},
		Prompts:     []Prompt{{Role: "system", Content: "test"}},
		Flows:       []map[string]interface{}{{"step": "test"}},
	}

	// Clone the assistant
	clone := original.Clone()

	// Verify all fields are correctly cloned
	assert.Equal(t, original.ID, clone.ID)
	assert.Equal(t, original.Type, clone.Type)
	assert.Equal(t, original.Name, clone.Name)
	assert.Equal(t, original.Avatar, clone.Avatar)
	assert.Equal(t, original.Connector, clone.Connector)
	assert.Equal(t, original.Path, clone.Path)
	assert.Equal(t, original.BuiltIn, clone.BuiltIn)
	assert.Equal(t, original.Sort, clone.Sort)
	assert.Equal(t, original.Description, clone.Description)
	assert.Equal(t, original.Tags, clone.Tags)
	assert.Equal(t, original.Readonly, clone.Readonly)
	assert.Equal(t, original.Mentionable, clone.Mentionable)
	assert.Equal(t, original.Automated, clone.Automated)
	assert.Equal(t, original.Options, clone.Options)
	assert.Equal(t, original.Prompts, clone.Prompts)
	assert.Equal(t, original.Flows, clone.Flows)

	// Verify deep copy by modifying original
	original.Tags[0] = "modified"
	original.Options["key"] = "modified"
	original.Flows[0]["step"] = "modified"
	assert.NotEqual(t, original.Tags[0], clone.Tags[0])
	assert.NotEqual(t, original.Options["key"], clone.Options["key"])
	assert.NotEqual(t, original.Flows[0]["step"], clone.Flows[0]["step"])

	// Test nil case
	var nilAssistant *Assistant
	assert.Nil(t, nilAssistant.Clone())
}

func TestLoad_Update(t *testing.T) {
	// Create a test assistant
	ast := &Assistant{
		ID:        "test-id",
		Name:      "Original Name",
		Connector: "original-connector",
	}

	// Test updating various fields
	updates := map[string]interface{}{
		"name":        "Updated Name",
		"avatar":      "updated-avatar",
		"description": "Updated description",
		"connector":   "updated-connector",
		"type":        "updated-type",
		"sort":        2,
		"mentionable": true,
		"automated":   true,
		"tags":        []string{"new-tag"},
		"options":     map[string]interface{}{"new": "value"},
	}

	err := ast.Update(updates)
	assert.NoError(t, err)

	// Verify updates
	assert.Equal(t, "Updated Name", ast.Name)
	assert.Equal(t, "updated-avatar", ast.Avatar)
	assert.Equal(t, "Updated description", ast.Description)
	assert.Equal(t, "updated-connector", ast.Connector)
	assert.Equal(t, "updated-type", ast.Type)
	assert.Equal(t, 2, ast.Sort)
	assert.True(t, ast.Mentionable)
	assert.True(t, ast.Automated)
	assert.Equal(t, []string{"new-tag"}, ast.Tags)
	assert.Equal(t, map[string]interface{}{"new": "value"}, ast.Options)

	// Test nil assistant
	var nilAssistant *Assistant
	err = nilAssistant.Update(updates)
	assert.Error(t, err)

	// Test invalid update that would make the assistant invalid
	invalidUpdates := map[string]interface{}{
		"name": "",
	}
	err = ast.Update(invalidUpdates)
	assert.Error(t, err)
}

func TestLoadBuiltIn(t *testing.T) {
	prepare(t)
	defer test.Clean()

	// Clear any existing cache and storage
	ClearCache()
	SetStorage(nil)

	// Create a mock store to verify built-in assistants are saved
	mockStore := &mockStore{
		data: make(map[string]map[string]interface{}),
	}
	SetStorage(mockStore)
	SetCache(100)

	// Test loading built-in assistants
	err := LoadBuiltIn()
	assert.NoError(t, err)

	// Verify Modi assistant was loaded
	assistant, exists := loaded.Get("modi")
	assert.True(t, exists, "Modi assistant should be loaded in cache")
	if exists {
		assert.Equal(t, "modi", assistant.ID)
		assert.Equal(t, "Modi", assistant.Name)
		assert.Equal(t, "deepseek", assistant.Connector)
		assert.True(t, assistant.BuiltIn)
		assert.True(t, assistant.Readonly)
		assert.NotNil(t, assistant.Prompts)
		assert.NotNil(t, assistant.Script)
	}

}

// mockStore implements store.Store interface for testing
type mockStore struct {
	data map[string]map[string]interface{}
}

func (m *mockStore) GetAssistant(id string) (map[string]interface{}, error) {
	if data, ok := m.data[id]; ok {
		return data, nil
	}
	return nil, fmt.Errorf("assistant not found: %s", id)
}

// Add other required interface methods with empty implementations
func (m *mockStore) GetThread(id string) (map[string]interface{}, error)  { return nil, nil }
func (m *mockStore) GetMessage(id string) (map[string]interface{}, error) { return nil, nil }
func (m *mockStore) GetFile(id string) (map[string]interface{}, error)    { return nil, nil }
func (m *mockStore) CreateAssistant(data map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) CreateThread(data map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) CreateMessage(data map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) CreateFile(data map[string]interface{}) (map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) UpdateAssistant(id string, data map[string]interface{}) error { return nil }
func (m *mockStore) UpdateThread(id string, data map[string]interface{}) error    { return nil }
func (m *mockStore) UpdateMessage(id string, data map[string]interface{}) error   { return nil }
func (m *mockStore) UpdateFile(id string, data map[string]interface{}) error      { return nil }
func (m *mockStore) DeleteAssistant(id string) error                              { return nil }
func (m *mockStore) DeleteThread(id string) error                                 { return nil }
func (m *mockStore) DeleteMessage(id string) error                                { return nil }
func (m *mockStore) DeleteFile(id string) error                                   { return nil }
func (m *mockStore) ListAssistants(query map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) ListThreads(query map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) ListMessages(query map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) ListFiles(query map[string]interface{}) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) DeleteAllChats(id string) error            { return nil }
func (m *mockStore) DeleteChat(id string, chatID string) error { return nil }
func (m *mockStore) GetAssistants(filter store.AssistantFilter) (*store.AssistantResponse, error) {
	return nil, nil
}
func (m *mockStore) GetChat(id string, chatID string) (*store.ChatInfo, error) { return nil, nil }
func (m *mockStore) GetChats(id string, filter store.ChatFilter) (*store.ChatGroupResponse, error) {
	return nil, nil
}
func (m *mockStore) GetHistory(id string, chatID string) ([]map[string]interface{}, error) {
	return nil, nil
}
func (m *mockStore) SaveAssistant(assistant map[string]interface{}) (interface{}, error) {
	return nil, nil
}
func (m *mockStore) SaveHistory(sid string, messages []map[string]interface{}, cid string, context map[string]interface{}) error {
	return nil
}
func (m *mockStore) UpdateChatTitle(sid string, cid string, title string) error   { return nil }
func (m *mockStore) DeleteAssistants(filter store.AssistantFilter) (int64, error) { return 0, nil }
func (m *mockStore) GetAssistantTags() ([]string, error)                          { return []string{}, nil }

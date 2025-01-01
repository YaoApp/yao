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

func TestAssistant_LoadPath(t *testing.T) {
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

func TestAssistant_LoadStore(t *testing.T) {
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
				"connector":    "test-connector",
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
	assert.Equal(t, "test-connector", assistant.Connector)

	// Test cache functionality
	assistant2, err := LoadStore("test-id")
	assert.NoError(t, err)
	assert.Equal(t, assistant, assistant2) // Should be the same instance from cache

	// Test non-existent assistant
	_, err = LoadStore("non-existent")
	assert.Error(t, err)
}

func TestAssistant_Cache(t *testing.T) {
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
func (m *mockStore) UpdateChatTitle(sid string, cid string, title string) error { return nil }

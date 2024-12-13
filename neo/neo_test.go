package neo

// type customResponseRecorder struct {
// 	*httptest.ResponseRecorder
// 	closeChannel chan bool
// }

// func (r *customResponseRecorder) CloseNotify() <-chan bool {
// 	return r.closeChannel
// }

// func newCustomResponseRecorder() *customResponseRecorder {
// 	return &customResponseRecorder{
// 		ResponseRecorder: httptest.NewRecorder(),
// 		closeChannel:     make(chan bool, 1),
// 	}
// }

// func TestDSL_Prompts(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer Test_clean(t)

// 	resetDB()
// 	neo := &DSL{
// 		Prompts: []Prompt{
// 			{Role: "system", Content: "You are a helpful assistant", Name: "ai"},
// 			{Role: "user", Content: "Hello", Name: "user"},
// 		},
// 		ConversationSetting: conversation.Setting{
// 			Connector: "default",
// 			Table:     "chat_messages",
// 		},
// 	}
// 	err := neo.newConversation()
// 	assert.NoError(t, err)

// 	prompts := neo.prompts()
// 	assert.Equal(t, 2, len(prompts))
// 	assert.Equal(t, "system", prompts[0]["role"])
// 	assert.Equal(t, "You are a helpful assistant", prompts[0]["content"])
// 	assert.Equal(t, "ai", prompts[0]["name"])
// }

// func TestDSL_ChatMessages(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer Test_clean(t)

// 	resetDB()
// 	neo := &DSL{
// 		Prompts: []Prompt{
// 			{Role: "system", Content: "You are a helpful assistant"},
// 		},
// 		ConversationSetting: conversation.Setting{
// 			Connector: "default",
// 			Table:     "chat_messages",
// 		},
// 	}

// 	err := neo.newConversation()
// 	assert.NoError(t, err)

// 	ctx := Context{
// 		Sid:    "test-session",
// 		ChatID: "test-chat",
// 	}

// 	messages, err := neo.chatMessages(ctx, "Hello AI")
// 	assert.NoError(t, err)
// 	assert.Equal(t, 2, len(messages))
// 	assert.Equal(t, "system", messages[0]["role"])
// 	assert.Equal(t, "user", messages[1]["role"])
// 	assert.Equal(t, "Hello AI", messages[1]["content"])
// }

// func TestDSL_Answer(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer Test_clean(t)

// 	gin.SetMode(gin.TestMode)
// 	w := newCustomResponseRecorder()
// 	c, _ := gin.CreateTestContext(w)

// 	ctx := Context{
// 		Sid:     "test-session",
// 		ChatID:  "test-chat",
// 		Context: context.Background(),
// 	}

// 	resetDB()
// 	neo := &DSL{
// 		Connector: "gpt-3_5-turbo",
// 		Option: map[string]interface{}{
// 			"temperature": 0.7,
// 			"max_tokens":  150,
// 		},
// 		Prompts: []Prompt{
// 			{Role: "system", Content: "You are a helpful assistant"},
// 		},
// 		ConversationSetting: conversation.Setting{
// 			Connector: "default",
// 			Table:     "chat_messages",
// 		},
// 	}

// 	err := neo.newAI()
// 	assert.NoError(t, err)

// 	err = neo.newConversation()
// 	assert.NoError(t, err)

// 	c.Request = httptest.NewRequest("POST", "/chat", nil)

// 	neo.AI = &mockAI{}

// 	err = neo.Answer(ctx, "Hello AI", c)
// 	assert.NoError(t, err)
// }

// // func TestDSL_NewAI(t *testing.T) {
// // 	test.Prepare(t, config.Conf)
// // 	defer Test_clean(t)

// // 	tests := []struct {
// // 		name      string
// // 		connector string
// // 		wantErr   string
// // 	}{
// // 		{
// // 			name:      "Mock AI",
// // 			connector: "mock",
// // 			wantErr:   "",
// // 		},
// // 		{
// // 			name:      "Specific mock model",
// // 			connector: "mock:gpt-4",
// // 			wantErr:   "",
// // 		},
// // 		{
// // 			name:      "Invalid connector",
// // 			connector: "invalid-connector",
// // 			wantErr:   "AI connector invalid-connector not found",
// // 		},
// // 	}

// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			neo := &DSL{
// // 				Connector: tt.connector,
// // 			}
// // 			neo.newConversation()

// // 			assert.Panics(t, func() {
// // 				neo.newAI()
// // 			})

// // 		})
// // 	}
// // }

// func TestDSL_Select(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer Test_clean(t)

// 	resetDB()
// 	neo := &DSL{
// 		ConversationSetting: conversation.Setting{
// 			Connector: "default",
// 			Table:     "chat_messages",
// 		},
// 	}

// 	err := neo.newConversation()
// 	assert.NoError(t, err)

// 	err = neo.Select("invalid-model")
// 	assert.Error(t, err)

// 	// err = neo.Select("gpt-3_5-turbo")
// 	// assert.NoError(t, err)
// 	// assert.NotNil(t, neo.AI)

// }

// // func TestDSL_NewConversation(t *testing.T) {
// // 	test.Prepare(t, config.Conf)
// // 	defer Test_clean(t)

// // 	tests := []struct {
// // 		name      string
// // 		connector string
// // 		wantErr   bool
// // 	}{
// // 		{
// // 			name:      "Default connector",
// // 			connector: "default",
// // 			wantErr:   false,
// // 		},
// // 		{
// // 			name:      "Empty connector",
// // 			connector: "",
// // 			wantErr:   false,
// // 		},
// // 		{
// // 			name:      "Invalid connector",
// // 			connector: "invalid-connector",
// // 			wantErr:   true,
// // 		},
// // 	}

// // 	for _, tt := range tests {
// // 		t.Run(tt.name, func(t *testing.T) {
// // 			neo := &DSL{
// // 				ConversationSetting: conversation.Setting{
// // 					Connector: tt.connector,
// // 				},
// // 			}
// // 			assert.Panics(t, func() {
// // 				neo.newConversation()
// // 			})
// // 		})
// // 	}
// // }

// func TestDSL_SaveHistory(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer Test_clean(t)

// 	neo := &DSL{
// 		ConversationSetting: conversation.Setting{
// 			Connector: "default",
// 			Table:     "chat_messages",
// 		},
// 	}

// 	resetDB()
// 	err := neo.newConversation()
// 	assert.NoError(t, err)

// 	messages := []map[string]interface{}{
// 		{
// 			"role":    "user",
// 			"content": "Hello",
// 			"name":    "test-user",
// 		},
// 	}

// 	content := []byte("Hi there!")
// 	neo.saveHistory("test-session", "test-chat", content, messages)

// 	// Verify the history was saved
// 	history, err := neo.Conversation.GetHistory("test-session", "test-chat")
// 	assert.NoError(t, err)
// 	assert.NotEmpty(t, history)
// }

// func TestDSL_Send(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer Test_clean(t)

// 	gin.SetMode(gin.TestMode)
// 	w := httptest.NewRecorder()
// 	c, _ := gin.CreateTestContext(w)

// 	resetDB()
// 	neo := &DSL{
// 		ConversationSetting: conversation.Setting{
// 			Connector: "default",
// 			Table:     "chat_messages",
// 		},
// 	}

// 	err := neo.newConversation()
// 	assert.NoError(t, err)
// 	ctx := Context{
// 		Sid:    "test-session",
// 		ChatID: "test-chat",
// 	}

// 	msg := &message.JSON{
// 		Message: &message.Message{Text: "Test message"},
// 	}
// 	messages := []map[string]interface{}{
// 		{"role": "user", "content": "Hello"},
// 	}
// 	content := []byte("Test content")

// 	err = neo.send(ctx, msg, messages, content, c)
// 	assert.NoError(t, err)
// }

// func Test_clean(t *testing.T) {
// 	defer test.Clean()

// }

// func resetDB() {
// 	sch := capsule.Global.Schema()
// 	sch.DropTable("chat_messages")
// }

// type mockAI struct{}

// func (m *mockAI) ChatCompletionsWith(ctx context.Context, messages []map[string]interface{}, options map[string]interface{}, callback func([]byte) int) (interface{}, *exception.Exception) {
// 	callback([]byte(`{"choices":[{"delta":{"content":"Mock response"}}]}`))
// 	callback([]byte(`{"choices":[{"finish_reason":"stop"}]}`))
// 	return nil, nil
// }

// func (m *mockAI) ChatCompletions(messages []map[string]interface{}, options map[string]interface{}, callback func([]byte) int) (interface{}, *exception.Exception) {
// 	return nil, nil
// }

// func (m *mockAI) GetContent(response interface{}) (string, *exception.Exception) {
// 	return "Mock content", nil
// }

// func (m *mockAI) Embeddings(input interface{}, user string) (interface{}, *exception.Exception) {
// 	return nil, nil
// }

// func (m *mockAI) Tiktoken(input string) (int, error) {
// 	return 0, nil
// }

// func (m *mockAI) MaxToken() int {
// 	return 4096
// }

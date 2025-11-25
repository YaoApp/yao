package jsapi

// func TestOutputConstructor(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer test.Clean()

// 	tests := []struct {
// 		name        string
// 		script      string
// 		expectError bool
// 	}{
// 		{
// 			name: "Create Output with context",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					return output !== undefined && output !== null;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Create Output without context should fail",
// 			script: `
// 				function test(ctx) {
// 					try {
// 						const output = new Output();
// 						return false;
// 					} catch (e) {
// 						return e.toString().includes("context argument");
// 					}
// 				}
// 			`,
// 			expectError: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			ctx := agentContext.New(context.Background(), nil, "test-chat-123", "")
// 			ctx.AssistantID = "test-assistant-456"

// 			// Execute test script with v8.Call
// 			res, err := v8.Call(v8.CallOptions{}, tt.script, &ctx)
// 			if tt.expectError {
// 				assert.Error(t, err)
// 				return
// 			}

// 			assert.NoError(t, err)
// 			assert.True(t, res.(bool))
// 		})
// 	}
// }

// func TestOutputSend(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer test.Clean()

// 	tests := []struct {
// 		name        string
// 		script      string
// 		expectError bool
// 		validate    func(*testing.T, *agentContext.Context)
// 	}{
// 		{
// 			name: "Send text message with object",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					output.Send({
// 						type: "text",
// 						props: { content: "Hello World" }
// 					});
// 					return true;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send text message with string shorthand",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					output.Send("Hello World");
// 					return true;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send message with all fields",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					output.Send({
// 						type: "text",
// 						props: { content: "Test" },
// 						id: "msg-1",
// 						delta: true,
// 						done: false,
// 						delta_path: "content",
// 						delta_action: "append",
// 						metadata: {
// 							timestamp: 1234567890,
// 							sequence: 1,
// 							trace_id: "trace-123"
// 						}
// 					});
// 					return true;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send error message",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					output.Send({
// 						type: "error",
// 						props: {
// 							message: "Something went wrong",
// 							code: "ERR_001"
// 						}
// 					});
// 					return true;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send without message should fail",
// 			script: `
// 				function test(ctx) {
// 					try {
// 						const output = new Output(ctx);
// 						output.Send();
// 						return false;
// 					} catch (e) {
// 						return e.toString().includes("message argument");
// 					}
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send message without type should fail",
// 			script: `
// 				function test(ctx) {
// 					try {
// 						const output = new Output(ctx);
// 						output.Send({ props: { content: "test" } });
// 						return false;
// 					} catch (e) {
// 						return e.toString().includes("type is required");
// 					}
// 				}
// 			`,
// 			expectError: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Create context with mock writer
// 			ctx := agentContext.New(context.Background(), nil, "test-chat", "")
// 			ctx.Writer = &mockWriter{}

// 			// Execute test script with v8.Call
// 			res, err := v8.Call(v8.CallOptions{}, tt.script, &ctx)
// 			if tt.expectError {
// 				assert.Error(t, err)
// 				return
// 			}

// 			assert.NoError(t, err)
// 			assert.True(t, res.(bool))

// 			if tt.validate != nil {
// 				tt.validate(t, &ctx)
// 			}
// 		})
// 	}
// }

// func TestOutputSendGroup(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer test.Clean()

// 	tests := []struct {
// 		name        string
// 		script      string
// 		expectError bool
// 	}{
// 		{
// 			name: "Send message group",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					output.SendGroup({
// 						id: "group-1",
// 						messages: [
// 							{ type: "text", props: { content: "Message 1" } },
// 							{ type: "text", props: { content: "Message 2" } }
// 						]
// 					});
// 					return true;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send group with metadata",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					output.SendGroup({
// 						id: "group-1",
// 						messages: [
// 							{ type: "text", props: { content: "Test" } }
// 						],
// 						metadata: {
// 							timestamp: 1234567890,
// 							sequence: 1
// 						}
// 					});
// 					return true;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send empty group",
// 			script: `
// 				function test(ctx) {
// 					const output = new Output(ctx);
// 					output.SendGroup({
// 						id: "group-1",
// 						messages: []
// 					});
// 					return true;
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send group without id should fail",
// 			script: `
// 				function test(ctx) {
// 					try {
// 						const output = new Output(ctx);
// 						output.SendGroup({
// 							messages: [
// 								{ type: "text", props: { content: "Test" } }
// 							]
// 						});
// 						return false;
// 					} catch (e) {
// 						return e.toString().includes("id is required");
// 					}
// 				}
// 			`,
// 			expectError: false,
// 		},
// 		{
// 			name: "Send group without messages should fail",
// 			script: `
// 				function test(ctx) {
// 					try {
// 						const output = new Output(ctx);
// 						output.SendGroup({ id: "group-1" });
// 						return false;
// 					} catch (e) {
// 						return e.toString().includes("messages is required");
// 					}
// 				}
// 			`,
// 			expectError: false,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			// Create context with mock writer
// 			ctx := agentContext.New(context.Background(), nil, "test-chat", "")
// 			ctx.Writer = &mockWriter{}

// 			// Execute test script with v8.Call
// 			res, err := v8.Call(v8.CallOptions{}, tt.script, &ctx)
// 			if tt.expectError {
// 				assert.Error(t, err)
// 				return
// 			}

// 			assert.NoError(t, err)
// 			assert.True(t, res.(bool))
// 		})
// 	}
// }

// func TestOutputChaining(t *testing.T) {
// 	test.Prepare(t, config.Conf)
// 	defer test.Clean()

// 	script := `
// 		function test(ctx) {
// 			const output = new Output(ctx);

// 			// Send should return the output object for chaining
// 			const result = output.Send("Message 1");

// 			// Should be able to chain sends
// 			output.Send("Message 2").Send("Message 3");

// 			return result !== undefined;
// 		}
// 	`

// 	ctx := agentContext.New(context.Background(), nil, "test-chat", "")
// 	ctx.Writer = &mockWriter{}

// 	// Execute test script with v8.Call
// 	res, err := v8.Call(v8.CallOptions{}, script, &ctx)
// 	assert.NoError(t, err)
// 	assert.True(t, res.(bool))
// }

// // mockWriter is a mock implementation of http.ResponseWriter for testing
// type mockWriter struct {
// 	data   [][]byte
// 	header http.Header
// }

// func (w *mockWriter) Header() http.Header {
// 	if w.header == nil {
// 		w.header = make(http.Header)
// 	}
// 	return w.header
// }

// func (w *mockWriter) Write(p []byte) (n int, err error) {
// 	w.data = append(w.data, p)
// 	return len(p), nil
// }

// func (w *mockWriter) WriteHeader(statusCode int) {}

// func (w *mockWriter) Flush() {}

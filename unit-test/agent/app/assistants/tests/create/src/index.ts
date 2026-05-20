// @ts-nocheck

function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const content = messages[0]?.content || "";

  switch (content) {
    case "return_null":
      return null;

    case "return_undefined":
      return undefined;

    case "return_empty":
      return {};

    case "return_full":
      return {
        messages: [
          { role: "system", content: "You are a helpful assistant." },
          { role: "user", content: "Hello!" },
        ],
        audio: { voice: "alloy", format: "mp3" },
        temperature: 0.7,
        max_tokens: 2000,
        max_completion_tokens: 1500,
        metadata: { test: "full_response", user_id: "test_user_123" },
      };

    case "return_partial":
      return {
        messages: [{ role: "user", content: "Partial test" }],
        temperature: 0.5,
      };

    case "return_process":
      return {
        messages: [
          { role: "system", content: "Processing request" },
          { role: "user", content: "process_call" },
        ],
        metadata: { test: "process_call", processed: true },
      };

    case "adjust_context":
      return {
        messages: [{ role: "system", content: "Context fields will be adjusted" }],
        connector: "adjusted-connector",
        locale: "zh-cn",
        theme: "dark",
        route: "/adjusted/route",
        metadata: {
          adjusted: true,
          original_assistant: ctx.assistant_id,
        },
      };

    case "nested_script_call":
      return {
        messages: [
          { role: "system", content: "Nested script execution" },
          { role: "user", content: "nested_level_1" },
        ],
        metadata: { test: "nested_script_call", depth: 1 },
      };

    case "deep_nested_call":
      return {
        messages: [
          { role: "system", content: "Deep nested script execution" },
          { role: "user", content: "nested_level_3" },
        ],
        metadata: { test: "deep_nested_call", depth: 3 },
      };

    default:
      return { messages: [{ role: "user", content: content }] };
  }
}

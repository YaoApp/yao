// @ts-nocheck

function Next(
  ctx: agent.Context,
  payload: agent.NextHookPayload
): agent.NextHookResponse | null {
  const completion = payload.completion;
  const messages = payload.messages;
  const tools = payload.tools;
  const error = payload.error;

  const testScenario = getTestScenario(messages);

  switch (testScenario) {
    case "return_null":
      return null;
    case "return_undefined":
      return undefined;
    case "return_empty":
      return {};
    case "return_custom_data":
      return {
        data: { message: "Custom response from Next Hook", test: true, timestamp: Date.now() },
      };
    case "return_data_with_metadata":
      return {
        data: { result: "success", value: 42 },
        metadata: { hook: "next", processed: true },
      };
    case "return_delegate":
      return {
        delegate: {
          agent_id: "tests.hook-echo",
          messages: [{ role: "user", content: "Hello from delegated agent" }],
        },
      };
    case "verify_payload":
      return scenarioVerifyPayload(messages, completion, tools, error);
    case "verify_tools":
      return scenarioVerifyTools(tools);
    case "conditional_delegate":
      return scenarioConditionalDelegate(completion);
    case "handle_error":
      return scenarioHandleError(error);
    default:
      return {
        data: {
          message: "Next Hook executed successfully",
          scenario: testScenario,
          completion_content: completion ? completion.content : null,
          message_count: messages ? messages.length : 0,
          tool_count: tools ? tools.length : 0,
        },
        metadata: { hook: "next", test: "default" },
      };
  }
}

function getTestScenario(messages: any[]): string {
  if (!messages || messages.length === 0) return "default";
  for (let i = messages.length - 1; i >= 0; i--) {
    const msg = messages[i];
    if (msg.role === "user" && msg.content && typeof msg.content === "string") {
      return msg.content.toLowerCase().trim();
    }
  }
  return "default";
}

function scenarioVerifyPayload(messages: any[], completion: any, tools: any[], error: string): any {
  const checks = [];
  checks.push(messages && Array.isArray(messages) ? `Messages: ${messages.length} messages` : "Messages: MISSING");
  if (completion) {
    checks.push(`Completion: ${completion.content ? "has content" : "no content"}`);
    checks.push(`Completion.Usage: ${completion.usage ? "present" : "missing"}`);
  } else {
    checks.push("Completion: MISSING");
  }
  if (tools && Array.isArray(tools)) {
    checks.push(`Tools: ${tools.length} tool calls`);
  } else {
    checks.push("Tools: none");
  }
  checks.push(error ? `Error: ${error}` : "Error: none");
  return { data: { validation: "success", checks: checks } };
}

function scenarioVerifyTools(tools: any[]): any {
  if (!tools || tools.length === 0) return { data: { message: "No tool calls to verify" } };
  return {
    data: {
      message: "Tool calls processed",
      total_tools: tools.length,
      successful: tools.filter((t: any) => !t.error).length,
      failed: tools.filter((t: any) => t.error).length,
    },
  };
}

function scenarioConditionalDelegate(completion: any): any {
  if (completion && completion.content) {
    const content = completion.content.toLowerCase();
    if (content.includes("delegate") || content.includes("forward")) {
      return {
        delegate: {
          agent_id: "tests.hook-echo",
          messages: [{ role: "user", content: "Delegated due to keyword match" }],
        },
      };
    }
  }
  return { data: { message: "No delegation needed", reason: "No matching keywords" } };
}

function scenarioHandleError(error: string): any {
  if (error) {
    return { data: { message: "Error was handled by Next Hook", error: error, recovered: true } };
  }
  return { data: { message: "No error to handle" } };
}

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

    case "verify_context":
      return scenarioVerifyContext(ctx);

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

    case "adjust_uses":
      return {
        messages: [{ role: "system", content: "Uses configuration will be adjusted" }],
        uses: {
          vision: "mcp:vision-server",
          audio: "mcp:audio-server",
          search: "agent",
          fetch: "mcp:fetch-server",
        },
        metadata: { uses_adjusted: true, chat_id: ctx.chat_id },
      };

    case "adjust_uses_force":
      return {
        messages: [{ role: "system", content: "Uses configuration will be forced" }],
        uses: { vision: "tests.vision-test", audio: "mcp:audio-server" },
        force_uses: true,
        metadata: { uses_forced: true, chat_id: ctx.chat_id },
      };

    default:
      return { messages: [{ role: "user", content: content }] };
  }
}

function scenarioVerifyContext(ctx: agent.Context): agent.Create {
  const validations: string[] = [];
  let allValid = true;

  function check(name: string, actual: any, expected: any): void {
    if (actual === expected) {
      validations.push(`${name}:true`);
    } else {
      validations.push(`${name}:false:${actual}`);
      allValid = false;
    }
  }

  if (ctx.authorized) {
    check("authorized.user_id", ctx.authorized.user_id, "test-user-123");
    check("authorized.team_id", ctx.authorized.team_id, "test-team-456");
    check("authorized.tenant_id", ctx.authorized.tenant_id, "test-tenant-789");
  } else {
    validations.push("authorized:false:missing");
    allValid = false;
  }

  check("chat_id", ctx.chat_id, "chat-test-create-hook");
  check("assistant_id", ctx.assistant_id, "tests.hook-echo");
  check("locale", ctx.locale, "en-us");
  check("theme", ctx.theme, "light");

  if (ctx.client) {
    check("client.type", ctx.client.type, "web");
    check("client.user_agent", ctx.client.user_agent, "TestAgent/1.0");
    check("client.ip", ctx.client.ip, "127.0.0.1");
  } else {
    validations.push("client:false:missing");
    allValid = false;
  }

  check("referer", ctx.referer, "api");
  check("accept", ctx.accept, "cui-web");

  return {
    messages: [
      { role: "system", content: allValid ? "success:all_fields_validated" : "failure:validation_failed" },
      { role: "assistant", content: validations.join("\n") },
    ],
  };
}

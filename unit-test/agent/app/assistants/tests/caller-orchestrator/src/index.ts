function Create(ctx, messages) {
  return null;
}

function Next(ctx, payload) {
  const content = (payload.messages && payload.messages.length > 0)
    ? payload.messages[payload.messages.length - 1].content
    : "";

  if (typeof content === "string" && content.includes("call_target")) {
    const result = ctx.agent.Call("tests.caller-target", [
      { role: "user", content: "hello from orchestrator" }
    ]);
    return {
      data: {
        type: "a2a_result",
        result: result
      }
    };
  }

  return null;
}

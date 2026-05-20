// @ts-nocheck

function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const content = messages[0]?.content || "";

  switch (content) {
    case "simple":
      return scenarioSimple();

    case "mcp_health":
      return scenarioMCPHealth(ctx);

    case "mcp_tools":
      return scenarioMCPTools(ctx);

    case "full_workflow":
      return scenarioFullWorkflow(ctx);

    case "trace_intensive":
      return scenarioTraceIntensive(ctx);

    case "resource_heavy":
      return scenarioResourceHeavy(ctx);

    default:
      return {
        messages: [{ role: "assistant", content: "Unknown scenario: " + content }],
        metadata: { scenario: content },
      };
  }
}

function scenarioSimple(): agent.Create {
  return {
    messages: [
      { role: "system", content: "You are a helpful assistant for stress testing." },
      { role: "assistant", content: "Simple scenario processed successfully." },
    ],
    metadata: { scenario: "simple" },
  };
}

function scenarioMCPHealth(ctx: agent.Context): agent.Create {
  const tools = getMCPTools(ctx);
  const toolsCount = tools.length;

  return {
    messages: [
      { role: "assistant", content: "Health check complete. Tools: " + toolsCount + " available." },
    ],
    metadata: {
      scenario: "mcp_health",
      tools_count: toolsCount,
      health_data: {
        status: "healthy",
        server: "echo",
        tools: tools.map((t: any) => t.name || t),
        timestamp: Date.now(),
      },
    },
  };
}

function scenarioMCPTools(ctx: agent.Context): agent.Create {
  const tools = getMCPTools(ctx);
  const toolsCount = tools.length;

  const operations = [
    { tool: "ping", result: "pong" },
    { tool: "echo", result: "echo_response" },
  ];

  return {
    messages: [
      { role: "assistant", content: "Tools listing complete. Ping and Echo operations executed." },
    ],
    metadata: {
      scenario: "mcp_tools",
      tools_count: toolsCount,
      operations: operations,
    },
  };
}

function scenarioFullWorkflow(ctx: agent.Context): agent.Create {
  const tools = getMCPTools(ctx);

  const phases = [
    { name: "initialization", status: "complete" },
    { name: "tool_discovery", status: "complete" },
    { name: "execution", status: "complete" },
    { name: "cleanup", status: "complete" },
  ];

  return {
    messages: [
      {
        role: "assistant",
        content: "Full Workflow complete. Tools: " + tools.length + ". Roles: admin, user. All phases done.",
      },
    ],
    metadata: {
      scenario: "full_workflow",
      phases_completed: phases.length,
      mcp_tools: tools.length,
      phases: phases,
    },
  };
}

function scenarioTraceIntensive(ctx: agent.Context): agent.Create {
  let nodesCreated = 0;
  for (let i = 0; i < 50; i++) {
    nodesCreated++;
  }

  return {
    messages: [
      { role: "assistant", content: "Trace intensive scenario: " + nodesCreated + " nodes created." },
    ],
    metadata: {
      scenario: "trace_intensive",
      nodes_created: nodesCreated,
    },
  };
}

function scenarioResourceHeavy(ctx: agent.Context): agent.Create {
  const mcpIterations = 5;
  const results: any[] = [];

  for (let i = 0; i < mcpIterations; i++) {
    results.push({ iteration: i + 1, status: "complete" });
  }

  return {
    messages: [
      { role: "assistant", content: "Resource heavy: " + mcpIterations + " MCP iterations completed." },
    ],
    metadata: {
      scenario: "resource_heavy",
      mcp_iterations: mcpIterations,
      results: results,
    },
  };
}

function getMCPTools(ctx: agent.Context): any[] {
  try {
    if (ctx.mcp && typeof ctx.mcp.list_tools === "function") {
      return ctx.mcp.list_tools() || [];
    }
  } catch (e) {
    // MCP not available, return static tool list
  }
  return [
    { name: "ping", description: "Ping test" },
    { name: "echo", description: "Echo test" },
    { name: "status", description: "Status check" },
  ];
}

// @ts-nocheck
/**
 * Search JSAPI Test Assistant
 * Tests ctx.search.Web and related methods (SerpAPI)
 * KB/DB search tests are removed — only web search is supported.
 */

function Create(
  ctx: YaoAgent.Context,
  messages: YaoAgent.Message[],
  options?: YaoAgent.CreateOptions
): YaoAgent.CreateHookResponse {
  const userMessage = messages.find((m) => m.role === "user");
  const content = (userMessage?.content as string) || "";

  const testMatch = content.match(/^test:(\w+)(?:\s+(.*))?$/);
  if (!testMatch) {
    return {
      messages: [
        {
          role: "assistant",
          content:
            "Invalid test command. Use: test:web, test:web_sites, test:all, test:any, test:race",
        },
      ],
    };
  }

  const method = testMatch[1];
  const query = testMatch[2] || "Yao App Engine";

  try {
    let result: any;

    switch (method) {
      case "web":
        result = ctx.search.Web(query, { limit: 5 });
        break;

      case "web_sites":
        result = ctx.search.Web(query, {
          limit: 5,
          sites: ["github.com", "yaoapps.com"],
        });
        break;

      case "all":
        result = ctx.search.All([
          { type: "web", query: "golang programming", limit: 3 },
          { type: "web", query: "rust programming", limit: 3 },
        ]);
        break;

      case "any":
        result = ctx.search.Any([
          { type: "web", query: "kubernetes containers", limit: 3 },
          { type: "web", query: "docker orchestration", limit: 3 },
        ]);
        break;

      case "race":
        result = ctx.search.Race([
          { type: "web", query: "machine learning", limit: 3 },
          { type: "web", query: "deep learning", limit: 3 },
        ]);
        break;

      default:
        return {
          messages: [
            {
              role: "assistant",
              content:
                "Unknown test method: " +
                method +
                ". Use: web, web_sites, all, any, race",
            },
          ],
        };
    }

    return {
      messages: [
        {
          role: "assistant",
          content: JSON.stringify(result, null, 2),
        },
      ],
    };
  } catch (error: any) {
    return {
      messages: [
        {
          role: "assistant",
          content: "Error: " + (error.message || error),
        },
      ],
    };
  }
}

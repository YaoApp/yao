// @ts-nocheck

function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const content = messages[0]?.content || "";

  switch (content) {
    case "disable_search":
      return { search: false };

    case "enable_web_search":
      return {
        search: {
          need_search: true,
          types: ["web"],
        },
      };

    case "enable_all_search":
      return {
        search: {
          need_search: true,
          types: ["web", "kb"],
        },
      };

    default:
      return {};
  }
}

// @ts-nocheck

function Create(ctx: agent.Context, messages: agent.Message[]): agent.Create {
  const content = messages[0]?.content || "";

  switch (content) {
    case "use_chat_preset":
      return { prompt_preset: "chat" };

    case "use_task_preset":
      return { prompt_preset: "task" };

    case "disable_global":
      return { disable_global_prompts: true };

    default:
      return {};
  }
}

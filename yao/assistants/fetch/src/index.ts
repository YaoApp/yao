/**
 * Fetch Helper Agent - Create Hook
 *
 * Extracts URLs from the user's last message, fetches their content via http.Get,
 * converts HTML to Markdown, and injects the content into the conversation.
 */

// @ts-nocheck

const URL_REGEX = /https?:\/\/[^\s<>"')\]]+/gi;
const MAX_CONTENT_LENGTH = 5000;

/**
 * Create hook - extracts URLs from the last user message, fetches and converts content
 */
function Create(
  ctx: agent.Context,
  messages: agent.Message[],
  options?: Record<string, any>
): agent.HookCreateResponse | null {
  const lastMsg = findLastUserMessage(messages);
  if (!lastMsg) return null;

  const text = extractText(lastMsg);
  if (!text) return null;

  const urls = text.match(URL_REGEX);
  if (!urls || urls.length === 0) return null;

  const seen = new Set<string>();
  const fetched: string[] = [];

  for (const raw of urls) {
    const url = raw.replace(/[.,;:!?)}\]]+$/, "");
    if (seen.has(url)) continue;
    seen.add(url);

    try {
      const resp = http.Get(url, {}, { "User-Agent": "YaoFetchHelper/1.0" });
      if (resp.code >= 200 && resp.code < 300 && resp.data) {
        let body =
          typeof resp.data === "string"
            ? resp.data
            : JSON.stringify(resp.data);

        const contentType: string = extractContentType(resp.headers);
        if (contentType.includes("text/html") || looksLikeHTML(body)) {
          try {
            body = Process("text.HTMLToMarkdown", body);
          } catch (_) {}
        }

        if (body.length > MAX_CONTENT_LENGTH) {
          body = body.substring(0, MAX_CONTENT_LENGTH) + "\n... [truncated]";
        }
        fetched.push(`--- Content from ${url} ---\n${body}\n--- End ---`);
      }
    } catch (_) {
      // Skip failed fetches silently
    }
  }

  if (fetched.length === 0) return null;

  return {
    messages: [
      {
        role: "user",
        content: `${text}\n\nFetched content:\n\n${fetched.join("\n\n")}`,
      },
    ],
  };
}

function findLastUserMessage(
  messages: agent.Message[]
): agent.Message | null {
  for (let i = messages.length - 1; i >= 0; i--) {
    if (messages[i].role === "user") return messages[i];
  }
  return null;
}

function extractText(msg: agent.Message): string {
  if (typeof msg.content === "string") return msg.content;
  if (Array.isArray(msg.content)) {
    return msg.content
      .filter((p: any) => p.type === "text" && p.text)
      .map((p: any) => p.text)
      .join("\n");
  }
  return "";
}

function extractContentType(headers: Record<string, any>): string {
  if (!headers) return "";
  for (const key of Object.keys(headers)) {
    if (key.toLowerCase() === "content-type") {
      const val = headers[key];
      return (Array.isArray(val) ? val[0] : val || "").toLowerCase();
    }
  }
  return "";
}

function looksLikeHTML(text: string): boolean {
  const trimmed = text.trimStart();
  return (
    trimmed.startsWith("<!") ||
    trimmed.startsWith("<html") ||
    trimmed.startsWith("<HTML")
  );
}

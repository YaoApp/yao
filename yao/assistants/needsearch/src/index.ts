/**
 * Need Search Agent - Next Hook
 * Parses LLM response and extracts search intent with error tolerance
 */

// @ts-nocheck

interface SearchResult {
  need_search: boolean;
  search_types: string[];
  confidence: number;
}

/**
 * Next hook - processes search intent response
 * Uses json.Parse for fault-tolerant JSON parsing
 */
function Next(
  ctx: agent.Context,
  payload: agent.NextHookPayload
): agent.NextHookResponse | null {
  const completion = payload.completion;

  // No completion, return null for standard handling
  if (!completion || !completion.content) {
    return null;
  }

  // Remove markdown code block if present
  let content = completion.content.trim();
  if (content.startsWith("```json")) {
    content = content.slice(7); // Remove ```json
  } else if (content.startsWith("```")) {
    content = content.slice(3); // Remove ```
  }
  if (content.endsWith("```")) {
    content = content.slice(0, -3); // Remove trailing ```
  }
  content = content.trim();

  // Default result
  let result: SearchResult = {
    need_search: false,
    search_types: [],
    confidence: 0,
  };

  try {
    // Use json.Parse for fault-tolerant parsing
    const parsed = Process("json.Parse", content) as {
      need_search?: boolean;
      search_types?: string[];
      confidence?: number;
    } | null;

    if (parsed) {
      result.need_search = Boolean(parsed.need_search);
      result.search_types = Array.isArray(parsed.search_types)
        ? parsed.search_types.filter(
            (t) =>
              typeof t === "string" &&
              ["web", "kb", "db"].includes(t.toLowerCase())
          )
        : [];
      result.confidence =
        typeof parsed.confidence === "number"
          ? Math.min(1, Math.max(0, parsed.confidence))
          : 0.5;
    }
  } catch (e) {
    // If json.Parse fails, try to extract from text
    result = extractFromText(content);
  }

  // Return parsed result
  return {
    data: result,
  };
}

/**
 * Extract search intent from plain text when JSON parsing fails
 */
function extractFromText(text: string): SearchResult {
  const lower = text.toLowerCase();

  // Check for explicit indicators
  const needSearch =
    lower.includes("true") ||
    lower.includes("need") ||
    lower.includes("search") ||
    lower.includes("web") ||
    lower.includes("kb") ||
    lower.includes("db");

  const noSearch =
    lower.includes("false") ||
    lower.includes("no search") ||
    lower.includes("not need");

  // Extract search types
  const searchTypes: string[] = [];
  if (lower.includes("web")) searchTypes.push("web");
  if (lower.includes("kb") || lower.includes("knowledge"))
    searchTypes.push("kb");
  if (lower.includes("db") || lower.includes("database"))
    searchTypes.push("db");

  // Determine need_search
  const need = noSearch ? false : needSearch && searchTypes.length > 0;

  return {
    need_search: need,
    search_types: need ? searchTypes : [],
    confidence: 0.5, // Low confidence for text extraction
  };
}

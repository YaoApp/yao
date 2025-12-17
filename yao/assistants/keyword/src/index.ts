/**
 * Keyword Extraction Agent - Next Hook
 * Parses LLM response and extracts keywords with error tolerance
 */

// @ts-nocheck

/**
 * Next hook - processes keyword extraction response
 * Uses text.ExtractJSON for fault-tolerant JSON extraction from LLM output
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

  const content = completion.content;
  let keywords: string[] = [];

  try {
    // Use text.ExtractJSON for fault-tolerant extraction
    // Handles markdown code blocks, broken JSON, etc.
    const parsed = Process("text.ExtractJSON", content) as {
      keywords?: string[];
    } | null;

    if (parsed && Array.isArray(parsed.keywords)) {
      keywords = parsed.keywords.filter(
        (k) => typeof k === "string" && k.trim().length > 0
      );
    }
  } catch (e) {
    // If extraction fails, try to extract keywords from text
    keywords = extractKeywordsFromText(content);
  }

  // If still no keywords, try extracting from raw text
  if (keywords.length === 0) {
    keywords = extractKeywordsFromText(content);
  }

  // Return parsed keywords
  return {
    data: {
      keywords: keywords,
    },
  };
}

/**
 * Extract keywords from plain text when JSON parsing fails
 * Handles formats like:
 * - Comma-separated: "keyword1, keyword2, keyword3"
 * - Line-separated: "keyword1\nkeyword2\nkeyword3"
 * - Bullet points: "- keyword1\n- keyword2"
 * - Numbered: "1. keyword1\n2. keyword2"
 */
function extractKeywordsFromText(text: string): string[] {
  const keywords: string[] = [];

  // Remove common prefixes/suffixes
  let cleaned = text
    .replace(/^[\s\S]*?keywords?[\s:：]*\[?/i, "") // Remove "keywords:" prefix
    .replace(/\][\s\S]*$/, "") // Remove trailing ]
    .trim();

  // Try line-by-line extraction
  const lines = cleaned.split(/[\n\r]+/);

  for (const line of lines) {
    // Remove bullet points, numbers, quotes
    let keyword = line
      .replace(/^[\s\-\*\•\d\.]+/, "") // Remove bullets/numbers
      .replace(/^["'`]+|["'`]+$/g, "") // Remove quotes
      .replace(/,\s*$/, "") // Remove trailing comma
      .trim();

    // Skip empty or too long
    if (keyword.length > 0 && keyword.length < 100) {
      // Split by comma if contains multiple
      if (keyword.includes(",")) {
        const parts = keyword.split(",").map((p) => p.trim());
        for (const part of parts) {
          if (part.length > 0 && part.length < 100) {
            keywords.push(part);
          }
        }
      } else {
        keywords.push(keyword);
      }
    }
  }

  // Deduplicate
  return [...new Set(keywords)];
}

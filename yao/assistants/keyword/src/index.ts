/**
 * Keyword Extraction Agent - Next Hook
 * Parses LLM response and extracts keywords with weight
 * Format: ["keyword:weight", ...] -> [{k, w}, ...]
 */

// @ts-nocheck

/** Keyword with weight */
interface Keyword {
  k: string; // keyword
  w: number; // weight (0.1-1.0)
}

/**
 * Next hook - processes keyword extraction response
 * Parses format: ["keyword1:0.9", "keyword2:0.8", ...]
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
  let keywords: Keyword[] = [];

  try {
    // Extract JSON array from response
    const parsed = Process("text.ExtractJSON", content) as string[] | null;

    if (parsed && Array.isArray(parsed)) {
      keywords = parseKeywordArray(parsed);
    }
  } catch (e) {
    // If extraction fails, try to extract from text
    keywords = extractKeywordsFromText(content);
  }

  // If still no keywords, try extracting from raw text
  if (keywords.length === 0) {
    keywords = extractKeywordsFromText(content);
  }

  // Sort by weight descending and limit to 5
  keywords = keywords.sort((a, b) => b.w - a.w).slice(0, 5);

  // Return parsed keywords
  return {
    data: {
      keywords: keywords,
    },
  };
}

/**
 * Parse keyword array format: ["keyword:weight", ...]
 * Examples: ["AI:0.9", "机器学习:0.8", "deep learning:0.7"]
 */
function parseKeywordArray(items: (string | any)[]): Keyword[] {
  const keywords: Keyword[] = [];

  for (const item of items) {
    if (typeof item === "string") {
      const parsed = parseKeywordString(item);
      if (parsed) {
        keywords.push(parsed);
      }
    } else if (item && typeof item === "object" && item.k) {
      // Fallback: handle {k, w} format
      const k = String(item.k).trim();
      const w =
        typeof item.w === "number" ? Math.min(1.0, Math.max(0.1, item.w)) : 0.5;
      if (k.length > 0) {
        keywords.push({ k, w });
      }
    }
  }

  return keywords;
}

/**
 * Parse single keyword string: "keyword:weight" or "keyword"
 */
function parseKeywordString(str: string): Keyword | null {
  const trimmed = str.trim().replace(/^["']+|["']+$/g, ""); // Remove quotes
  if (!trimmed) return null;

  // Try to split by last colon (keyword may contain colons)
  const lastColonIdx = trimmed.lastIndexOf(":");
  if (lastColonIdx > 0) {
    const keyword = trimmed.substring(0, lastColonIdx).trim();
    const weightStr = trimmed.substring(lastColonIdx + 1).trim();
    const weight = parseFloat(weightStr);

    if (keyword && !isNaN(weight)) {
      return {
        k: keyword,
        w: Math.min(1.0, Math.max(0.1, weight)),
      };
    }
  }

  // No weight found, return with default weight
  return { k: trimmed, w: 0.5 };
}

/**
 * Extract keywords from plain text when JSON parsing fails
 */
function extractKeywordsFromText(text: string): Keyword[] {
  const keywords: Keyword[] = [];

  // Try to find array-like content
  const arrayMatch = text.match(/\[([^\]]+)\]/);
  if (arrayMatch) {
    const items = arrayMatch[1].split(",");
    for (const item of items) {
      const parsed = parseKeywordString(item);
      if (parsed) {
        keywords.push(parsed);
      }
    }
    if (keywords.length > 0) return keywords;
  }

  // Fallback: line-by-line extraction
  const lines = text.split(/[\n\r,]+/);
  let defaultWeight = 1.0;

  for (const line of lines) {
    let cleaned = line
      .replace(/^[\s\-\*\•\d\.\[\]"'`]+/, "") // Remove prefixes
      .replace(/[\]"'`]+$/, "") // Remove suffixes
      .trim();

    if (cleaned.length > 0 && cleaned.length < 100) {
      const parsed = parseKeywordString(cleaned);
      if (parsed) {
        // Use parsed weight or assign decreasing default
        if (parsed.w === 0.5) {
          parsed.w = Math.max(0.1, defaultWeight);
          defaultWeight -= 0.1;
        }
        keywords.push(parsed);
      }
    }
  }

  return keywords;
}

/**
 * QueryDSL Generator Agent - Hooks
 *
 * Scenarios (via metadata.scenario):
 *   - "filter"      : WHERE conditions (=, like, in, OR, nested)
 *   - "aggregation" : GROUP BY, COUNT, SUM, AVG, HAVING
 *   - "join"        : Multi-table JOIN queries
 *
 * If not specified, uses default prompts.yml (basic queries)
 */

// @ts-nocheck

// Valid scenario names that map to prompt presets in prompts/ directory
const VALID_SCENARIOS = ["filter", "aggregation", "join", "complex"];

/**
 * Create hook - selects prompt preset based on metadata.scenario
 */
function Create(
  ctx: agent.Context,
  messages: agent.Message[],
  options?: Record<string, any>
): agent.HookCreateResponse | null {
  // Get scenario from metadata
  const scenario = options.metadata?.scenario || ctx.metadata?.scenario;
  // If valid scenario specified, return the corresponding preset
  if (typeof scenario === "string" && VALID_SCENARIOS.includes(scenario)) {
    return {
      prompt_preset: scenario,
    };
  }

  // No preset - use default prompts.yml
  return null;
}

/**
 * Next hook - extracts QueryDSL JSON from LLM response
 */
function Next(
  ctx: agent.Context,
  payload: agent.NextHookPayload
): agent.NextHookResponse | null {
  const completion = payload.completion;

  if (!completion || !completion.content) {
    return {
      data: { error: "empty_response", message: "LLM returned empty content" },
    };
  }

  const content = completion.content;

  // Use text.ExtractJSON for fault-tolerant extraction
  const dsl = Process("text.ExtractJSON", content);
  if (dsl && typeof dsl === "object" && Object.keys(dsl).length > 0) {
    return { data: dsl };
  }

  // Extraction failed, return error with original content
  return {
    data: {
      error: "extraction_failed",
      message: "Failed to extract JSON from LLM response",
      raw: content,
    },
  };
}

package types

// InspirationReport - P0 output (simple markdown for LLM)
type InspirationReport struct {
	Clock   *ClockContext `json:"clock"`   // time context
	Content string        `json:"content"` // markdown text for LLM
}

// Content is markdown like:
// ## Summary
// ...
// ## Highlights
// - [High] Sales up 50%
// - [Medium] New lead from BigCorp
// ## Opportunities
// ...
// ## Risks
// ...
// ## World News
// ...
// ## Pending
// ...

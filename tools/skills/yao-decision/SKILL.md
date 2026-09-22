---
name: yao-decision
description: Typed decision expert. ALWAYS invoke this skill when you need structured classification, scoring, or probability estimation — routing tickets, grading content, detecting intent, or any task requiring typed answers with calibrated confidence.
---

# Decision Tools

Use these tools when you need structured, typed decisions rather than free-form LLM text. Decision models return machine-readable answers with probabilities and confidence scores.

## decision_decide

Send a state (context) and one or more typed questions to a decision model.

### Question Types

| Type | Purpose | Criteria | Answer Fields |
|------|---------|----------|---------------|
| `choice` | Classify into one of N options | `{option: description}` map | `choice`, `probabilities`, `confidence` |
| `score` | Rate on a scale | `[level_labels]` array | `score`, `legend`, `probabilities`, `confidence` |
| `noul` | Estimate a probability (0-1) | _(none)_ | `noul` |

### Basic usage (choice + noul):
```bash
tai tool decision_decide --state "Customer says: my billing is wrong, I was charged twice" --questions '{"department":{"type":"choice","instructions":"Route to the right department","criteria":{"billing":"Payment issues","technical":"Product bugs","sales":"Purchase inquiries"}},"is_urgent":{"type":"noul","instructions":"Is this urgent?"}}'
```

### With score question:
```bash
tai tool decision_decide --state "The agent resolved the issue in 2 minutes with a clear explanation" --questions '{"quality":{"type":"score","instructions":"Rate the support quality","criteria":["Poor","Below average","Average","Good","Excellent"]}}'
```

### With explicit provider:
```bash
tai tool decision_decide --state "some context" --questions '{"q1":{"type":"noul","instructions":"Is this positive?"}}' --provider t862699292876.typesafe
```

### With model override:
```bash
tai tool decision_decide --state "some context" --questions '{"q1":{"type":"noul","instructions":"Is this positive?"}}' --model jev-latest
```

### Batch decisions (multiple states, same questions):
```bash
tai tool decision_decide --states '["Customer A says billing is wrong","Customer B loves the product","Customer C wants a refund"]' --questions '{"sentiment":{"type":"choice","instructions":"Classify sentiment","criteria":{"positive":"Happy","negative":"Unhappy","neutral":"Neither"}}}'
```

### Reading from files:
```bash
tai tool decision_decide --state_file customer_message.txt --questions_file routing_questions.json
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| state | string/object/array | yes* | The context for the decision (*mutually exclusive with states) |
| questions | JSON object | yes | Map of question_id to question definition |
| model | string | no | Model name override (default: provider's configured model) |
| provider | string | no | Provider connector ID (default: decision role or first available) |
| timeout | integer | no | Timeout in seconds (default: 60) |
| states | array | no* | Array of state items for batch decisions (*mutually exclusive with state) |
| state_file | string | no | Read state from file (tai CLI only) |
| questions_file | string | no | Read questions from JSON file (tai CLI only) |

### Response Format

Single state:
```json
{
  "answers": {
    "department": {
      "type": "choice",
      "choice": "billing",
      "probabilities": {"billing": 0.92, "technical": 0.05, "sales": 0.03},
      "confidence": 0.89
    },
    "is_urgent": {
      "type": "noul",
      "noul": 0.73
    }
  },
  "model": "jev-1.13.0",
  "usage": {"input_tokens": 245, "output_tokens": 38}
}
```

> Note: `model` in the response is the resolved concrete model ID (e.g. `jev-1.13.0`), not the alias you requested (e.g. `jev-latest`).

Batch states:
```json
{
  "results": [
    {"state_index": 0, "answers": {...}, "model": "jev-1.13.0", "usage": {...}},
    {"state_index": 1, "answers": {...}, "model": "jev-1.13.0", "usage": {...}},
    {"state_index": 2, "error": "timed out after 60s"}
  ]
}
```

## decision_providers

List available decision providers and their models.

```bash
tai tool decision_providers
```

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| capability | string | no | Filter by capability (default: `decision`) |

Returns a list of providers with available models, connector IDs, and availability status.

## Constraints

- `state` and `states` are mutually exclusive; provide one or the other.
- Each question must have `type` (`choice`/`score`/`noul`) and `instructions`.
- `criteria` is required for `choice` (map) and `score` (array); `noul` must **not** have criteria.
- Batch states must not contain empty strings or null values.
- `timeout` defaults to 60s; values ≤ 0 are treated as the default.
- Batch mode runs up to 5 concurrent calls; timeout applies to the entire batch.
- Response `model` is the resolved concrete model ID (e.g. `jev-1.13.0`), not the requested alias.

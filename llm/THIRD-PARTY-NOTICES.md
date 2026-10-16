# Third-Party Notices

This package (`yao/llm`) contains code that references and reuses material
from the following open-source project.

## PiG — Provider-Integration Gateway

| | |
|---|---|
| **Repository** | <https://github.com/MichaelKinsy/PiG> |
| **Version referenced** | v0.4.1 |
| **License** | MIT — Copyright Hewlett Packard Enterprise Development LP |

### What was reused

- **Type vocabulary.** Core enumeration types (`API`, `ModelType`,
  `ProviderID`, `ThinkingLevel`, `ThinkingLevelMap`,
  `SamplingParamsByThinkingLevel`) are modelled after the corresponding
  definitions in `github.com/MichaelKinsy/PiG/ai`. String constants were
  kept identical so that data serialised by either system remains
  interoperable.

- **Struct field layout.** `Provider` mirrors `ai.Provider` +
  `ai.ProviderMetadata`; `Model` mirrors `ai.Model` + `ai.GeneratedModel`.
  Field names and JSON tags are aligned one-to-one where the domain concept
  overlaps.

- **Preset model catalogue.** The 42 YAML files under `presets/` were
  generated from PiG's `GeneratedModel` static catalogue (1 621 model
  entries across all providers). Provider metadata, model capabilities,
  context-window sizes, and cost tiers originate from that catalogue.

- **Route configuration types.** `PiAiRoute` and `PiAiModelConfig`
  reproduce the cordis.yml route schema consumed by the `dsh-llm-pi-ai`
  plugin, which is part of the PiG ecosystem.

### What was not reused

All CRUD logic, database schema (`.mod.yao` models), scope-cascade
resolution, registry caching, legacy-connector migration, and HTTP/Process
handler code is original to Yao.

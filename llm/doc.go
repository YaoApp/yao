// Package llm manages LLM provider and model configuration, replacing
// the gou/connector dependency for LLM concerns.
//
// Storage fields use PiG ai vocabulary so that yao/llm.Model → ai.Model
// is a trivial field copy, eliminating the ExtractProfile translation layer.
//
// # Data model
//
// Three Yao system tables back the package:
//
//   - __yao.llm.provider  — one row per API endpoint (credentials + protocol)
//   - __yao.llm.model     — one row per deployable model under a provider
//   - __yao.llm.role_binding — maps a role name to a provider+model pair
//
// # Scope cascade
//
// Role bindings support a three-level override: User > Team > System.
// [ResolveRole] walks the cascade and returns the first match.
//
// # Preset catalogue
//
// Embedded YAML files under presets/ supply 1600+ model definitions for
// auto-detection. [LookupPreset] and [ListPresets] query this catalogue.
//
// # Migration
//
// [MigrateFromLegacy] performs a one-time translation from the old
// KV-store-based llmprovider registry into the new DB tables. The
// migration is idempotent and guarded by a marker in __yao.config.
//
// # Conversion
//
// [ToConnectorProfile] bridges to the legacy ConnectorProfile consumed by
// the tai runner. [ToPiAiRoute] produces the dsh cordis.yml route struct
// directly — the zero-translation path for the dsh runner.
package llm

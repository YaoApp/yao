package llm

import (
	"strings"

	goumodel "github.com/yaoapp/gou/model"
	goustore "github.com/yaoapp/gou/store"
	"github.com/yaoapp/yao/llmprovider"
	"github.com/yaoapp/yao/setting"
)

const migrationConfigKey = "llm.migrated"

// MigrateFromLegacy performs a one-time migration from the old llmprovider
// KV store to the new yao/llm database tables. Skips when:
//   - the migration marker already exists in __yao.config, or
//   - llmprovider.Global is nil, or
//   - the old KV store has no providers
func MigrateFromLegacy() error {
	if isMigrated() {
		return nil
	}

	if llmprovider.Global == nil {
		setMigrated()
		return nil
	}

	oldProviders, err := llmprovider.Global.List(
		&llmprovider.ProviderFilter{Source: llmprovider.ProviderSourceAll},
		true, // withKey: get plain-text API keys
	)
	if err != nil || len(oldProviders) == 0 {
		setMigrated()
		return nil
	}

	// Group old providers by (ProviderID, scope) to avoid UNIQUE collisions.
	groups := groupProviders(oldProviders)

	// Track old-key → new-provider-ref for role migration, and
	// old-variant-ID → merged-model-ID for model resolution.
	keyToRef := make(map[string]int, len(oldProviders))
	variantMap := make(map[string]string)

	for _, g := range groups {
		p := Provider{
			Name:       g.name,
			ProviderID: g.key.providerID,
			API:        g.api,
			BaseURL:    g.baseURL,
			APIKey:     g.apiKey,
			Enabled:    g.enabled,
			Status:     g.status,
			PresetKey:  g.presetKey,
			Scope:      g.key.scope,
			TeamID:     g.key.teamID,
			UserID:     g.key.userID,
		}

		// Translate all old models, then merge same-wire-name variants.
		translated := make([]Model, 0, len(g.models))
		for _, mi := range g.models {
			translated = append(translated, translateModel(mi, p.API))
		}
		merged := mergeModelVariants(translated, variantMap)

		id, err := CreateProvider(&p)
		if err != nil {
			continue // skip on UNIQUE constraint or other errors
		}
		for _, key := range g.oldKeys {
			keyToRef[key] = id
		}

		for j := range merged {
			merged[j].ProviderRef = id
			ApplyPreset(&merged[j], string(p.ProviderID))
			if _, err := CreateModel(&merged[j]); err != nil {
				continue
			}
		}
	}

	// Migrate role bindings (non-fatal).
	_ = MigrateRoles(keyToRef, variantMap)

	setMigrated()
	return nil
}

// ---------------------------------------------------------------------------
// Provider grouping — merge old per-connector providers by vendor identity
// ---------------------------------------------------------------------------

// groupKey uniquely identifies a new Provider target.
// baseURL is included so that providers with different endpoints (e.g.
// official OpenAI vs Tao Service) are kept separate even when both map
// to the same ProviderID.
type groupKey struct {
	providerID ProviderID
	baseURL    string
	scope      Scope
	teamID     string
	userID     string
}

// providerGroup collects old Providers that map to the same new Provider.
type providerGroup struct {
	key       groupKey
	name      string
	api       API
	baseURL   string
	apiKey    string
	presetKey string
	enabled   bool
	status    string
	oldKeys   []string
	models    []llmprovider.ModelInfo
}

// groupProviders partitions old providers into groups that share the same
// (ProviderID, baseURL, scope) tuple. Providers with the same ProviderID
// and scope but different baseURLs stay separate. When that happens the
// second group's ProviderID gets a numeric suffix to avoid the DB UNIQUE
// constraint on (provider_id, scope, team_id, user_id).
func groupProviders(oldProviders []llmprovider.Provider) []providerGroup {
	idx := map[groupKey]int{} // key → position in result
	var groups []providerGroup

	for i := range oldProviders {
		old := &oldProviders[i]
		scope, teamID, userID := translateScope(old.Owner)
		gk := groupKey{
			providerID: inferProviderID(old),
			baseURL:    old.APIURL,
			scope:      scope,
			teamID:     teamID,
			userID:     userID,
		}

		pos, exists := idx[gk]
		if !exists {
			presetKey := old.PresetKey
			if presetKey == "" {
				presetKey = old.Key
			}
			status := old.Status
			if status == "" {
				status = "unconfigured"
			}
			groups = append(groups, providerGroup{
				key:       gk,
				name:      old.Name,
				api:       inferAPI(old),
				baseURL:   old.APIURL,
				apiKey:    old.APIKey,
				presetKey: presetKey,
				enabled:   old.Enabled,
				status:    status,
				oldKeys:   []string{old.Key},
				models:    append([]llmprovider.ModelInfo(nil), old.Models...),
			})
			idx[gk] = len(groups) - 1
		} else {
			g := &groups[pos]
			g.oldKeys = append(g.oldKeys, old.Key)
			g.models = append(g.models, old.Models...)
			if old.Enabled {
				g.enabled = true
			}
		}
	}

	// Dedup: when multiple groups share the same (providerID, scope) but
	// differ by baseURL, derive a slug from the provider Name for the
	// duplicates (e.g. "MiniMax M2.1 Highspeed" → "minimax-m2.1-highspeed").
	type scopeKey struct {
		providerID ProviderID
		scope      Scope
		teamID     string
		userID     string
	}
	seen := map[scopeKey]int{}
	for i := range groups {
		sk := scopeKey{groups[i].key.providerID, groups[i].key.scope, groups[i].key.teamID, groups[i].key.userID}
		n := seen[sk]
		seen[sk] = n + 1
		if n > 0 {
			groups[i].key.providerID = slugifyName(groups[i].name)
		}
	}

	return groups
}

// slugifyName converts a display name to a URL-safe ProviderID slug.
// "MiniMax M2.1 Highspeed" → "minimax-m2.1-highspeed"
// "GPT-4.1 Mini" → "gpt-4.1-mini"
// "Qwen3.8-27B NVFP4 (No Thinking)" → "qwen3.8-27b-nvfp4-no-thinking"
func slugifyName(name string) ProviderID {
	s := strings.ToLower(name)
	s = strings.ReplaceAll(s, " ", "-")

	// Keep only alphanumeric, hyphen, dot; drop everything else.
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '.' {
			b.WriteRune(r)
		}
	}
	s = b.String()

	// Collapse consecutive hyphens and trim edges.
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}
	s = strings.Trim(s, "-.")
	if s == "" {
		s = "provider"
	}
	return ProviderID(s)
}

// ---------------------------------------------------------------------------
// Model variant merging — collapse same-wire-name entries into one Model
// ---------------------------------------------------------------------------

// mergeModelVariants groups translated Models by their wire model name
// (ModelAPI when set, else ModelID) and merges same-name entries into a
// single Model with a unified ThinkingLevelMap. Non-base variant IDs are
// recorded in Metadata["variant_aliases"].
//
// variantMap is populated with oldVariantID → mergedModelID for every merged
// variant so that role bindings can be remapped.
func mergeModelVariants(models []Model, variantMap map[string]string) []Model {
	type bucket struct {
		baseIdx  int   // index of the base model (wire name == ModelID)
		indices  []int // all indices in this bucket
		wireName string
	}

	bucketMap := map[string]*bucket{}
	var bucketOrder []string

	for i := range models {
		wn := models[i].ModelAPI
		if wn == "" {
			wn = models[i].ModelID
		}
		b, exists := bucketMap[wn]
		if !exists {
			b = &bucket{baseIdx: -1, wireName: wn}
			bucketMap[wn] = b
			bucketOrder = append(bucketOrder, wn)
		}
		b.indices = append(b.indices, i)
		if models[i].ModelID == wn {
			b.baseIdx = len(b.indices) - 1 // position within indices
		}
	}

	out := make([]Model, 0, len(bucketOrder))
	for _, wn := range bucketOrder {
		b := bucketMap[wn]
		if len(b.indices) == 1 {
			out = append(out, models[b.indices[0]])
			continue
		}

		// Pick base: prefer the entry whose ModelID equals the wire name.
		basePos := 0
		if b.baseIdx >= 0 {
			basePos = b.baseIdx
		}
		merged := models[b.indices[basePos]]

		// Ensure ModelID is the wire name for the merged record.
		merged.ModelID = wn
		if merged.ModelAPI == "" {
			merged.ModelAPI = wn
		}

		// Merge ThinkingLevelMap from all variants.
		if merged.ThinkingLevelMap == nil {
			merged.ThinkingLevelMap = make(ThinkingLevelMap)
		}
		for _, idx := range b.indices {
			for lv, v := range models[idx].ThinkingLevelMap {
				if _, exists := merged.ThinkingLevelMap[lv]; !exists {
					merged.ThinkingLevelMap[lv] = v
				}
			}
			// Take largest context/output values.
			if models[idx].ContextWindow > merged.ContextWindow {
				merged.ContextWindow = models[idx].ContextWindow
			}
			if models[idx].MaxOutputTokens > merged.MaxOutputTokens {
				merged.MaxOutputTokens = models[idx].MaxOutputTokens
			}
			if models[idx].Enabled {
				merged.Enabled = true
			}
			if models[idx].Reasoning {
				merged.Reasoning = true
			}
		}

		// Recompute MaxThinking from the merged map.
		if len(merged.ThinkingLevelMap) > 0 {
			keys := make(map[string]*string, len(merged.ThinkingLevelMap))
			for k, v := range merged.ThinkingLevelMap {
				keys[string(k)] = v
			}
			merged.MaxThinking = inferMaxThinking(keys)
		}

		// Record variant aliases (all non-base ModelIDs).
		var aliases []string
		for _, idx := range b.indices {
			mid := models[idx].ModelID
			if mid != wn {
				aliases = append(aliases, mid)
				variantMap[mid] = wn
			}
		}
		if len(aliases) > 0 {
			if merged.Metadata == nil {
				merged.Metadata = make(JSONMap)
			}
			merged.Metadata["variant_aliases"] = aliases
		}

		out = append(out, merged)
	}
	return out
}

// ---------------------------------------------------------------------------
// Translate — the one-time connector→PiG vocabulary translator (§6)
// ---------------------------------------------------------------------------

// Translate converts an old llmprovider.Provider to the new Provider + []Model.
// This is the only code that does connector→PiG value-domain translation,
// and it runs exactly once during migration.
func Translate(old *llmprovider.Provider) (Provider, []Model) {
	scope, teamID, userID := translateScope(old.Owner)

	presetKey := old.PresetKey
	if presetKey == "" {
		presetKey = old.Key
	}

	p := Provider{
		Name:       old.Name,
		ProviderID: inferProviderID(old),
		API:        inferAPI(old),
		BaseURL:    old.APIURL,
		APIKey:     old.APIKey,
		Enabled:    old.Enabled,
		Status:     old.Status,
		PresetKey:  presetKey,
		Scope:      scope,
		TeamID:     teamID,
		UserID:     userID,
	}
	if p.Status == "" {
		p.Status = "unconfigured"
	}

	models := make([]Model, 0, len(old.Models))
	for _, mi := range old.Models {
		models = append(models, translateModel(mi, p.API))
	}

	return p, models
}

// ---------------------------------------------------------------------------
// MigrateRoles — role binding migration
// ---------------------------------------------------------------------------

// MigrateRoles reads role assignments from all scopes (system, team, user)
// and writes them as RoleBinding records.
// keyToRef maps old provider keys to new provider auto-increment IDs.
// variantMap maps old variant model IDs to merged model IDs.
func MigrateRoles(keyToRef map[string]int, variantMap map[string]string) error {
	if setting.Global == nil {
		return nil
	}

	// System scope
	migrateRolesForScope(ScopeSystem, "", "", keyToRef, variantMap)

	// Team and user scopes: enumerate via KV store keys.
	s, err := goustore.Get("__yao.store")
	if err != nil {
		return nil // store not available — skip team/user scopes
	}

	for _, key := range s.Keys("setting:t*:llm.roles") {
		teamID := extractScopeID(key, "setting:t", ":llm.roles")
		if teamID != "" {
			migrateRolesForScope(ScopeTeam, teamID, "", keyToRef, variantMap)
		}
	}
	for _, key := range s.Keys("setting:u*:llm.roles") {
		userID := extractScopeID(key, "setting:u", ":llm.roles")
		if userID != "" {
			migrateRolesForScope(ScopeUser, "", userID, keyToRef, variantMap)
		}
	}
	return nil
}

// extractScopeID extracts the ID from a store key like "setting:t123:llm.roles".
func extractScopeID(key, prefix, suffix string) string {
	if !strings.HasPrefix(key, prefix) || !strings.HasSuffix(key, suffix) {
		return ""
	}
	return key[len(prefix) : len(key)-len(suffix)]
}

// migrateRolesForScope reads roles at a single scope and writes RoleBindings.
func migrateRolesForScope(scope Scope, teamID, userID string, keyToRef map[string]int, variantMap map[string]string) {
	sid := setting.ScopeID{Scope: setting.Scope(scope)}
	switch scope {
	case ScopeTeam:
		sid.TeamID = teamID
	case ScopeUser:
		sid.UserID = userID
	}

	data, err := setting.Global.Get(sid, llmprovider.RolesNamespace)
	if err != nil {
		return
	}

	for role, target := range data {
		rt := parseOldRoleTarget(target)
		if rt.Provider == "" {
			continue
		}

		ref, modelID := resolveOldRoleRef(rt, keyToRef)
		if ref == 0 {
			continue
		}

		// Remap variant model IDs to the merged base ID.
		if mapped, ok := variantMap[modelID]; ok {
			modelID = mapped
		}

		rb := &RoleBinding{
			Role:        role,
			ProviderRef: ref,
			ModelID:     modelID,
			Scope:       scope,
			TeamID:      teamID,
			UserID:      userID,
		}
		if _, err := SetRoleBinding(rb); err != nil {
			continue
		}
	}
}

// ---------------------------------------------------------------------------
// Provider-level inference
// ---------------------------------------------------------------------------

// inferProviderID maps an old connector to a ProviderID.
// URL-based inference takes priority (most reliable); when the URL doesn't
// match any known vendor domain, the provider Name is slugified so that
// non-standard services get meaningful IDs (e.g. "Tao Service" → "tao-service")
// instead of inheriting the generic connector type like "openai".
func inferProviderID(old *llmprovider.Provider) ProviderID {
	if pid := inferProviderFromURL(old.APIURL); pid != "" {
		return pid
	}
	return slugifyName(old.Name)
}

func inferProviderFromURL(rawURL string) ProviderID {
	u := strings.ToLower(rawURL)
	switch {
	case strings.Contains(u, "deepseek"):
		return ProviderDeepSeek
	case strings.Contains(u, "moonshot"):
		return ProviderMoonshot
	case strings.Contains(u, "anthropic"):
		return ProviderAnthropic
	case strings.Contains(u, "groq"):
		return ProviderGroq
	case strings.Contains(u, "together"):
		return ProviderTogether
	case strings.Contains(u, "openrouter"):
		return ProviderOpenRouter
	case strings.Contains(u, "googleapis"), strings.Contains(u, "generativelanguage"):
		return ProviderGoogle
	case strings.Contains(u, "mistral"):
		return ProviderMistral
	case strings.Contains(u, "azure"):
		return ProviderAzure
	case strings.Contains(u, "openai"):
		return ProviderOpenAI
	default:
		return ""
	}
}

// inferAPI maps an old connector type to the new wire protocol API.
func inferAPI(old *llmprovider.Provider) API {
	switch old.Type {
	case "anthropic":
		return APIAnthropicMessages
	default:
		return APIOpenAICompletions
	}
}

// translateScope converts old ProviderOwner to the new scope triple.
func translateScope(owner llmprovider.ProviderOwner) (Scope, string, string) {
	switch owner.Type {
	case "team":
		return ScopeTeam, owner.TeamID, ""
	case "user":
		return ScopeUser, "", owner.UserID
	default:
		return ScopeSystem, "", ""
	}
}

// ---------------------------------------------------------------------------
// Model-level translation
// ---------------------------------------------------------------------------

func translateModel(mi llmprovider.ModelInfo, api API) Model {
	capSet := buildCapSet(mi.Capabilities)

	m := Model{
		ModelID:               mi.ID,
		ModelAPI:              mi.Model,
		Name:                  mi.Name,
		Type:                  inferModelType(capSet),
		Enabled:               mi.Enabled,
		ContextWindow:         mi.MaxInputTokens,
		MaxOutputTokens:       mi.MaxOutputTokens,
		Input:                 []string{"text"},
		Capabilities:          mi.Capabilities,
		SupportsImages:        capSet["vision"],
		SupportsToolUse:       capSet["tool_calls"],
		Reasoning:             capSet["reasoning"],
		Audio:                 capSet["audio"],
		STT:                   capSet["stt"],
		Streaming:             capSet["streaming"],
		JSON:                  capSet["json"],
		Embedding:             capSet["embedding"],
		ImageGeneration:       capSet["image_generation"],
		OCR:                   capSet["ocr"],
		TemperatureAdjustable: capSet["temperature_adjustable"],
	}

	// Family from metadata
	if mi.Metadata != nil {
		if f, ok := mi.Metadata["model_family"].(string); ok {
			m.Family = f
		}
	}

	// Metadata: carry over UI display fields, excluding keys already extracted
	// into dedicated struct fields.
	m.Metadata = filterMetadata(mi.Metadata)

	if capSet["image_editing"] {
		m.ImageEditing = "multipart"
	}
	if m.SupportsImages {
		m.Input = []string{"text", "image"}
	}

	// Reasoning → ThinkingLevelMap
	m.ThinkingLevelMap = translateThinkingMap(mi)
	if m.ThinkingLevelMap != nil {
		keys := make(map[string]*string, len(m.ThinkingLevelMap))
		for k, v := range m.ThinkingLevelMap {
			keys[string(k)] = v
		}
		m.MaxThinking = inferMaxThinking(keys)
	}

	// Model-level compat — protocol-aware
	m.Compat = extractModelCompat(mi, api)

	return m
}

// metadataExtractedKeys lists metadata keys that are promoted to dedicated
// Model struct fields and must not be duplicated into Model.Metadata.
var metadataExtractedKeys = map[string]bool{
	"model_family":      true,
	"reasoning_efforts": true,
	"reasoning_effort":  true,
	"thinking_format":   true,
}

// filterMetadata returns a JSONMap with UI display fields from the old
// ModelInfo.Metadata, excluding keys already extracted into struct fields.
func filterMetadata(meta map[string]interface{}) JSONMap {
	if meta == nil {
		return nil
	}
	out := make(JSONMap, len(meta))
	for k, v := range meta {
		if metadataExtractedKeys[k] {
			continue
		}
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func buildCapSet(caps []string) map[string]bool {
	s := make(map[string]bool, len(caps))
	for _, c := range caps {
		s[c] = true
	}
	return s
}

// inferModelType derives ModelType from the capability set.
// decision → ModelTypeDecision, image_generation → ModelTypeImage, else → ModelTypeChat.
// Other capabilities (embedding, audio, stt) are captured via dedicated bool fields.
func inferModelType(capSet map[string]bool) ModelType {
	if capSet["decision"] {
		return ModelTypeDecision
	}
	if capSet["image_generation"] {
		return ModelTypeImage
	}
	return ModelTypeChat
}

// wireModelName returns the upstream model name sent on the wire.
// ModelInfo.Model is set for variant entries (e.g. "deepseek-flash"); the base
// entry leaves it empty, in which case the wire name equals the entry ID.
func wireModelName(mi llmprovider.ModelInfo) string {
	if mi.Model != "" {
		return mi.Model
	}
	return mi.ID
}

// translateThinkingMap converts old reasoning data to ThinkingLevelMap.
// Priority: Metadata["reasoning_efforts"] > Options["reasoning_effort"] > Options["thinking"]
func translateThinkingMap(mi llmprovider.ModelInfo) ThinkingLevelMap {
	// 1. Metadata["reasoning_efforts"] — explicit list
	if mi.Metadata != nil {
		if raw, ok := mi.Metadata["reasoning_efforts"]; ok {
			if efforts := parseOldEffortsList(raw); efforts != nil {
				return efforts
			}
		}
	}

	// 2. Options["reasoning_effort"] — single effort string
	if mi.Options != nil {
		if effort, ok := mi.Options["reasoning_effort"].(string); ok && effort != "" {
			return singleEffortToMap(effort)
		}
	}

	// 3. Options["thinking"] — deepseek-style
	if mi.Options != nil {
		if thinking, ok := mi.Options["thinking"].(map[string]interface{}); ok {
			if t, ok := thinking["type"].(string); ok {
				switch t {
				case "enabled", "adaptive":
					high := "high"
					return ThinkingLevelMap{ThinkingHigh: &high}
				case "disabled":
					return ThinkingLevelMap{ThinkingOff: nil}
				}
			}
		}
	}

	return nil
}

// parseOldEffortsList converts a reasoning_efforts value ([]interface{} or
// []string like ["none","low","medium","high"]) into ThinkingLevelMap.
func parseOldEffortsList(raw interface{}) ThinkingLevelMap {
	var items []string
	switch v := raw.(type) {
	case []interface{}:
		for _, item := range v {
			if s, ok := item.(string); ok {
				items = append(items, s)
			}
		}
	case []string:
		items = v
	default:
		return nil
	}
	if len(items) == 0 {
		return nil
	}

	out := make(ThinkingLevelMap, len(items))
	for _, s := range items {
		switch s {
		case "none":
			out[ThinkingOff] = nil
		case "thinking":
			high := "high"
			out[ThinkingHigh] = &high
		case "off":
			out[ThinkingOff] = nil
		case "low", "medium", "high", "xhigh", "max":
			v := s
			out[ThinkingLevel(s)] = &v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// singleEffortToMap converts a single reasoning_effort string to a one-entry ThinkingLevelMap.
func singleEffortToMap(effort string) ThinkingLevelMap {
	switch effort {
	case "none", "off":
		return ThinkingLevelMap{ThinkingOff: nil}
	case "thinking":
		high := "high"
		return ThinkingLevelMap{ThinkingHigh: &high}
	case "low", "medium", "high", "xhigh", "max":
		v := effort
		return ThinkingLevelMap{ThinkingLevel(effort): &v}
	default:
		high := "high"
		return ThinkingLevelMap{ThinkingHigh: &high}
	}
}

// extractModelCompat builds a model-level compat JSONMap from ModelInfo options.
// The api parameter determines protocol-specific fields: thinkingFormat is only
// meaningful for openai-completions; anthropic-messages uses native thinking.
func extractModelCompat(mi llmprovider.ModelInfo, api API) JSONMap {
	compat := make(JSONMap)

	// ThinkingFormat — only for non-anthropic (openai-completions) protocols.
	// anthropic-messages conveys thinking natively; thinkingFormat must be empty.
	if api != APIAnthropicMessages {
		if mi.Metadata != nil {
			if v, ok := mi.Metadata["thinking_format"].(string); ok && v != "" {
				compat["thinkingFormat"] = v
			}
		}
		if _, hasTF := compat["thinkingFormat"]; !hasTF && mi.Options != nil {
			if thinking, ok := mi.Options["thinking"].(map[string]interface{}); ok {
				if _, hasType := thinking["type"]; hasType {
					compat["thinkingFormat"] = "deepseek"
				}
			}
		}
	}

	// supportsDeveloperRole — only anthropic-messages supports the developer role.
	if api != APIAnthropicMessages {
		compat["supportsDeveloperRole"] = false
	}

	// BudgetTokens from Options["thinking"]["budget_tokens"] — protocol-independent.
	if mi.Options != nil {
		if thinking, ok := mi.Options["thinking"].(map[string]interface{}); ok {
			if bt, ok := thinking["budget_tokens"].(float64); ok && bt > 0 {
				compat["budgetTokens"] = int(bt)
			} else if bt, ok := thinking["budget_tokens"].(int); ok && bt > 0 {
				compat["budgetTokens"] = bt
			}
		}
	}

	if len(compat) == 0 {
		return nil
	}
	return compat
}

// ---------------------------------------------------------------------------
// Role target resolution
// ---------------------------------------------------------------------------

// oldRoleTarget mirrors llmprovider.RoleTarget for parsing.
type oldRoleTarget struct {
	Provider string
	Model    string
}

func parseOldRoleTarget(v interface{}) oldRoleTarget {
	switch t := v.(type) {
	case map[string]interface{}:
		provider, _ := t["provider"].(string)
		model, _ := t["model"].(string)
		return oldRoleTarget{Provider: provider, Model: model}
	default:
		return oldRoleTarget{}
	}
}

// resolveOldRoleRef maps an old role target to a new provider_ref + model_id.
// Tries keyToRef lookup first; falls back to finding by provider_id in DB.
func resolveOldRoleRef(rt oldRoleTarget, keyToRef map[string]int) (int, string) {
	// Try direct key lookup from migration mapping
	if ref, ok := keyToRef[rt.Provider]; ok {
		modelID := rt.Model
		if modelID == "" {
			modelID = extractModelFromKey(rt.Provider)
		}
		return ref, modelID
	}

	// Key might contain model suffix (e.g. "s.openai:gpt-4o")
	baseKey := rt.Provider
	modelSuffix := ""
	if idx := strings.LastIndex(rt.Provider, ":"); idx > 0 {
		baseKey = rt.Provider[:idx]
		modelSuffix = rt.Provider[idx+1:]
	}
	if ref, ok := keyToRef[baseKey]; ok {
		modelID := rt.Model
		if modelID == "" {
			modelID = modelSuffix
		}
		return ref, modelID
	}

	return 0, ""
}

func extractModelFromKey(key string) string {
	if idx := strings.LastIndex(key, ":"); idx > 0 {
		return key[idx+1:]
	}
	return ""
}

// ---------------------------------------------------------------------------
// Migration marker (uses __yao.config table)
// ---------------------------------------------------------------------------

func isMigrated() bool {
	m, err := goumodel.Get("__yao.config")
	if err != nil {
		return false
	}
	rows, err := m.Get(goumodel.QueryParam{
		Wheres: []goumodel.QueryWhere{{Column: "key", Value: migrationConfigKey}},
		Limit:  1,
	})
	return err == nil && len(rows) > 0
}

func setMigrated() {
	m, err := goumodel.Get("__yao.config")
	if err != nil {
		return
	}
	_, _ = m.Create(map[string]interface{}{
		"key":         migrationConfigKey,
		"value":       `{"version":1}`,
		"description": "LLM provider migration from KV store to DB tables",
		"category":    "llm",
		"type":        "migration",
		"readonly":    true,
	})
}

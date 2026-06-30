package setting

import (
	"fmt"
	"regexp"

	"github.com/yaoapp/yao/setting"
)

var ValidSecretKey = regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`)

// SecretsWriteContext identifies the scope and namespace for a secrets write operation.
type SecretsWriteContext struct {
	Scope     setting.ScopeID
	Namespace string
}

// SecretUpdateEntry is the per-key payload for create/update.
type SecretUpdateEntry struct {
	Value       string `json:"value"`
	Label       string `json:"label,omitempty"`
	Description string `json:"description,omitempty"`
	Required    bool   `json:"required,omitempty"`
	Multiline   bool   `json:"multiline,omitempty"`
}

// SecretsUpdate writes one or more secrets via read-modify-write on the given namespace.
// Returns the list of keys that were updated.
func SecretsUpdate(ctx SecretsWriteContext, body map[string]SecretUpdateEntry) ([]string, error) {
	reg := setting.Global
	if reg == nil {
		return nil, fmt.Errorf("setting registry not initialized")
	}

	for key := range body {
		if !ValidSecretKey.MatchString(key) {
			return nil, fmt.Errorf("invalid key %q: must match ^[A-Z][A-Z0-9_]*$", key)
		}
	}

	existing, _ := reg.Get(ctx.Scope, ctx.Namespace)
	if existing == nil {
		existing = make(map[string]interface{})
	}

	secretsRaw, _ := existing["secrets"]
	var secrets map[string]interface{}
	if m, ok := secretsRaw.(map[string]interface{}); ok {
		secrets = m
	} else {
		secrets = make(map[string]interface{})
	}

	for key, entry := range body {
		secrets[key] = map[string]interface{}{
			"value":       setting.Encrypt(entry.Value),
			"label":       entry.Label,
			"description": entry.Description,
			"required":    entry.Required,
			"multiline":   entry.Multiline,
		}
	}

	existing["secrets"] = secrets
	if _, err := reg.Set(ctx.Scope, ctx.Namespace, existing); err != nil {
		return nil, err
	}

	keys := make([]string, 0, len(body))
	for k := range body {
		keys = append(keys, k)
	}
	return keys, nil
}

// SecretDelete removes a single secret key via read-modify-write.
func SecretDelete(ctx SecretsWriteContext, key string) error {
	reg := setting.Global
	if reg == nil {
		return fmt.Errorf("setting registry not initialized")
	}

	existing, _ := reg.Get(ctx.Scope, ctx.Namespace)
	if existing == nil {
		return nil
	}

	secretsRaw, _ := existing["secrets"]
	secrets, ok := secretsRaw.(map[string]interface{})
	if !ok || secrets == nil {
		return nil
	}

	delete(secrets, key)
	existing["secrets"] = secrets

	_, err := reg.Set(ctx.Scope, ctx.Namespace, existing)
	return err
}

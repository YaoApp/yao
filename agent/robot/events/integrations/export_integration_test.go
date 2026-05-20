//go:build integration

package integrations

import robottypes "github.com/yaoapp/yao/agent/robot/types"

// ParseIntegrationsForTest exposes parseIntegrations for external integration tests.
func ParseIntegrationsForTest(intg *robottypes.Integrations) []string {
	return parseIntegrations(intg)
}

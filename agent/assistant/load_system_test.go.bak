package assistant

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveSystemConnector_NoConfig(t *testing.T) {
	saved := systemConfig
	systemConfig = nil
	defer func() { systemConfig = saved }()

	assert.Equal(t, "", resolveSystemConnector("__yao.title"))
	assert.Equal(t, "", resolveSystemConnector("__yao.keyword"))
	assert.Equal(t, "", resolveSystemConnector("__yao.querydsl"))
	assert.Equal(t, "", resolveSystemConnector("__yao.vision"))
}

func TestResolveSystemConnector_PerAgentOverride(t *testing.T) {
	saved := systemConfig
	systemConfig = &SystemConfig{
		Title: "openai.gpt-4o",
	}
	defer func() { systemConfig = saved }()

	assert.Equal(t, "openai.gpt-4o", resolveSystemConnector("__yao.title"))
	assert.Equal(t, "", resolveSystemConnector("__yao.keyword"))
	assert.Equal(t, "", resolveSystemConnector("__yao.querydsl"))
	assert.Equal(t, "", resolveSystemConnector("__yao.vision"))
}

func TestResolveSystemConnector_RoleLevelOnly(t *testing.T) {
	saved := systemConfig
	systemConfig = &SystemConfig{
		Default: "openai.gpt-4o",
		Light:   "openai.gpt-4o-mini",
	}
	defer func() { systemConfig = saved }()

	// Role-level keys don't produce per-agent overrides
	assert.Equal(t, "", resolveSystemConnector("__yao.title"))
	assert.Equal(t, "", resolveSystemConnector("__yao.keyword"))
	assert.Equal(t, "", resolveSystemConnector("__yao.querydsl"))
	assert.Equal(t, "", resolveSystemConnector("__yao.vision"))
}

func TestResolveSystemConnector_UnknownAgent(t *testing.T) {
	saved := systemConfig
	systemConfig = &SystemConfig{
		Default: "openai.gpt-4o",
		Title:   "openai.gpt-4o",
	}
	defer func() { systemConfig = saved }()

	assert.Equal(t, "", resolveSystemConnector("__yao.nonexistent"))
	assert.Equal(t, "", resolveSystemConnector("custom.agent"))
}

package main

import (
	"net/http"
	"strings"
)

type MockMode string

const (
	ModeEcho      MockMode = "echo"
	ModeToolCall  MockMode = "tool-call"
	ModeMultiTurn MockMode = "multi-turn"
	ModeError429  MockMode = "error-429"
	ModeError500  MockMode = "error-500"
	ModeSlow      MockMode = "slow"
	ModeReasoning MockMode = "reasoning"
	ModeFixture   MockMode = "fixture"
	ModeValidator MockMode = "validator"
	ModeGenerator MockMode = "generator"
)

const MockModeHeader = "X-Mock-Mode"

func parseMockMode(r *http.Request) MockMode {
	mode := MockMode(r.Header.Get(MockModeHeader))
	switch mode {
	case ModeEcho, ModeToolCall, ModeMultiTurn, ModeError429, ModeError500,
		ModeSlow, ModeReasoning, ModeFixture, ModeValidator, ModeGenerator:
		return mode
	default:
		return ModeEcho
	}
}

// detectModeFromModel infers the mock mode from the model name when no
// X-Mock-Mode header is set. This allows connectors to select behaviour
// purely via their configured model string.
func detectModeFromModel(model string) MockMode {
	m := strings.ToLower(model)
	switch {
	case strings.Contains(m, "validator"):
		return ModeValidator
	case strings.Contains(m, "generator"):
		return ModeGenerator
	case strings.Contains(m, "slow"):
		return ModeSlow
	default:
		return ModeEcho
	}
}

func fixtureKey(r *http.Request) string {
	return r.Header.Get("X-Mock-Fixture")
}

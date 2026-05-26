//go:build unit

package tai_test

import (
	"testing"

	"github.com/yaoapp/yao/tai"
	"github.com/yaoapp/yao/unit-test/agent/testprepare"
)

func TestRunnersGate_DockerOnly(t *testing.T) {
	testprepare.PrepareUnit(t)
	runners := tai.ExportLocalRunners(true, false, nil)
	assertLen(t, runners, 4)
	assertContains(t, runners, "yaocode")
	assertContains(t, runners, "claude")
	assertContains(t, runners, "opencode")
	assertContains(t, runners, "tai")
}

func TestRunnersGate_HostExecOnly_NoneDetected(t *testing.T) {
	testprepare.PrepareUnit(t)
	runners := tai.ExportLocalRunners(false, true, nil)
	assertLen(t, runners, 1)
	assertContains(t, runners, "yaocode")
}

func TestRunnersGate_HostExecOnly_ClaudeDetected(t *testing.T) {
	testprepare.PrepareUnit(t)
	detected := map[string]bool{"claude": true}
	runners := tai.ExportLocalRunners(false, true, detected)
	assertLen(t, runners, 2)
	assertContains(t, runners, "yaocode")
	assertContains(t, runners, "claude")
}

func TestRunnersGate_HostExecOnly_AllDetected(t *testing.T) {
	testprepare.PrepareUnit(t)
	detected := map[string]bool{"claude": true, "opencode": true, "tai": true}
	runners := tai.ExportLocalRunners(false, true, detected)
	assertLen(t, runners, 4)
	assertContains(t, runners, "yaocode")
	assertContains(t, runners, "claude")
	assertContains(t, runners, "opencode")
	assertContains(t, runners, "tai")
}

func TestRunnersGate_Both(t *testing.T) {
	testprepare.PrepareUnit(t)
	runners := tai.ExportLocalRunners(true, true, nil)
	assertLen(t, runners, 4)
}

func TestRunnersGate_Neither(t *testing.T) {
	testprepare.PrepareUnit(t)
	runners := tai.ExportLocalRunners(false, false, nil)
	if len(runners) != 0 {
		t.Errorf("expected empty runners, got %v", runners)
	}
}

func TestDetectedRunners_Empty(t *testing.T) {
	testprepare.PrepareUnit(t)
	m := tai.ExportDetectedRunners()
	if m == nil {
		t.Fatal("expected non-nil map")
	}
}

func assertLen(t *testing.T, list []string, expected int) {
	t.Helper()
	if len(list) != expected {
		t.Errorf("expected %d runners, got %d: %v", expected, len(list), list)
	}
}

func assertContains(t *testing.T, list []string, item string) {
	t.Helper()
	for _, v := range list {
		if v == item {
			return
		}
	}
	t.Errorf("expected %v to contain %q", list, item)
}

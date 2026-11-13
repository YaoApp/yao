package computer

import (
	"testing"
	"time"
)

func TestTimeValue(t *testing.T) {
	if timeValue(nil) != nil || timeValue("") != nil || timeValue("bad") != nil {
		t.Fatal("invalid time should be nil")
	}
	now := time.Now().UTC().Truncate(time.Second)
	if got := timeValue(now); got == nil || !got.Equal(now) {
		t.Fatalf("time.Time=%v", got)
	}
	if got := timeValue(&now); got == nil || !got.Equal(now) {
		t.Fatalf("*time.Time=%v", got)
	}
	formatted := now.Format("2006-01-02 15:04:05")
	if got := timeValue(formatted); got == nil || !got.Equal(now) {
		t.Fatalf("db time=%v want %v", got, now)
	}
	if got := timeValue(now.Format(time.RFC3339)); got == nil || !got.Equal(now) {
		t.Fatalf("rfc3339=%v", got)
	}
}

func TestStringHelpersAndClock(t *testing.T) {
	if got := stringValue(map[string]interface{}{"n": 12}, "n"); got != "12" {
		t.Fatalf("number=%q", got)
	}
	if got := firstString(map[string]interface{}{"a": nil, "b": "x"}, "a", "b"); got != "x" {
		t.Fatalf("firstString=%q", got)
	}
	if nowUnixNano() == 0 {
		t.Fatal("clock returned zero")
	}
}

func TestFallbackSeedAndFirstStringBranches(t *testing.T) {
	if fallbackSeed() == 0 {
		t.Fatal("fallback seed zero")
	}
	if got := firstString(map[string]interface{}{"a": "", "b": "value"}, "a", "b"); got != "value" {
		t.Fatalf("firstString=%q", got)
	}
	if got := firstString(map[string]interface{}{"a": nil}, "a"); got != "" {
		t.Fatalf("firstString nil=%q", got)
	}
}

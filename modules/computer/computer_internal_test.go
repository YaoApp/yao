package computer

import (
	"testing"
	"time"

	"github.com/yaoapp/yao/modules/computer/types"
)

// Internal unit tests cover unexported pure helpers without DB/network.
func TestComputerFromRow(t *testing.T) {
	row := map[string]interface{}{
		"id": "cmp-123", "node_id": "node-abcdefgh", "name": "demo",
		"provider_type": "local", "scope": "user", "visibility": "private",
		"status": "pending", "__yao_created_by": "", "__yao_user_id": "u2",
		"__yao_team_id": "t2", "metadata": map[string]interface{}{"data_dir": "/tmp/d"},
		"created_at": "2026-11-12T00:00:00Z",
	}
	got := computerFromRow(row)
	if got.ID != "cmp-123" || got.UserID != "u2" || got.TeamID != "t2" || got.DataDir != "/tmp/d" {
		t.Fatalf("computerFromRow: %+v", got)
	}
	if got.CreatedAt.IsZero() || got.CreatedAt.Location() != time.UTC {
		t.Fatalf("created_at not parsed: %+v", got.CreatedAt)
	}
	if c := computerFromRow(map[string]interface{}{"metadata": "invalid", "created_at": "invalid"}); c.DataDir != "" || !c.CreatedAt.IsZero() {
		t.Fatalf("invalid row mapping: %+v", c)
	}
}

func TestStringValueAndFirstString(t *testing.T) {
	if got := stringValue(map[string]interface{}{"s": "value"}, "s"); got != "value" {
		t.Fatalf("stringValue string=%q", got)
	}
	if got := stringValue(map[string]interface{}{"n": nil}, "n"); got != "<nil>" {
		t.Fatalf("stringValue nil=%q", got)
	}
	if got := firstString(map[string]interface{}{"a": "", "b": "value"}, "a", "b"); got != "value" {
		t.Fatalf("firstString fallback=%q", got)
	}
	if got := firstString(map[string]interface{}{}); got != "" {
		t.Fatalf("firstString empty=%q", got)
	}
}

func TestModelQueryByAndClock(t *testing.T) {
	param := modelQueryBy("node_id", "node-abcdefgh")
	if len(param.Wheres) != 1 || param.Wheres[0].Column != "node_id" || param.Wheres[0].Value != "node-abcdefgh" {
		t.Fatalf("query: %+v", param)
	}
	if nowUnixNano() == 0 {
		t.Fatal("clock returned zero")
	}
}

func TestVisibilityConstants(t *testing.T) {
	for _, value := range []string{types.VisibilityPrivate, types.VisibilityMembers, types.VisibilityTeam, types.VisibilityGlobal} {
		if !validVisibility(value) {
			t.Fatalf("visibility %s invalid", value)
		}
	}
	if validVisibility("unknown") {
		t.Fatal("unknown visibility accepted")
	}
}

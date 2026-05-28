package robot

import (
	"encoding/json"
	"testing"
)

// ==================== Permission helper tests ====================

func TestGetEffectiveTeamID(t *testing.T) {
	tests := []struct {
		name string
		auth *AuthInfo
		want string
	}{
		{"nil auth", nil, ""},
		{"team present", &AuthInfo{TeamID: "team-1", UserID: "user-1"}, "team-1"},
		{"no team, user fallback", &AuthInfo{UserID: "user-1"}, "user-1"},
		{"empty both", &AuthInfo{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getEffectiveTeamID(tt.auth)
			if got != tt.want {
				t.Errorf("getEffectiveTeamID() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestCanRead(t *testing.T) {
	tests := []struct {
		name           string
		auth           *AuthInfo
		robotTeamID    string
		robotCreatedBy string
		want           bool
	}{
		{"nil auth", nil, "t1", "u1", false},
		{"no constraints (admin)", &AuthInfo{UserID: "u1"}, "t1", "u2", true},
		{"creator always reads", &AuthInfo{UserID: "u1", TeamOnly: true}, "t1", "u1", true},
		{"same team reads", &AuthInfo{UserID: "u2", TeamID: "t1", TeamOnly: true}, "t1", "u1", true},
		{"different team blocked", &AuthInfo{UserID: "u2", TeamID: "t2", TeamOnly: true}, "t1", "u1", false},
		{"owner only - not creator", &AuthInfo{UserID: "u2", OwnerOnly: true}, "t1", "u1", false},
		{"owner only - is creator", &AuthInfo{UserID: "u1", OwnerOnly: true}, "t1", "u1", true},
		{"team only, empty robot team, not creator", &AuthInfo{UserID: "u2", TeamID: "t1", TeamOnly: true}, "", "u1", false},
		{"team only, empty created_by, same team", &AuthInfo{UserID: "u2", TeamID: "t1", TeamOnly: true}, "t1", "", true},
		{"team only, empty created_by, diff team", &AuthInfo{UserID: "u2", TeamID: "t2", TeamOnly: true}, "t1", "", false},
		{"owner only, empty created_by", &AuthInfo{UserID: "u1", OwnerOnly: true}, "t1", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canRead(tt.auth, tt.robotTeamID, tt.robotCreatedBy)
			if got != tt.want {
				t.Errorf("canRead() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCanWrite(t *testing.T) {
	tests := []struct {
		name           string
		auth           *AuthInfo
		robotTeamID    string
		robotCreatedBy string
		want           bool
	}{
		{"nil auth", nil, "t1", "u1", false},
		{"no constraints (admin)", &AuthInfo{UserID: "u1"}, "t1", "u2", true},
		{"creator writes (owner only)", &AuthInfo{UserID: "u1", OwnerOnly: true}, "t1", "u1", true},
		{"non-creator blocked (owner only)", &AuthInfo{UserID: "u2", OwnerOnly: true}, "t1", "u1", false},
		{"team only - creator same team", &AuthInfo{UserID: "u1", TeamID: "t1", TeamOnly: true}, "t1", "u1", true},
		{"team only - creator different team", &AuthInfo{UserID: "u1", TeamID: "t2", TeamOnly: true}, "t1", "u1", false},
		{"team only - creator, robot has no team", &AuthInfo{UserID: "u1", TeamID: "t1", TeamOnly: true}, "", "u1", true},
		{"team only - not creator", &AuthInfo{UserID: "u2", TeamID: "t1", TeamOnly: true}, "t1", "u1", false},
		{"owner only, empty created_by", &AuthInfo{UserID: "u1", OwnerOnly: true}, "t1", "", false},
		{"no constraints, empty both robot fields", &AuthInfo{UserID: "u1"}, "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := canWrite(tt.auth, tt.robotTeamID, tt.robotCreatedBy)
			if got != tt.want {
				t.Errorf("canWrite() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildListFilter(t *testing.T) {
	tests := []struct {
		name            string
		auth            *AuthInfo
		requestedTeamID string
		want            string
	}{
		{"nil auth", nil, "req-team", "req-team"},
		{"nil auth, empty request", nil, "", ""},
		{"no constraints, requested", &AuthInfo{UserID: "u1"}, "req-team", "req-team"},
		{"no constraints, empty fallback to team", &AuthInfo{UserID: "u1", TeamID: "t1"}, "", "t1"},
		{"no constraints, empty fallback to user", &AuthInfo{UserID: "u1"}, "", "u1"},
		{"team only override", &AuthInfo{UserID: "u1", TeamID: "t1", TeamOnly: true}, "req-team", "t1"},
		{"team only, empty team ID", &AuthInfo{UserID: "u1", TeamOnly: true}, "req-team", "req-team"},
		{"owner only override", &AuthInfo{UserID: "u1", OwnerOnly: true}, "req-team", "u1"},
		{"owner only, empty user", &AuthInfo{OwnerOnly: true}, "req-team", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildListFilter(tt.auth, tt.requestedTeamID)
			if got != tt.want {
				t.Errorf("buildListFilter() = %q, want %q", got, tt.want)
			}
		})
	}
}

// ==================== Whitelist struct tests (security core) ====================

func TestCreateRequestWhitelist(t *testing.T) {
	input := `{
		"display_name": "Good Bot",
		"bio": "Does good things",
		"system_prompt": "Be good",
		"agents": ["agent-1"],
		"workspace": "ws-1",
		"autonomous_mode": true,
		"mcp_servers": ["evil-server"],
		"cost_limit": 99999,
		"language_model": "gpt-4",
		"status": "active",
		"robot_status": "working",
		"role_id": "admin",
		"manager_id": "hacker",
		"robot_email": "evil@evil.com",
		"authorized_senders": ["*"],
		"team_id": "stolen-team"
	}`

	var req CreateRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.DisplayName != "Good Bot" {
		t.Errorf("DisplayName = %q, want %q", req.DisplayName, "Good Bot")
	}
	if req.Bio != "Does good things" {
		t.Errorf("Bio = %q, want %q", req.Bio, "Does good things")
	}
	if req.SystemPrompt != "Be good" {
		t.Errorf("SystemPrompt = %q, want %q", req.SystemPrompt, "Be good")
	}
	if len(req.Agents) != 1 || req.Agents[0] != "agent-1" {
		t.Errorf("Agents = %v, want [agent-1]", req.Agents)
	}
	if req.Workspace != "ws-1" {
		t.Errorf("Workspace = %q, want %q", req.Workspace, "ws-1")
	}
	if req.AutonomousMode == nil || !*req.AutonomousMode {
		t.Error("AutonomousMode should be true")
	}

	buf, _ := json.Marshal(req)
	s := string(buf)

	for _, forbidden := range []string{
		"mcp_servers", "cost_limit", "language_model",
		"status", "robot_status", "role_id", "manager_id",
		"robot_email", "authorized_senders", "team_id",
		"evil-server", "99999", "gpt-4", "hacker", "evil@evil.com", "stolen-team",
	} {
		if contains(s, forbidden) {
			t.Errorf("SECURITY: forbidden field %q leaked through whitelist", forbidden)
		}
	}
}

func TestUpdateRequestWhitelist(t *testing.T) {
	input := `{
		"display_name": "Updated Name",
		"bio": "Updated Bio",
		"mcp_servers": ["evil-server"],
		"cost_limit": 99999,
		"language_model": "gpt-4",
		"status": "active",
		"robot_status": "working",
		"robot_email": "evil@evil.com"
	}`

	var req UpdateRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.DisplayName == nil || *req.DisplayName != "Updated Name" {
		t.Errorf("DisplayName = %v, want 'Updated Name'", req.DisplayName)
	}
	if req.Bio == nil || *req.Bio != "Updated Bio" {
		t.Errorf("Bio = %v, want 'Updated Bio'", req.Bio)
	}

	buf, _ := json.Marshal(req)
	s := string(buf)

	for _, forbidden := range []string{
		"mcp_servers", "cost_limit", "language_model",
		"status", "robot_status", "robot_email",
	} {
		if contains(s, forbidden) {
			t.Errorf("SECURITY: forbidden field %q leaked through whitelist", forbidden)
		}
	}
}

func TestUpdateRequestNilVsOmit(t *testing.T) {
	t.Run("omitted fields are nil", func(t *testing.T) {
		input := `{"display_name": "Only Name"}`
		var req UpdateRequest
		if err := json.Unmarshal([]byte(input), &req); err != nil {
			t.Fatal(err)
		}
		if req.DisplayName == nil {
			t.Error("DisplayName should not be nil")
		}
		if req.Bio != nil {
			t.Error("Bio should be nil (not provided)")
		}
		if req.SystemPrompt != nil {
			t.Error("SystemPrompt should be nil (not provided)")
		}
		if req.AutonomousMode != nil {
			t.Error("AutonomousMode should be nil (not provided)")
		}
		if req.RobotConfig != nil {
			t.Error("RobotConfig should be nil (not provided)")
		}
	})

	t.Run("empty string is not nil", func(t *testing.T) {
		input := `{"workspace": ""}`
		var req UpdateRequest
		if err := json.Unmarshal([]byte(input), &req); err != nil {
			t.Fatal(err)
		}
		if req.Workspace == nil {
			t.Error("Workspace should not be nil (explicitly empty)")
		}
		if *req.Workspace != "" {
			t.Errorf("Workspace = %q, want empty string", *req.Workspace)
		}
	})
}

func TestTriggerRequestWhitelist(t *testing.T) {
	input := `{
		"type": "human",
		"messages": [{"role": "user", "content": "hello"}],
		"action": "task.add",
		"executor_mode": "sandbox",
		"plan_at": "2025-01-01T00:00:00Z",
		"locale": "en"
	}`

	var req TriggerRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.Type != "human" {
		t.Errorf("Type = %q, want %q", req.Type, "human")
	}
	if len(req.Messages) != 1 {
		t.Fatalf("Messages len = %d, want 1", len(req.Messages))
	}
	if req.Messages[0].Role != "user" || req.Messages[0].Content != "hello" {
		t.Errorf("Messages[0] = %+v, unexpected", req.Messages[0])
	}

	buf, _ := json.Marshal(req)
	s := string(buf)

	for _, forbidden := range []string{
		"action", "executor_mode", "plan_at", "locale",
		"task.add", "sandbox",
	} {
		if contains(s, forbidden) {
			t.Errorf("SECURITY: forbidden field %q leaked through whitelist", forbidden)
		}
	}
}

func TestTriggerRequestEventType(t *testing.T) {
	input := `{
		"type": "event",
		"source": "webhook",
		"event_type": "lead.created",
		"data": {"lead_id": "123", "name": "Test Lead"}
	}`

	var req TriggerRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	if req.Type != "event" {
		t.Errorf("Type = %q, want %q", req.Type, "event")
	}
	if req.Source != "webhook" {
		t.Errorf("Source = %q, want %q", req.Source, "webhook")
	}
	if req.EventType != "lead.created" {
		t.Errorf("EventType = %q, want %q", req.EventType, "lead.created")
	}
	if req.Data["lead_id"] != "123" {
		t.Errorf("Data[lead_id] = %v, want '123'", req.Data["lead_id"])
	}
}

// ==================== robot_config whitelist tests ====================

func TestToolRobotConfigWhitelist(t *testing.T) {
	input := `{
		"identity": {"role": "analyst", "duties": ["analyze data"], "rules": ["be accurate"]},
		"quota": {"max": 5, "queue": 20, "priority": 8},
		"clock": {"mode": "times", "times": ["09:00"], "days": ["Mon"], "tz": "Asia/Shanghai"},
		"triggers": {"clock": {"enabled": true}, "event": {"enabled": false}},
		"executor": {"mode": "sandbox", "max_duration": "30m"},
		"default_locale": "zh",
		"integrations": {"telegram": {"enabled": true, "bot_token": "STOLEN"}},
		"kb": {"collections": ["secret-kb"]},
		"db": {"models": ["admin_table"]},
		"learn": {"on": true},
		"resources": {"agents": ["internal-agent"]},
		"delivery": {"email": {"enabled": true}},
		"events": [{"type": "webhook", "source": "/admin"}]
	}`

	var cfg ToolRobotConfig
	if err := json.Unmarshal([]byte(input), &cfg); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Whitelisted fields should survive
	if cfg.Identity == nil || cfg.Identity.Role != "analyst" {
		t.Errorf("Identity.Role = %v, want 'analyst'", cfg.Identity)
	}
	if len(cfg.Identity.Duties) != 1 || cfg.Identity.Duties[0] != "analyze data" {
		t.Errorf("Identity.Duties = %v, want [analyze data]", cfg.Identity.Duties)
	}
	if len(cfg.Identity.Rules) != 1 || cfg.Identity.Rules[0] != "be accurate" {
		t.Errorf("Identity.Rules = %v, want [be accurate]", cfg.Identity.Rules)
	}
	if cfg.Quota == nil || cfg.Quota.Max != 5 || cfg.Quota.Queue != 20 || cfg.Quota.Priority != 8 {
		t.Errorf("Quota = %+v, want {5, 20, 8}", cfg.Quota)
	}
	if cfg.Clock == nil || cfg.Clock.Mode != "times" {
		t.Errorf("Clock.Mode = %v, want 'times'", cfg.Clock)
	}
	if cfg.Clock.TZ != "Asia/Shanghai" {
		t.Errorf("Clock.TZ = %q, want 'Asia/Shanghai'", cfg.Clock.TZ)
	}
	if cfg.Triggers == nil || cfg.Triggers.Clock == nil || !cfg.Triggers.Clock.Enabled {
		t.Error("Triggers.Clock.Enabled should be true")
	}
	if cfg.Triggers.Event == nil || cfg.Triggers.Event.Enabled {
		t.Error("Triggers.Event.Enabled should be false")
	}
	if cfg.Executor == nil || cfg.Executor.Mode != "sandbox" {
		t.Errorf("Executor.Mode = %v, want 'sandbox'", cfg.Executor)
	}
	if cfg.DefaultLocale != "zh" {
		t.Errorf("DefaultLocale = %q, want 'zh'", cfg.DefaultLocale)
	}

	// Dangerous fields MUST be dropped
	buf, _ := json.Marshal(cfg)
	s := string(buf)

	for _, forbidden := range []string{
		"integrations", "telegram", "bot_token", "STOLEN",
		"secret-kb", "collections",
		"admin_table", `"models"`,
		`"learn"`, `"on":true`,
		"internal-agent", `"resources"`,
		`"delivery"`,
		`"events"`, "/admin",
	} {
		if contains(s, forbidden) {
			t.Errorf("SECURITY: dangerous field %q leaked through robot_config whitelist", forbidden)
		}
	}
}

func TestToolRobotConfigPartialFields(t *testing.T) {
	t.Run("only identity", func(t *testing.T) {
		input := `{"identity": {"role": "writer"}}`
		var cfg ToolRobotConfig
		if err := json.Unmarshal([]byte(input), &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Identity == nil || cfg.Identity.Role != "writer" {
			t.Error("Identity.Role should be 'writer'")
		}
		if cfg.Quota != nil {
			t.Error("Quota should be nil")
		}
		if cfg.Clock != nil {
			t.Error("Clock should be nil")
		}
	})

	t.Run("only quota", func(t *testing.T) {
		input := `{"quota": {"max": 3}}`
		var cfg ToolRobotConfig
		if err := json.Unmarshal([]byte(input), &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Identity != nil {
			t.Error("Identity should be nil")
		}
		if cfg.Quota == nil || cfg.Quota.Max != 3 {
			t.Error("Quota.Max should be 3")
		}
		if cfg.Quota.Queue != 0 {
			t.Errorf("Quota.Queue = %d, want 0 (zero value)", cfg.Quota.Queue)
		}
	})

	t.Run("empty object", func(t *testing.T) {
		input := `{}`
		var cfg ToolRobotConfig
		if err := json.Unmarshal([]byte(input), &cfg); err != nil {
			t.Fatal(err)
		}
		if cfg.Identity != nil || cfg.Quota != nil || cfg.Clock != nil {
			t.Error("All sub-configs should be nil for empty input")
		}
	})
}

// ==================== End-to-end unmarshal simulation ====================

func TestCreateRequestWithRobotConfig(t *testing.T) {
	input := `{
		"display_name": "Sales Bot",
		"robot_config": {
			"identity": {"role": "sales", "duties": ["follow up leads"]},
			"quota": {"max": 3},
			"integrations": {"telegram": {"bot_token": "SECRET"}}
		}
	}`

	var req CreateRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatal(err)
	}

	if req.DisplayName != "Sales Bot" {
		t.Errorf("DisplayName = %q", req.DisplayName)
	}
	if req.RobotConfig == nil {
		t.Fatal("RobotConfig should not be nil")
	}
	if req.RobotConfig.Identity == nil || req.RobotConfig.Identity.Role != "sales" {
		t.Error("RobotConfig.Identity.Role should be 'sales'")
	}
	if req.RobotConfig.Quota == nil || req.RobotConfig.Quota.Max != 3 {
		t.Error("RobotConfig.Quota.Max should be 3")
	}

	buf, _ := json.Marshal(req)
	s := string(buf)
	if contains(s, "telegram") || contains(s, "SECRET") || contains(s, "integrations") {
		t.Error("SECURITY: integrations leaked through nested robot_config")
	}
}

func TestUpdateRequestWithRobotConfig(t *testing.T) {
	input := `{
		"robot_config": {
			"clock": {"mode": "interval", "every": "1h"},
			"kb": {"collections": ["stolen-data"]}
		}
	}`

	var req UpdateRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		t.Fatal(err)
	}

	if req.RobotConfig == nil {
		t.Fatal("RobotConfig should not be nil")
	}
	if req.RobotConfig.Clock == nil || req.RobotConfig.Clock.Mode != "interval" {
		t.Error("Clock.Mode should be 'interval'")
	}
	if req.RobotConfig.Clock.Every != "1h" {
		t.Errorf("Clock.Every = %q, want '1h'", req.RobotConfig.Clock.Every)
	}

	buf, _ := json.Marshal(req)
	s := string(buf)
	if contains(s, "kb") || contains(s, "stolen-data") {
		t.Error("SECURITY: kb leaked through nested robot_config in update")
	}
}

// ==================== Response type JSON serialization ====================

func TestRobotSummaryJSON(t *testing.T) {
	s := RobotSummary{
		MemberID:       "rob-1",
		DisplayName:    "Test Bot",
		Bio:            "A test",
		Status:         "idle",
		AutonomousMode: true,
		Running:        2,
	}
	buf, err := json.Marshal(s)
	if err != nil {
		t.Fatal(err)
	}

	var m map[string]interface{}
	json.Unmarshal(buf, &m)

	if m["member_id"] != "rob-1" {
		t.Errorf("member_id = %v", m["member_id"])
	}
	if m["autonomous_mode"] != true {
		t.Errorf("autonomous_mode = %v", m["autonomous_mode"])
	}
	if m["running"] != float64(2) {
		t.Errorf("running = %v", m["running"])
	}
}

func TestRobotResponseHidesInternalFields(t *testing.T) {
	r := RobotResponse{
		Data:         map[string]string{"member_id": "rob-1"},
		YaoTeamID:    "secret-team",
		YaoCreatedBy: "secret-user",
	}
	buf, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	s := string(buf)

	if contains(s, "secret-team") || contains(s, "secret-user") {
		t.Error("YaoTeamID/YaoCreatedBy should not appear in JSON (json:\"-\")")
	}
	if !contains(s, "rob-1") {
		t.Error("Data should be serialized")
	}
}

func TestRobotStateHidesInternalFields(t *testing.T) {
	s := RobotState{
		MemberID:     "rob-1",
		Status:       "idle",
		Running:      0,
		MaxRunning:   2,
		RunningIDs:   []string{"exec-1"},
		YaoTeamID:    "internal-team",
		YaoCreatedBy: "internal-user",
	}
	buf, _ := json.Marshal(s)
	str := string(buf)

	if contains(str, "internal-team") || contains(str, "internal-user") {
		t.Error("internal fields should not appear in JSON")
	}
	if !contains(str, "exec-1") {
		t.Error("running_ids should be serialized")
	}
}

// ==================== helpers ====================

func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

package setting

// SystemInfoData is the top-level response for GET /setting/system.
type SystemInfoData struct {
	App              AppInfo       `json:"app"`
	Deployment       string        `json:"deployment"`
	DeploymentLabel  string        `json:"deployment_label"`
	LicenseKey       string        `json:"license_key,omitempty"`
	Server           VersionInfo   `json:"server"`
	Client           VersionInfo   `json:"client"`
	Environment      string        `json:"environment"`
	EnvironmentLabel string        `json:"environment_label"`
	Technical        TechnicalInfo `json:"technical"`
	Promotions       []Promotion   `json:"promotions,omitempty"`
}

// Promotion is a localized CTA banner returned by the API.
type Promotion struct {
	ID    string `json:"id"`
	Title string `json:"title"`
	Desc  string `json:"desc"`
	Link  string `json:"link"`
	Label string `json:"label"`
}

// AppInfo describes the running application.
type AppInfo struct {
	Name        string `json:"name"`
	Short       string `json:"short"`
	Description string `json:"description"`
	Logo        string `json:"logo"`
	Version     string `json:"version"`
}

// VersionInfo carries build metadata for a component (engine / CUI).
type VersionInfo struct {
	Version   string `json:"version"`
	BuildDate string `json:"build_date"`
	CommitSHA string `json:"commit"`
}

// TechnicalInfo contains runtime / infrastructure details.
type TechnicalInfo struct {
	Listen       string `json:"listen"`
	DBDriver     string `json:"db_driver"`
	SessionStore string `json:"session_store"`
}

// CheckUpdateResult is the response for POST /setting/system/check-update.
type CheckUpdateResult struct {
	HasUpdate      bool   `json:"has_update"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version,omitempty"`
	DownloadURL    string `json:"download_url,omitempty"`
}

// ---------------------------------------------------------------------------
// Cloud Service
// ---------------------------------------------------------------------------

// CloudRegion is a static entry loaded from cloud_presets.yml.
type CloudRegion struct {
	Key     string            `json:"key"     yaml:"key"`
	Label   map[string]string `json:"label"   yaml:"label"`
	APIURL  string            `json:"api_url" yaml:"api_url"`
	Default bool              `json:"default,omitempty" yaml:"default"`
}

// CloudPageData is the response for GET /setting/cloud.
type CloudPageData struct {
	Regions []CloudRegion `json:"regions"`
	Region  string        `json:"region"`
	APIURL  string        `json:"api_url"`
	APIKey  string        `json:"api_key"`
	Status  string        `json:"status"`
}

// CloudTestResult is the response for POST /setting/cloud/test.
type CloudTestResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// ---------------------------------------------------------------------------
// LLM Providers
// ---------------------------------------------------------------------------

// LLMPageData is the aggregated response for GET /setting/llm.
type LLMPageData struct {
	Providers       []interface{}          `json:"providers"`
	Roles           map[string]interface{} `json:"roles"`
	PresetProviders []interface{}          `json:"preset_providers"`
}

// ---------------------------------------------------------------------------
// Search & Scrape
// ---------------------------------------------------------------------------

type SearchProviderField struct {
	Key         string            `json:"key"         yaml:"key"`
	Label       map[string]string `json:"label"       yaml:"label"`
	Type        string            `json:"type"        yaml:"type"`
	Default     string            `json:"default,omitempty"     yaml:"default"`
	Placeholder string            `json:"placeholder,omitempty" yaml:"placeholder"`
	Hint        map[string]string `json:"hint,omitempty"        yaml:"hint"`
}

type SearchProviderPreset struct {
	Key         string                `json:"key"                    yaml:"key"`
	Name        string                `json:"name"                   yaml:"name"`
	Description map[string]string     `json:"description,omitempty"  yaml:"description"`
	Website     string                `json:"website,omitempty"      yaml:"website"`
	Tools       []string              `json:"tools"                  yaml:"tools"`
	ToolLabels  []map[string]string   `json:"tool_labels"            yaml:"tool_labels"`
	Fields      []SearchProviderField `json:"fields"                 yaml:"fields"`
	IsCloud     bool                  `json:"is_cloud,omitempty"     yaml:"is_cloud"`
}

type SearchProviderConfig struct {
	PresetKey   string            `json:"preset_key"`
	Enabled     bool              `json:"enabled"`
	FieldValues map[string]string `json:"field_values"`
	Status      string            `json:"status"`
}

type SearchToolAssignment struct {
	WebSearch *string `json:"web_search"`
	WebScrape *string `json:"web_scrape"`
}

type SearchPageData struct {
	Presets        []SearchProviderPreset `json:"presets"`
	Providers      []SearchProviderConfig `json:"providers"`
	ToolAssignment SearchToolAssignment   `json:"tool_assignment"`
}

type SearchTestResult struct {
	Success   bool   `json:"success"`
	Message   string `json:"message"`
	LatencyMs int64  `json:"latency_ms,omitempty"`
}

// ---------------------------------------------------------------------------
// SMTP
// ---------------------------------------------------------------------------

type SmtpPreset struct {
	Key        string            `json:"key"        yaml:"key"`
	Name       string            `json:"name"       yaml:"name"`
	Host       string            `json:"host"       yaml:"host"`
	Port       int               `json:"port"       yaml:"port"`
	Encryption string            `json:"encryption" yaml:"encryption"`
	Hint       map[string]string `json:"hint,omitempty" yaml:"hint"`
	URL        string            `json:"url,omitempty"  yaml:"url"`
	Default    bool              `json:"default,omitempty" yaml:"default"`
}

type SmtpConfig struct {
	Enabled    bool   `json:"enabled"`
	PresetKey  string `json:"preset_key"`
	Host       string `json:"host"`
	Port       int    `json:"port"`
	Encryption string `json:"encryption"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromName   string `json:"from_name"`
	FromEmail  string `json:"from_email"`
	Status     string `json:"status"`
	LastSentAt string `json:"last_sent_at,omitempty"`
}

type SmtpPageData struct {
	Presets []SmtpPreset `json:"presets"`
	Config  SmtpConfig   `json:"config"`
}

type SmtpTestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// Sandbox
// ---------------------------------------------------------------------------

type ComputerNode struct {
	NodeID           string `json:"node_id"`
	DisplayName      string `json:"display_name"`
	Kind             string `json:"kind"`
	OS               string `json:"os"`
	Arch             string `json:"arch"`
	CPU              int    `json:"cpu"`
	MemoryGB         int    `json:"memory_gb"`
	DockerVersion    string `json:"docker_version,omitempty"`
	RunningSandboxes int    `json:"running_sandboxes"`
	Online           bool   `json:"online"`
}

type SandboxRegistryConfig struct {
	RegistryURL string `json:"registry_url"`
	Username    string `json:"username"`
	Password    string `json:"password"`
}

type SandboxImage struct {
	ID             string   `json:"id"`
	AssistantNames []string `json:"assistant_names"`
	ImageName      string   `json:"image_name"`
	Tag            string   `json:"tag"`
	SizeMB         int      `json:"size_mb"`
	Status         string   `json:"status"`
	Progress       *int     `json:"progress,omitempty"`
	ErrorMessage   string   `json:"error_message,omitempty"`
}

type SandboxPageData struct {
	Nodes    []ComputerNode            `json:"nodes"`
	Registry SandboxRegistryConfig     `json:"registry"`
	Images   map[string][]SandboxImage `json:"images"`
}

// ---------------------------------------------------------------------------
// Setup Status
// ---------------------------------------------------------------------------

type Checkpoint struct {
	Status   string `json:"status"`
	Required bool   `json:"required"`
	Label    string `json:"label"`
	Path     string `json:"path"`
	Detail   string `json:"detail,omitempty"`
}

type SetupStatus struct {
	Completed           bool                  `json:"completed"`
	Checkpoints         map[string]Checkpoint `json:"checkpoints"`
	OnboardingCompleted bool                  `json:"onboarding_completed"`
	BannerDismissed     bool                  `json:"banner_dismissed"`
}

type AssistantSetupStatus struct {
	AssistantID   string                `json:"assistant_id"`
	AssistantName string                `json:"assistant_name"`
	Ready         bool                  `json:"ready"`
	Checkpoints   map[string]Checkpoint `json:"checkpoints"`
}

// ---------------------------------------------------------------------------
// User Preference
// ---------------------------------------------------------------------------

type PreferenceData struct {
	EmailNotification   *bool `json:"email_notification,omitempty"`
	BannerDismissed     *bool `json:"banner_dismissed,omitempty"`
	OnboardingCompleted *bool `json:"onboarding_completed,omitempty"`
}

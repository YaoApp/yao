package setting

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/commercial"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/share"
	"gopkg.in/yaml.v3"
)

const cdnBase = "https://get.yaoapps.com/yao"

// update check cache (package-level, protected by mutex)
var (
	updateCache     *CheckUpdateResult
	updateCacheTime time.Time
	updateMu        sync.Mutex
	cacheTTL        = 10 * time.Minute
)

// handleSystemInfo returns aggregated system information.
// GET /setting/system?locale=zh-cn
func handleSystemInfo(c *gin.Context) {
	locale := strings.ToLower(c.DefaultQuery("locale", "en-us"))

	env := share.App.Option["env"]
	environment, _ := env.(string)
	if environment == "" {
		environment = config.Conf.Mode
	}
	if environment == "" {
		environment = "production"
	}

	listen := fmt.Sprintf("%s:%d", config.Conf.Host, config.Conf.Port)
	sessionStore := config.Conf.Session.Store
	if sessionStore == "" {
		sessionStore = "file"
	}

	lang := langFromLocale(locale)
	lic := commercial.License
	deployment := lic.Edition
	if deployment == "" {
		deployment = "community"
	}

	var licenseKey string
	if lic.Valid && lic.SerialNumber != "" {
		licenseKey = lic.SerialNumber
	}

	data := SystemInfoData{
		App: AppInfo{
			Name:        share.App.Name,
			Short:       share.App.Short,
			Description: share.App.Description,
			Logo:        "/api/__yao/app/icons/app.png",
			Version:     share.App.Version,
		},
		Deployment:       deployment,
		DeploymentLabel:  resolveLabel(promFile.Labels.Deployment, deployment, lang, deployment),
		LicenseKey:       licenseKey,
		Environment:      environment,
		EnvironmentLabel: resolveLabel(promFile.Labels.Environment, environment, lang, environment),
		Server: VersionInfo{
			Version:   share.VERSION,
			BuildDate: share.PRVERSION,
			CommitSHA: share.PRVERSION,
		},
		Client: VersionInfo{
			Version:   share.CUI,
			BuildDate: share.PRCUI,
			CommitSHA: share.PRCUI,
		},
		Technical: TechnicalInfo{
			Listen:       listen,
			DBDriver:     config.Conf.DB.Driver,
			SessionStore: sessionStore,
		},
		Promotions: buildPromotions(deployment, locale),
	}

	response.RespondWithSuccess(c, http.StatusOK, data)
}

//go:embed promotions.yml
var promotionsYML []byte

type promotionEntry struct {
	ID   string                     `yaml:"id"`
	Link map[string]string          `yaml:"link"`
	I18n map[string]promotionLocale `yaml:"i18n"`
}

type promotionLocale struct {
	Title string `yaml:"title"`
	Desc  string `yaml:"desc"`
	Label string `yaml:"label"`
}

type promotionsFile struct {
	Labels struct {
		Deployment  map[string]map[string]string `yaml:"deployment"`
		Environment map[string]map[string]string `yaml:"environment"`
	} `yaml:"labels"`
	Community  []promotionEntry `yaml:"community"`
	Enterprise []promotionEntry `yaml:"enterprise"`
	Cloud      []promotionEntry `yaml:"cloud"`
}

var promFile promotionsFile

func init() {
	yaml.Unmarshal(promotionsYML, &promFile)
}

func resolveLabel(m map[string]map[string]string, key, lang, fallback string) string {
	if langs, ok := m[key]; ok {
		if v, ok := langs[lang]; ok {
			return v
		}
		if v, ok := langs["en"]; ok {
			return v
		}
	}
	return fallback
}

func langFromLocale(locale string) string {
	if strings.HasPrefix(locale, "zh") {
		return "zh"
	}
	return "en"
}

func buildPromotions(deployment, locale string) []Promotion {
	lang := langFromLocale(locale)

	var entries []promotionEntry
	switch deployment {
	case "community":
		entries = promFile.Community
	case "enterprise":
		entries = promFile.Enterprise
	case "cloud":
		entries = promFile.Cloud
	}
	if len(entries) == 0 {
		return nil
	}

	promos := make([]Promotion, 0, len(entries))
	for _, e := range entries {
		loc, ok := e.I18n[lang]
		if !ok {
			loc = e.I18n["en"]
		}
		link := e.Link[lang]
		if link == "" {
			link = e.Link["en"]
		}
		promos = append(promos, Promotion{
			ID:    e.ID,
			Title: loc.Title,
			Desc:  loc.Desc,
			Link:  link,
			Label: loc.Label,
		})
	}
	return promos
}

// handleSystemCheckUpdate checks for a newer engine release.
// Uses the same CDN source as `yao upgrade` and yao-desktop:
//
//	GET https://get.yaoapps.com/yao/latest.json
//
// POST /setting/system/check-update
func handleSystemCheckUpdate(c *gin.Context) {
	updateMu.Lock()
	if updateCache != nil && time.Since(updateCacheTime) < cacheTTL {
		result := *updateCache
		updateMu.Unlock()
		response.RespondWithSuccess(c, http.StatusOK, result)
		return
	}
	updateMu.Unlock()

	result := fetchLatestVersion()

	updateMu.Lock()
	updateCache = &result
	updateCacheTime = time.Now()
	updateMu.Unlock()

	response.RespondWithSuccess(c, http.StatusOK, result)
}

// cdnLatest mirrors the JSON structure of get.yaoapps.com/yao/latest.json
// (same format used by cmd/upgrade.go and yao-desktop updater.rs).
type cdnLatest struct {
	Version    string            `json:"version"`
	ReleasedAt string            `json:"released_at"`
	Assets     map[string]string `json:"assets"`
}

func fetchLatestVersion() CheckUpdateResult {
	current := strings.TrimPrefix(share.VERSION, "v")
	base := CheckUpdateResult{HasUpdate: false, CurrentVersion: current}

	url := cdnBase + "/latest.json"
	client := &http.Client{Timeout: 15 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return base
	}
	req.Header.Set("User-Agent", fmt.Sprintf("yao/%s", share.VERSION))

	resp, err := client.Do(req)
	if err != nil {
		return base
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return base
	}

	var data cdnLatest
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return base
	}

	latest := strings.TrimPrefix(data.Version, "v")
	if latest == "" {
		return base
	}

	platformKey := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	downloadURL := data.Assets[platformKey]

	return CheckUpdateResult{
		HasUpdate:      latest != current,
		CurrentVersion: current,
		LatestVersion:  latest,
		DownloadURL:    downloadURL,
	}
}

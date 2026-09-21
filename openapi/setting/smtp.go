package setting

import (
	"crypto/tls"
	_ "embed"
	"fmt"
	"net"
	"net/http"
	"net/smtp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yaoapp/yao/openapi/oauth/authorized"
	oauthTypes "github.com/yaoapp/yao/openapi/oauth/types"
	"github.com/yaoapp/yao/openapi/response"
	"github.com/yaoapp/yao/setting"
	"gopkg.in/yaml.v3"
)

//go:embed smtp_presets.yml
var smtpPresetsYML []byte

const smtpNS = "smtp"

var smtpPresetsMap map[string][]SmtpPreset

func init() {
	smtpPresetsMap = make(map[string][]SmtpPreset)
	if err := yaml.Unmarshal(smtpPresetsYML, &smtpPresetsMap); err != nil {
		smtpPresetsMap = map[string][]SmtpPreset{}
	}
}

func smtpGetPresets(locale string) []SmtpPreset {
	locale = strings.ToLower(locale)
	if presets, ok := smtpPresetsMap[locale]; ok {
		return presets
	}
	if presets, ok := smtpPresetsMap["en-us"]; ok {
		return presets
	}
	return nil
}

func smtpDefaultPreset(presets []SmtpPreset) *SmtpPreset {
	for i := range presets {
		if presets[i].Default {
			return &presets[i]
		}
	}
	if len(presets) > 0 {
		return &presets[0]
	}
	return nil
}

func smtpScope(info *oauthTypes.AuthorizedInfo) setting.ScopeID {
	if info.TeamID != "" {
		return setting.ScopeID{Scope: setting.ScopeTeam, TeamID: info.TeamID}
	}
	return setting.ScopeID{Scope: setting.ScopeUser, UserID: info.UserID}
}

// ---------------------------------------------------------------------------
// Rate limiter: 5 test emails per minute per scope
// ---------------------------------------------------------------------------

var (
	smtpRateMu    sync.Mutex
	smtpRateStore = map[string][]time.Time{}
)

const smtpRateLimit = 5
const smtpRateWindow = time.Minute

func smtpCheckRateLimit(key string) bool {
	smtpRateMu.Lock()
	defer smtpRateMu.Unlock()

	now := time.Now()
	cutoff := now.Add(-smtpRateWindow)

	var recent []time.Time
	for _, t := range smtpRateStore[key] {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}

	if len(recent) >= smtpRateLimit {
		smtpRateStore[key] = recent
		return false
	}

	smtpRateStore[key] = append(recent, now)
	return true
}

// ---------------------------------------------------------------------------
// GET /setting/smtp
// ---------------------------------------------------------------------------

func handleSmtpGet(c *gin.Context) {
	info := authorized.GetInfo(c)
	locale := c.Query("locale")
	if locale == "" {
		locale = "en-us"
	}

	presets := smtpGetPresets(locale)

	cfg := SmtpConfig{
		Enabled:    false,
		PresetKey:  "custom",
		Host:       "",
		Port:       465,
		Encryption: "ssl",
		Username:   "",
		Password:   "",
		FromName:   "",
		FromEmail:  "",
		Status:     "unconfigured",
	}

	hasSaved := false
	if setting.Global != nil {
		saved, _ := setting.Global.GetMerged(info.UserID, info.TeamID, smtpNS)
		if saved != nil {
			smtpLoadConfig(&cfg, saved)
			hasSaved = true
		}
	}

	if !hasSaved {
		if def := smtpDefaultPreset(presets); def != nil {
			cfg.PresetKey = def.Key
			cfg.Host = def.Host
			cfg.Port = def.Port
			cfg.Encryption = def.Encryption
		}
	}

	response.RespondWithSuccess(c, http.StatusOK, SmtpPageData{
		Presets: presets,
		Config:  cfg,
	})
}

// ---------------------------------------------------------------------------
// PUT /setting/smtp
// ---------------------------------------------------------------------------

func handleSmtpUpdate(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := smtpScope(info)

	var body struct {
		PresetKey  string `json:"preset_key"`
		Host       string `json:"host"`
		Port       int    `json:"port"`
		Encryption string `json:"encryption"`
		Username   string `json:"username"`
		Password   string `json:"password"`
		FromName   string `json:"from_name"`
		FromEmail  string `json:"from_email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	existing, _ := setting.Global.Get(scope, smtpNS)

	pwd := body.Password
	if pwd == "" {
		if v, ok := existing["password"].(string); ok && v != "" {
			pwd = decryptValue(v)
		}
	}

	validated := false
	if body.Host != "" && body.Username != "" && pwd != "" {
		if err := smtpValidateConnection(body.Host, body.Port, body.Encryption, body.Username, pwd); err != nil {
			respondError(c, http.StatusBadRequest, err.Error())
			return
		}
		validated = true
	}

	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}

	m["preset_key"] = body.PresetKey
	m["host"] = body.Host
	m["port"] = body.Port
	m["encryption"] = body.Encryption
	m["username"] = body.Username
	m["from_name"] = body.FromName
	m["from_email"] = body.FromEmail

	if body.Password != "" {
		m["password"] = encryptValue(body.Password)
	}

	if validated {
		m["status"] = "connected"
	}

	if _, err := setting.Global.Set(scope, smtpNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	cfg := SmtpConfig{
		PresetKey:  "custom",
		Port:       465,
		Encryption: "ssl",
		Status:     "unconfigured",
	}
	smtpLoadConfig(&cfg, m)

	response.RespondWithSuccess(c, http.StatusOK, cfg)
}

// ---------------------------------------------------------------------------
// PUT /setting/smtp/toggle
// ---------------------------------------------------------------------------

func handleSmtpToggle(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := smtpScope(info)

	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	existing, _ := setting.Global.Get(scope, smtpNS)
	m := make(map[string]interface{})
	for k, v := range existing {
		m[k] = v
	}
	m["enabled"] = body.Enabled
	if !body.Enabled {
		m["status"] = "unconfigured"
	}

	if _, err := setting.Global.Set(scope, smtpNS, m); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}

	cfg := SmtpConfig{
		PresetKey:  "custom",
		Port:       465,
		Encryption: "ssl",
		Status:     "unconfigured",
	}
	smtpLoadConfig(&cfg, m)

	response.RespondWithSuccess(c, http.StatusOK, cfg)
}

// ---------------------------------------------------------------------------
// POST /setting/smtp/test
// ---------------------------------------------------------------------------

func handleSmtpTest(c *gin.Context) {
	if !guardOwner(c) {
		return
	}
	info := authorized.GetInfo(c)
	scope := smtpScope(info)

	var body struct {
		ToEmail string `json:"to_email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		respondError(c, http.StatusBadRequest, "invalid request body")
		return
	}
	if strings.TrimSpace(body.ToEmail) == "" {
		respondError(c, http.StatusBadRequest, "to_email is required")
		return
	}

	rateKey := scope.TeamID
	if rateKey == "" {
		rateKey = scope.UserID
	}
	if !smtpCheckRateLimit(rateKey) {
		response.RespondWithSuccess(c, http.StatusOK, SmtpTestResult{
			Success: false,
			Message: "Rate limit exceeded, please wait a moment",
		})
		return
	}

	if setting.Global == nil {
		respondError(c, http.StatusInternalServerError, "setting registry not initialized")
		return
	}

	saved, _ := setting.Global.Get(scope, smtpNS)
	if saved == nil {
		response.RespondWithSuccess(c, http.StatusOK, SmtpTestResult{
			Success: false,
			Message: "SMTP not configured",
		})
		return
	}

	cfg := SmtpConfig{PresetKey: "custom", Port: 465, Encryption: "ssl", Status: "unconfigured"}
	smtpLoadConfig(&cfg, saved)

	if cfg.Host == "" || cfg.Username == "" {
		response.RespondWithSuccess(c, http.StatusOK, SmtpTestResult{
			Success: false,
			Message: "SMTP host and username are required",
		})
		return
	}

	password := ""
	if v, ok := saved["password"].(string); ok && v != "" {
		password = decryptValue(v)
	}

	fromAddr := cfg.FromEmail
	if fromAddr == "" {
		fromAddr = cfg.Username
	}

	err := smtpSendTestEmail(cfg.Host, cfg.Port, cfg.Encryption, cfg.Username, password, cfg.FromName, fromAddr, body.ToEmail)
	if err != nil {
		saved["status"] = "disconnected"
		setting.Global.Set(scope, smtpNS, saved)
		response.RespondWithSuccess(c, http.StatusOK, SmtpTestResult{
			Success: false,
			Message: err.Error(),
		})
		return
	}

	saved["status"] = "connected"
	saved["last_sent_at"] = time.Now().UTC().Format(time.RFC3339)
	setting.Global.Set(scope, smtpNS, saved)

	response.RespondWithSuccess(c, http.StatusOK, SmtpTestResult{
		Success: true,
		Message: "Test email sent successfully",
	})
}

// ---------------------------------------------------------------------------
// SMTP connection validation (dial + auth, no email)
// ---------------------------------------------------------------------------

func smtpValidateConnection(host string, port int, encryption, username, password string) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))
	auth := smtp.PlainAuth("", username, password, host)

	switch encryption {
	case "ssl":
		tlsConfig := &tls.Config{ServerName: host}
		conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("SSL connection failed: %s", err.Error())
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("SMTP client failed: %s", err.Error())
		}
		defer client.Quit()
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %s", err.Error())
		}
		return nil

	case "tls":
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return fmt.Errorf("connection failed: %s", err.Error())
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("SMTP client failed: %s", err.Error())
		}
		defer client.Quit()
		if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
			return fmt.Errorf("STARTTLS failed: %s", err.Error())
		}
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %s", err.Error())
		}
		return nil

	default:
		conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
		if err != nil {
			return fmt.Errorf("connection failed: %s", err.Error())
		}
		defer conn.Close()
		client, err := smtp.NewClient(conn, host)
		if err != nil {
			return fmt.Errorf("SMTP client failed: %s", err.Error())
		}
		defer client.Quit()
		if err = client.Auth(auth); err != nil {
			return fmt.Errorf("authentication failed: %s", err.Error())
		}
		return nil
	}
}

// ---------------------------------------------------------------------------
// SMTP send helper
// ---------------------------------------------------------------------------

func smtpSendTestEmail(host string, port int, encryption, username, password, fromName, fromEmail, toEmail string) error {
	addr := net.JoinHostPort(host, fmt.Sprintf("%d", port))

	subject := "Yao SMTP Test"
	body := "This is a test email from Yao to verify your SMTP configuration."

	from := fromEmail
	if fromName != "" {
		from = fmt.Sprintf("%s <%s>", fromName, fromEmail)
	}

	msg := fmt.Sprintf("From: %s\r\nTo: %s\r\nSubject: %s\r\nMIME-Version: 1.0\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		from, toEmail, subject, body)

	auth := smtp.PlainAuth("", username, password, host)

	switch encryption {
	case "ssl":
		return smtpSendSSL(addr, host, auth, fromEmail, toEmail, []byte(msg))
	case "tls":
		return smtpSendStartTLS(addr, host, auth, fromEmail, toEmail, []byte(msg))
	default:
		return smtp.SendMail(addr, auth, fromEmail, []string{toEmail}, []byte(msg))
	}
}

func smtpSendSSL(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	tlsConfig := &tls.Config{ServerName: host}
	conn, err := tls.DialWithDialer(&net.Dialer{Timeout: 10 * time.Second}, "tcp", addr, tlsConfig)
	if err != nil {
		return fmt.Errorf("SSL connection failed: %s", err.Error())
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %s", err.Error())
	}
	defer client.Quit()

	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %s", err.Error())
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %s", err.Error())
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %s", err.Error())
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %s", err.Error())
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write failed: %s", err.Error())
	}
	return w.Close()
}

func smtpSendStartTLS(addr, host string, auth smtp.Auth, from, to string, msg []byte) error {
	conn, err := net.DialTimeout("tcp", addr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("connection failed: %s", err.Error())
	}
	defer conn.Close()

	client, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP client failed: %s", err.Error())
	}
	defer client.Quit()

	if err = client.StartTLS(&tls.Config{ServerName: host}); err != nil {
		return fmt.Errorf("STARTTLS failed: %s", err.Error())
	}
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("authentication failed: %s", err.Error())
	}
	if err = client.Mail(from); err != nil {
		return fmt.Errorf("MAIL FROM failed: %s", err.Error())
	}
	if err = client.Rcpt(to); err != nil {
		return fmt.Errorf("RCPT TO failed: %s", err.Error())
	}

	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("DATA failed: %s", err.Error())
	}
	if _, err = w.Write(msg); err != nil {
		return fmt.Errorf("write failed: %s", err.Error())
	}
	return w.Close()
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func smtpLoadConfig(cfg *SmtpConfig, m map[string]interface{}) {
	if v, ok := m["enabled"].(bool); ok {
		cfg.Enabled = v
	}
	if v, ok := m["preset_key"].(string); ok && v != "" {
		cfg.PresetKey = v
	}
	if v, ok := m["host"].(string); ok {
		cfg.Host = v
	}
	if v, ok := m["port"]; ok {
		switch p := v.(type) {
		case int:
			cfg.Port = p
		case float64:
			cfg.Port = int(p)
		case int64:
			cfg.Port = int(p)
		}
	}
	if v, ok := m["encryption"].(string); ok && v != "" {
		cfg.Encryption = v
	}
	if v, ok := m["username"].(string); ok {
		cfg.Username = v
	}
	if v, ok := m["password"].(string); ok && v != "" {
		cfg.Password = maskKey(decryptValue(v))
	}
	if v, ok := m["from_name"].(string); ok {
		cfg.FromName = v
	}
	if v, ok := m["from_email"].(string); ok {
		cfg.FromEmail = v
	}
	if v, ok := m["status"].(string); ok && v != "" {
		cfg.Status = v
	}
	if v, ok := m["last_sent_at"].(string); ok && v != "" {
		cfg.LastSentAt = v
	}
}

package volume

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	gitfmtcfg "github.com/go-git/go-git/v5/plumbing/format/config"
)

type credEntry struct {
	Host     string
	Username string
	Token    string
}

// initGitConfig ensures git/config contains a [credential] section with the
// correct helper priority chain: reset (clear system helpers) → store
// (workspace credentials first) → platform fallback (system credential
// manager as fallback). Upgrades existing configs that use the old format.
func initGitConfig(wsBase string) error {
	cfgPath := filepath.Join(wsBase, "git", "config")
	data, err := os.ReadFile(cfgPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	if !bytes.Contains(data, []byte("[credential]")) {
		block := credentialBlock()
		if len(data) > 0 && !bytes.HasSuffix(data, []byte("\n")) {
			block = "\n" + block
		}
		return writeSecureFile(cfgPath, append(data, []byte(block)...))
	}

	return upgradeCredentialSection(cfgPath, data)
}

func credentialHelpers() []string {
	helpers := []string{"", "store"}
	switch runtime.GOOS {
	case "windows":
		helpers = append(helpers, "manager")
	case "darwin":
		helpers = append(helpers, "osxkeychain")
	}
	return helpers
}

func credentialBlock() string {
	block := "[credential]\n"
	for _, h := range credentialHelpers() {
		block += "\thelper = " + h + "\n"
	}
	return block
}

func upgradeCredentialSection(cfgPath string, data []byte) error {
	cfg := gitfmtcfg.New()
	if err := gitfmtcfg.NewDecoder(bytes.NewReader(data)).Decode(cfg); err != nil {
		return fmt.Errorf("parse config: %w", err)
	}

	desired := credentialHelpers()
	sect := cfg.Section("credential")

	var current []string
	for _, opt := range sect.Options {
		if opt.IsKey("helper") {
			current = append(current, opt.Value)
		}
	}

	if len(current) == len(desired) {
		match := true
		for i := range current {
			if current[i] != desired[i] {
				match = false
				break
			}
		}
		if match {
			return nil
		}
	}

	var kept gitfmtcfg.Options
	for _, opt := range sect.Options {
		if !opt.IsKey("helper") {
			kept = append(kept, opt)
		}
	}
	for _, h := range desired {
		kept = append(kept, &gitfmtcfg.Option{Key: "helper", Value: h})
	}
	sect.Options = kept

	var buf bytes.Buffer
	if err := gitfmtcfg.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return writeSecureFile(cfgPath, buf.Bytes())
}

// readCredentialFile parses a standard .git-credentials file.
func readCredentialFile(path string) ([]credEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()

	var entries []credEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		u, err := url.Parse(line)
		if err != nil || u.User == nil || u.Host == "" {
			continue
		}
		pw, _ := u.User.Password()
		entries = append(entries, credEntry{
			Host:     u.Host,
			Username: u.User.Username(),
			Token:    pw,
		})
	}
	return entries, scanner.Err()
}

// writeCredentialFile serialises entries to .git-credentials format.
func writeCredentialFile(path string, entries []credEntry) error {
	var buf bytes.Buffer
	for _, e := range entries {
		fmt.Fprintf(&buf, "https://%s:%s@%s\n",
			url.PathEscape(e.Username), url.PathEscape(e.Token), e.Host)
	}
	return writeSecureFile(path, buf.Bytes())
}

// parseConfigKey splits "section.key" into (section, subsection, key).
func parseConfigKey(raw string) (section, subsection, key string) {
	parts := strings.SplitN(raw, ".", 3)
	switch len(parts) {
	case 2:
		return parts[0], "", parts[1]
	case 3:
		return parts[0], parts[1], parts[2]
	default:
		return raw, "", ""
	}
}

// ---------------------------------------------------------------------------
// localStorage: GitConfigGet
// ---------------------------------------------------------------------------

func (l *localStorage) GitConfigGet(_ context.Context, sessionID, key string) (map[string]string, error) {
	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return nil, err
	}
	cfgPath := filepath.Join(wsBase, "git", "config")

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}

	cfg := gitfmtcfg.New()
	if err := gitfmtcfg.NewDecoder(bytes.NewReader(data)).Decode(cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	values := map[string]string{}

	if key != "" {
		sec, sub, k := parseConfigKey(key)
		var val string
		if sub != "" {
			val = cfg.Section(sec).Subsection(sub).Option(k)
		} else {
			val = cfg.Section(sec).Option(k)
		}
		if val != "" {
			values[key] = val
		}
	} else {
		for _, sec := range cfg.Sections {
			for _, opt := range sec.Options {
				values[sec.Name+"."+opt.Key] = opt.Value
			}
			for _, sub := range sec.Subsections {
				for _, opt := range sub.Options {
					values[sec.Name+"."+sub.Name+"."+opt.Key] = opt.Value
				}
			}
		}
	}

	return values, nil
}

// ---------------------------------------------------------------------------
// localStorage: GitConfigSet
// ---------------------------------------------------------------------------

func (l *localStorage) GitConfigSet(_ context.Context, sessionID, key, value string) error {
	if key == "" {
		return fmt.Errorf("key is required")
	}

	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return err
	}
	if err := ensureConfigDirs(wsBase); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}

	cfgPath := filepath.Join(wsBase, "git", "config")
	data, _ := os.ReadFile(cfgPath)

	cfg := gitfmtcfg.New()
	if len(data) > 0 {
		if err := gitfmtcfg.NewDecoder(bytes.NewReader(data)).Decode(cfg); err != nil {
			return fmt.Errorf("parse config: %w", err)
		}
	}

	sec, sub, k := parseConfigKey(key)
	if sub != "" {
		cfg.Section(sec).Subsection(sub).SetOption(k, value)
	} else {
		cfg.Section(sec).SetOption(k, value)
	}

	var buf bytes.Buffer
	if err := gitfmtcfg.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}

	return writeSecureFile(cfgPath, buf.Bytes())
}

// ---------------------------------------------------------------------------
// localStorage: GitCredentialSet
// ---------------------------------------------------------------------------

func (l *localStorage) GitCredentialSet(_ context.Context, sessionID, host, username, token string) error {
	if host == "" || token == "" {
		return fmt.Errorf("host and token are required")
	}

	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return err
	}
	if err := ensureConfigDirs(wsBase); err != nil {
		return fmt.Errorf("ensure dirs: %w", err)
	}
	if err := initGitConfig(wsBase); err != nil {
		return fmt.Errorf("init git config: %w", err)
	}

	credPath := filepath.Join(wsBase, "git", "credentials")
	entries, _ := readCredentialFile(credPath)

	filtered := make([]credEntry, 0, len(entries))
	for _, e := range entries {
		if !strings.EqualFold(e.Host, host) {
			filtered = append(filtered, e)
		}
	}

	if username == "" {
		username = "x-access-token"
	}
	filtered = append(filtered, credEntry{Host: host, Username: username, Token: token})

	return writeCredentialFile(credPath, filtered)
}

// ---------------------------------------------------------------------------
// localStorage: GitCredentialList
// ---------------------------------------------------------------------------

func (l *localStorage) GitCredentialList(_ context.Context, sessionID string) ([]GitCredentialEntry, error) {
	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return nil, err
	}
	credPath := filepath.Join(wsBase, "git", "credentials")

	entries, err := readCredentialFile(credPath)
	if err != nil {
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	var result []GitCredentialEntry
	for _, e := range entries {
		result = append(result, GitCredentialEntry{
			Host:     e.Host,
			Username: e.Username,
		})
	}

	return result, nil
}

// ---------------------------------------------------------------------------
// localStorage: GitCredentialDelete
// ---------------------------------------------------------------------------

func (l *localStorage) GitCredentialDelete(_ context.Context, sessionID, host string) error {
	if host == "" {
		return fmt.Errorf("host is required")
	}

	wsBase, err := l.wsBase(sessionID)
	if err != nil {
		return err
	}
	credPath := filepath.Join(wsBase, "git", "credentials")

	entries, err := readCredentialFile(credPath)
	if err != nil {
		return fmt.Errorf("read credentials: %w", err)
	}
	if entries == nil {
		return nil
	}

	filtered := make([]credEntry, 0, len(entries))
	for _, e := range entries {
		if !strings.EqualFold(e.Host, host) {
			filtered = append(filtered, e)
		}
	}

	return writeCredentialFile(credPath, filtered)
}

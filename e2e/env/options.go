package env

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"time"
)

// Options controls one E2E run.
type Options struct {
	RepoRoot     string
	EnvFile      string
	DB           string
	PostgresDSN  string
	TaiDir       string
	TaiBin       string
	Keep         bool
	Timeout      time.Duration
	PGPrefix     string
	PGKeep       bool
	ArtifactsDir string
	RunID        string
	Profile      bool
	OpenAIKey    string
	DeepSeekKey  string
}

var runIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

// LoadOptions loads .env.common, the platform .env file, an optional explicit
// file and finally process environment variables.
func LoadOptions() (Options, error) {
	root, err := repoRoot()
	if err != nil {
		return Options{}, err
	}

	values := map[string]string{}
	for _, file := range []string{
		filepath.Join(root, "e2e", ".env.common"),
		filepath.Join(root, "e2e", ".env."+runtime.GOOS),
	} {
		if _, err := os.Stat(file); err == nil {
			parsed, err := parseDotEnv(file)
			if err != nil {
				return Options{}, err
			}
			for key, value := range parsed {
				values[key] = value
			}
		}
	}

	explicit := os.Getenv("YAO_E2E_ENV_FILE")
	if explicit != "" {
		fileValues, err := parseDotEnv(explicit)
		if err != nil {
			return Options{}, fmt.Errorf("YAO_E2E_ENV_FILE: %w", err)
		}
		for key, value := range fileValues {
			values[key] = value
		}
	}

	for _, key := range []string{
		"YAO_E2E_DB", "YAO_E2E_POSTGRES_DSN", "YAO_E2E_TAI_DIR",
		"YAO_E2E_TAI_BIN", "YAO_E2E_KEEP", "YAO_E2E_TIMEOUT",
		"YAO_E2E_PG_PREFIX", "YAO_E2E_PG_KEEP", "YAO_E2E_ARTIFACTS_DIR",
		"YAO_E2E_RUN_ID", "YAO_E2E_PROFILE", "OPENAI_KEY",
		"DEEPSEEK_KEY", "OPENAI_BASE_URL", "DEEPSEEK_BASE_URL",
	} {
		if value, ok := os.LookupEnv(key); ok {
			values[key] = value
		}
	}

	opts := Options{
		RepoRoot:     root,
		EnvFile:      explicit,
		DB:           first(values["YAO_E2E_DB"], "sqlite3"),
		PostgresDSN:  values["YAO_E2E_POSTGRES_DSN"],
		TaiDir:       first(values["YAO_E2E_TAI_DIR"], defaultTaiDir(root)),
		TaiBin:       values["YAO_E2E_TAI_BIN"],
		Keep:         values["YAO_E2E_KEEP"] == "1",
		Timeout:      parseDuration(values["YAO_E2E_TIMEOUT"], 10*time.Minute),
		PGPrefix:     first(values["YAO_E2E_PG_PREFIX"], "yao_e2e_"),
		PGKeep:       values["YAO_E2E_PG_KEEP"] == "1",
		ArtifactsDir: first(values["YAO_E2E_ARTIFACTS_DIR"], filepath.Join("e2e", "logs")),
		RunID:        values["YAO_E2E_RUN_ID"],
		Profile:      values["YAO_E2E_PROFILE"] == "1",
		OpenAIKey:    values["OPENAI_KEY"],
		DeepSeekKey:  values["DEEPSEEK_KEY"],
	}
	if opts.RunID == "" {
		opts.RunID, err = newRunID()
		if err != nil {
			return Options{}, err
		}
	}
	if !runIDPattern.MatchString(opts.RunID) {
		return Options{}, fmt.Errorf("invalid YAO_E2E_RUN_ID %q", opts.RunID)
	}
	if !filepath.IsAbs(opts.ArtifactsDir) {
		opts.ArtifactsDir = filepath.Join(root, opts.ArtifactsDir)
	}
	if err := opts.validate(); err != nil {
		return Options{}, err
	}
	return opts, nil
}

func (o Options) validate() error {
	switch o.DB {
	case "postgres":
		if o.PostgresDSN == "" {
			return fmt.Errorf("YAO_E2E_POSTGRES_DSN is required when YAO_E2E_DB=postgres")
		}
	case "sqlite3":
	default:
		return fmt.Errorf("unsupported YAO_E2E_DB %q", o.DB)
	}
	if o.TaiDir == "" {
		return fmt.Errorf("YAO_E2E_TAI_DIR is required")
	}
	info, err := os.Stat(o.TaiDir)
	if err != nil || !info.IsDir() {
		return fmt.Errorf("YAO_E2E_TAI_DIR not found: %s", o.TaiDir)
	}
	if o.Timeout <= 0 {
		return fmt.Errorf("YAO_E2E_TIMEOUT must be positive")
	}
	return nil
}

func repoRoot() (string, error) {
	_, file, _, ok := runtime.Caller(0)
	if ok {
		return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..")), nil
	}
	return os.Getwd()
}

func defaultTaiDir(root string) string {
	return filepath.Clean(filepath.Join(root, "..", "tai"))
}

func first(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func parseDuration(value string, fallback time.Duration) time.Duration {
	if value == "" {
		return fallback
	}
	d, err := time.ParseDuration(value)
	if err != nil {
		return fallback
	}
	return d
}

func newRunID() (string, error) {
	buf := make([]byte, 3)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return fmt.Sprintf("%s-%s-%s-p%d-%s",
		time.Now().UTC().Format("20060102T150405Z"),
		runtime.GOOS, runtime.GOARCH, os.Getpid(), hex.EncodeToString(buf)), nil
}

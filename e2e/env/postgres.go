package env

import (
	"database/sql"
	"fmt"
	"net/url"
	"os"
	"strings"

	_ "github.com/lib/pq"
)

type postgresDatabase struct {
	name string
	dsn  string
	base *sql.DB
	keep bool
}

func newPostgres(opts Options) (Database, error) {
	base, err := sql.Open("postgres", opts.PostgresDSN)
	if err != nil {
		return nil, err
	}
	if err := base.Ping(); err != nil {
		base.Close()
		return nil, fmt.Errorf("connect postgres: %w", err)
	}

	name := fmt.Sprintf("%s%s_%d_%s", opts.PGPrefix, platformName(), os.Getpid(), randomSuffix())
	// randomSuffix already contains pid-less entropy; keep the DB identifier short.
	if _, err := base.Exec("CREATE DATABASE " + quoteIdent(name)); err != nil {
		base.Close()
		return nil, fmt.Errorf("create test database %s: %w", name, err)
	}

	testDSN, err := replaceDatabase(opts.PostgresDSN, name)
	if err != nil {
		_, _ = base.Exec("DROP DATABASE " + quoteIdent(name) + " WITH (FORCE)")
		base.Close()
		return nil, err
	}
	return &postgresDatabase{name: name, dsn: testDSN, base: base, keep: opts.PGKeep}, nil
}

func (d *postgresDatabase) Name() string   { return d.name }
func (d *postgresDatabase) Driver() string { return "postgres" }
func (d *postgresDatabase) DSN() string    { return d.dsn }
func (d *postgresDatabase) Close() error {
	if d.base == nil {
		return nil
	}
	if d.keep {
		return d.base.Close()
	}
	var err error
	if _, dropErr := d.base.Exec("DROP DATABASE " + quoteIdent(d.name) + " WITH (FORCE)"); dropErr != nil {
		_, err = d.base.Exec("DROP DATABASE " + quoteIdent(d.name))
	}
	if closeErr := d.base.Close(); err == nil {
		err = closeErr
	}
	d.base = nil
	return err
}

func replaceDatabase(dsn, name string) (string, error) {
	u, err := url.Parse(dsn)
	if err != nil {
		return "", fmt.Errorf("parse postgres DSN: %w", err)
	}
	u.Path = "/" + name
	return u.String(), nil
}

func quoteIdent(value string) string {
	return `"` + strings.ReplaceAll(value, `"`, `""`) + `"`
}

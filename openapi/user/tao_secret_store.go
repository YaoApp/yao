package user

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/config"
)

type taoClientCredentials struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
}

func (c *taoClientCredentials) complete() bool {
	return c != nil && c.ClientID != "" && c.ClientSecret != ""
}

func taoClientStorePath() string {
	return filepath.Join(config.Conf.Root, ".tao", "client.json")
}

func loadTaoClientCredentials() (*taoClientCredentials, error) {
	id := os.Getenv("TAO_OAUTH_CLIENT_ID")
	secret := os.Getenv("TAO_OAUTH_CLIENT_SECRET")
	if id != "" || secret != "" {
		if id == "" || secret == "" {
			return nil, fmt.Errorf("TAO_OAUTH_CLIENT_ID and TAO_OAUTH_CLIENT_SECRET must both be set")
		}
		return &taoClientCredentials{ClientID: id, ClientSecret: secret}, nil
	}
	raw, err := os.ReadFile(taoClientStorePath())
	if os.IsNotExist(err) {
		return &taoClientCredentials{}, nil
	}
	if err != nil {
		return nil, err
	}
	var creds taoClientCredentials
	if err := json.Unmarshal(raw, &creds); err != nil {
		return nil, fmt.Errorf("parse tao client store: %w", err)
	}
	return &creds, nil
}

func saveTaoClientCredentials(id, secret string) error {
	if id == "" || secret == "" {
		return fmt.Errorf("client id and secret are required")
	}
	path := taoClientStorePath()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return err
	}
	data, _ := json.Marshal(taoClientCredentials{ClientID: id, ClientSecret: secret})
	return os.WriteFile(path, data, 0600)
}

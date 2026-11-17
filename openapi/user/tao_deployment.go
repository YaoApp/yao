package user

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"

	"github.com/yaoapp/yao/config"
)

type taoDeployment struct {
	ID         string `json:"deployment_id"`
	PublicKey  string `json:"deployment_public_key"`
	PrivateKey string `json:"-"`
}

func taoDeploymentPath() string {
	return filepath.Join(config.Conf.Root, ".tao", "deployment.json")
}

func loadTaoDeployment() (*taoDeployment, error) {
	if id := os.Getenv("TAO_DEPLOYMENT_ID"); id != "" {
		pub := os.Getenv("TAO_DEPLOYMENT_PUBLIC_KEY")
		if pub == "" {
			return nil, fmt.Errorf("TAO_DEPLOYMENT_PUBLIC_KEY is required with TAO_DEPLOYMENT_ID")
		}
		return &taoDeployment{ID: id, PublicKey: pub}, nil
	}

	path := taoDeploymentPath()
	raw, err := os.ReadFile(path)
	if err == nil {
		var d taoDeployment
		if err := json.Unmarshal(raw, &d); err != nil {
			return nil, fmt.Errorf("parse tao deployment: %w", err)
		}
		return &d, nil
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read tao deployment: %w", err)
	}

	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate deployment key: %w", err)
	}
	pubDER, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return nil, fmt.Errorf("marshal deployment public key: %w", err)
	}
	privDER, err := x509.MarshalPKCS8PrivateKey(priv)
	if err != nil {
		return nil, fmt.Errorf("marshal deployment private key: %w", err)
	}
	idBytes := make([]byte, 16)
	if _, err := rand.Read(idBytes); err != nil {
		return nil, fmt.Errorf("generate deployment id: %w", err)
	}
	d := &taoDeployment{
		ID:         hex.EncodeToString(idBytes),
		PublicKey:  string(pem.EncodeToMemory(&pem.Block{Type: "PUBLIC KEY", Bytes: pubDER})),
		PrivateKey: string(pem.EncodeToMemory(&pem.Block{Type: "PRIVATE KEY", Bytes: privDER})),
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return nil, fmt.Errorf("create tao deployment dir: %w", err)
	}
	stored := struct {
		ID         string `json:"deployment_id"`
		PublicKey  string `json:"deployment_public_key"`
		PrivateKey string `json:"deployment_private_key"`
	}{d.ID, d.PublicKey, d.PrivateKey}
	data, _ := json.Marshal(stored)
	if err := os.WriteFile(path, data, 0600); err != nil {
		return nil, fmt.Errorf("write tao deployment: %w", err)
	}
	return d, nil
}

func decodeTaoDeploymentPrivateKey(pemText string) (ed25519.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemText))
	if block == nil {
		return nil, fmt.Errorf("invalid private key PEM")
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}
	priv, ok := key.(ed25519.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("unsupported private key type")
	}
	return priv, nil
}

// keep base64 imported for future PEM/JWK variants and tests.
var _ = base64.RawURLEncoding

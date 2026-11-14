package computer

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/yaoapp/yao/config"
	"golang.org/x/crypto/ssh"
)

// ControlKey is the system-level SSH identity used by Yao to control cloud
// hosts. It is deliberately separate from user Vault credentials.
type ControlKey struct {
	KeyID      string
	PublicKey  string
	PrivateKey string
}

// LoadControlKey loads or creates the Yao Tai control key.
func LoadControlKey() (*ControlKey, error) {
	keyID := strings.TrimSpace(os.Getenv("YAO_TAI_CONTROL_KEY_ID"))
	publicEnv := strings.TrimSpace(os.Getenv("YAO_TAI_CONTROL_PUBLIC_KEY"))
	if raw := os.Getenv("YAO_TAI_CONTROL_PRIVATE_KEY"); strings.TrimSpace(raw) != "" {
		return controlKeyFromPrivate(raw, keyID, publicEnv)
	}

	privateFile := os.Getenv("YAO_TAI_CONTROL_PRIVATE_KEY_FILE")
	if privateFile == "" {
		root := config.Conf.DataRoot
		if root == "" {
			root = config.Conf.Root
		}
		if root == "" {
			root = "."
		}
		privateFile = filepath.Join(root, "tai", "control", "ed25519")
	}
	if err := os.MkdirAll(filepath.Dir(privateFile), 0700); err != nil {
		return nil, fmt.Errorf("create tai control key dir: %w", err)
	}

	if raw, err := os.ReadFile(privateFile); err == nil {
		return controlKeyFromPrivate(string(raw), keyID, publicEnv)
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read tai control key: %w", err)
	}

	_, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate tai control key: %w", err)
	}
	block, err := ssh.MarshalPrivateKey(privateKey, "")
	if err != nil {
		return nil, fmt.Errorf("marshal tai control key: %w", err)
	}
	raw := string(pem.EncodeToMemory(block))
	if err := os.WriteFile(privateFile, []byte(raw), 0600); err != nil {
		return nil, fmt.Errorf("write tai control key: %w", err)
	}
	key, err := controlKeyFromPrivate(raw, keyID, publicEnv)
	if err != nil {
		return nil, err
	}
	_ = os.WriteFile(privateFile+".pub", []byte(key.PublicKey+"\n"), 0644)
	return key, nil
}

func controlKeyFromPrivate(raw, keyID, expectedPublic string) (*ControlKey, error) {
	signer, err := ssh.ParsePrivateKey([]byte(raw))
	if err != nil {
		return nil, fmt.Errorf("parse tai control key: %w", err)
	}
	publicKey := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(signer.PublicKey())))
	if expectedPublic != "" {
		expected, _, _, _, err := ssh.ParseAuthorizedKey([]byte(expectedPublic))
		if err != nil {
			return nil, fmt.Errorf("parse configured tai control public key: %w", err)
		}
		if !strings.EqualFold(strings.TrimSpace(string(ssh.MarshalAuthorizedKey(expected))), publicKey) {
			return nil, fmt.Errorf("configured tai control public key does not match private key")
		}
	}
	if keyID == "" {
		keyID = "tai-control-" + strings.TrimPrefix(ssh.FingerprintSHA256(signer.PublicKey()), "SHA256:")[:12]
	}
	privateKey := strings.TrimSpace(raw)
	return &ControlKey{KeyID: keyID, PublicKey: publicKey, PrivateKey: privateKey}, nil
}

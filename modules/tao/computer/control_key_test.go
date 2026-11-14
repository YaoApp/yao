package computer_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/yaoapp/yao/config"
	taocomputer "github.com/yaoapp/yao/modules/tao/computer"
	"golang.org/x/crypto/ssh"
)

func TestLoadControlKeyGeneratesAndReloads(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "keys", "control")
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY_FILE", path)
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", "")
	t.Setenv("YAO_TAI_CONTROL_KEY_ID", "")
	key, err := taocomputer.LoadControlKey()
	if err != nil {
		t.Fatal(err)
	}
	if key.KeyID == "" || key.PublicKey == "" || key.PrivateKey == "" {
		t.Fatalf("key=%+v", key)
	}
	if st, err := os.Stat(path); err != nil {
		t.Fatal(err)
	} else if runtime.GOOS != "windows" && st.Mode().Perm() != 0600 {
		t.Fatalf("mode=%v", st.Mode())
	}
	if _, err := ssh.ParsePrivateKey([]byte(key.PrivateKey)); err != nil {
		t.Fatalf("private key: %v", err)
	}
	again, err := taocomputer.LoadControlKey()
	if err != nil {
		t.Fatal(err)
	}
	if again.PublicKey != key.PublicKey || again.KeyID != key.KeyID {
		t.Fatalf("reloaded=%+v", again)
	}
}

func TestLoadControlKeyFromEnv(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(privateKey, "")
	raw := string(pem.EncodeToMemory(block))
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", raw)
	t.Setenv("YAO_TAI_CONTROL_KEY_ID", "env-key")
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY_FILE", "")
	key, err := taocomputer.LoadControlKey()
	if err != nil {
		t.Fatal(err)
	}
	if key.KeyID != "env-key" || key.PublicKey == "" {
		t.Fatalf("key=%+v", key)
	}
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", "bad")
	if _, err := taocomputer.LoadControlKey(); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadControlKeyFileErrors(t *testing.T) {
	dir := t.TempDir()
	bad := filepath.Join(dir, "bad")
	if err := os.WriteFile(bad, []byte("not-a-key"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", "")
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY_FILE", bad)
	if _, err := taocomputer.LoadControlKey(); err == nil {
		t.Fatal("expected parse error")
	}
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY_FILE", dir)
	if _, err := taocomputer.LoadControlKey(); err == nil {
		t.Fatal("expected read error")
	}
}

func TestControlKeyDefaultID(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(privateKey, "")
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", string(pem.EncodeToMemory(block)))
	t.Setenv("YAO_TAI_CONTROL_KEY_ID", "")
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY_FILE", "")
	key, err := taocomputer.LoadControlKey()
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(key.KeyID, "tai-control-") {
		t.Fatalf("key id=%q", key.KeyID)
	}
}

func TestLoadControlKeyPublicMismatch(t *testing.T) {
	_, privateKey, _ := ed25519.GenerateKey(rand.Reader)
	block, _ := ssh.MarshalPrivateKey(privateKey, "")
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", string(pem.EncodeToMemory(block)))
	t.Setenv("YAO_TAI_CONTROL_PUBLIC_KEY", "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAI invalid")
	if _, err := taocomputer.LoadControlKey(); err == nil {
		t.Fatal("expected public key error")
	}
	t.Setenv("YAO_TAI_CONTROL_PUBLIC_KEY", "bad")
	if _, err := taocomputer.LoadControlKey(); err == nil {
		t.Fatal("expected parse error")
	}
}

func TestLoadControlKeyDefaultPathAndPublicMatch(t *testing.T) {
	root := t.TempDir()
	oldRoot, oldData := config.Conf.Root, config.Conf.DataRoot
	config.Conf.Root = root
	config.Conf.DataRoot = root
	t.Cleanup(func() { config.Conf.Root = oldRoot; config.Conf.DataRoot = oldData })
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", "")
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY_FILE", "")
	t.Setenv("YAO_TAI_CONTROL_KEY_ID", "file-key")
	key, err := taocomputer.LoadControlKey()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, "tai", "control", "ed25519")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("YAO_TAI_CONTROL_PRIVATE_KEY", key.PrivateKey)
	t.Setenv("YAO_TAI_CONTROL_PUBLIC_KEY", key.PublicKey)
	matched, err := taocomputer.LoadControlKey()
	if err != nil {
		t.Fatal(err)
	}
	if matched.PublicKey != key.PublicKey {
		t.Fatalf("matched=%+v", matched)
	}
}

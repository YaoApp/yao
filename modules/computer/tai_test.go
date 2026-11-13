package computer

import (
	"context"
	"strings"
	"testing"

	"github.com/yaoapp/yao/modules/computer/types"
)

func TestShellQuote(t *testing.T) {
	cases := map[string]string{"": "''", "abc": "'abc'", "a'b": "'a'\\''b'"}
	for in, want := range cases {
		if got := shellQuote(in); got != want {
			t.Fatalf("shellQuote(%q)=%q want %q", in, got, want)
		}
	}
	quoted := quoteArgs([]string{"tai", "a b"})
	if strings.Join(quoted, " ") != "'tai' 'a b'" {
		t.Fatalf("quoteArgs=%v", quoted)
	}
}

func TestSSHTaiControllerRequiresTarget(t *testing.T) {
	c := NewSSHTaiController()
	if _, err := c.run(context.Background(), types.SSHTarget{}, "true"); err == nil {
		t.Fatal("expected target error")
	}
	if _, err := c.run(context.Background(), types.SSHTarget{Host: "127.0.0.1", PrivateKey: "bad"}, "true"); err == nil {
		t.Fatal("expected key parse error")
	}
}

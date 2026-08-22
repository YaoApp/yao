//go:build unit

package shared_test

import (
	"strings"
	"testing"

	"github.com/yaoapp/yao/agent/sandbox/v2/shared"
)

func TestPsEnsureDir(t *testing.T) {
	var b strings.Builder
	shared.PsEnsureDir(&b, `C:\workspace\.yao\dsh`)
	out := b.String()

	if !strings.Contains(out, "Test-Path") {
		t.Error("should contain Test-Path guard")
	}
	if !strings.Contains(out, "New-Item") {
		t.Error("should contain New-Item")
	}
	if !strings.Contains(out, `C:\workspace\.yao\dsh`) {
		t.Error("should contain the directory path")
	}
}

func TestPsWriteFileUTF8_ContainsHereString(t *testing.T) {
	var b strings.Builder
	shared.PsWriteFileUTF8(&b, `C:\workspace\.yao\dsh\cordis.yml`, "name: test-plugin")
	out := b.String()

	if !strings.Contains(out, "WriteAllText") {
		t.Error("should contain WriteAllText")
	}
	if !strings.Contains(out, "UTF8Encoding") {
		t.Error("should use UTF8Encoding without BOM")
	}
	if !strings.Contains(out, "name: test-plugin") {
		t.Error("should contain file content verbatim")
	}
	if !strings.Contains(out, "Test-Path") {
		t.Error("should create parent directory")
	}
}

func TestPsWriteFileUTF8_PreservesSingleQuotes(t *testing.T) {
	var b strings.Builder
	shared.PsWriteFileUTF8(&b, `C:\ws\config.yml`, "  name: '@deepseek-ai/dsh-yaoapp-jsonrpc-stream'")
	out := b.String()

	if !strings.Contains(out, "'@deepseek-ai/dsh-yaoapp-jsonrpc-stream'") {
		t.Error("here-string content must preserve single quotes verbatim")
	}
	if strings.Contains(out, "''@deepseek-ai/dsh-yaoapp-jsonrpc-stream''") {
		t.Error("single quotes must not be doubled inside a here-string")
	}
}

func TestPsSearchExe(t *testing.T) {
	var b strings.Builder
	shared.PsSearchExe(&b, "tai.exe")
	out := b.String()

	if !strings.Contains(out, "Get-ChildItem") {
		t.Error("should search user directories")
	}
	if !strings.Contains(out, "tai.exe") {
		t.Error("should search for the specified executable")
	}
	if !strings.Contains(out, ".local\\bin") {
		t.Errorf("should search .local\\bin, got: %s", out)
	}
	if !strings.Contains(out, "APPDATA") {
		t.Error("should also add APPDATA\\npm to PATH")
	}
}

func TestPsQuoteArg_Simple(t *testing.T) {
	got := shared.PsQuoteArg("hello")
	if got != "'hello'" {
		t.Errorf("PsQuoteArg(hello) = %q, want 'hello'", got)
	}
}

func TestPsQuoteArg_EmbeddedQuote(t *testing.T) {
	got := shared.PsQuoteArg("it's")
	if got != "'it''s'" {
		t.Errorf("PsQuoteArg(it's) = %q, want 'it''s'", got)
	}
}

func TestPsQuoteArg_WindowsPath(t *testing.T) {
	got := shared.PsQuoteArg(`C:\workspace\.yao\dsh\cordis.yml`)
	if got != `'C:\workspace\.yao\dsh\cordis.yml'` {
		t.Errorf("PsQuoteArg = %q", got)
	}
}

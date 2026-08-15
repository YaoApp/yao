package shared

import (
	"fmt"
	"strings"
)

// PsWriteFileUTF8 appends PowerShell statements that create the parent
// directory and write content to path as UTF-8 without BOM, using a
// here-string (@'...'@) so no escaping is needed.
// path must use Windows backslash separators.
func PsWriteFileUTF8(b *strings.Builder, path, content string) {
	dir := path[:strings.LastIndex(path, `\`)]
	PsEnsureDir(b, dir)
	b.WriteString(fmt.Sprintf(
		"[IO.File]::WriteAllText('%s', @'\n%s\n'@, (New-Object System.Text.UTF8Encoding $false))\n",
		path, content))
}

// PsEnsureDir appends a PowerShell idempotent mkdir statement.
func PsEnsureDir(b *strings.Builder, dir string) {
	b.WriteString(fmt.Sprintf(
		"if (!(Test-Path '%s')) { New-Item -ItemType Directory -Path '%s' -Force | Out-Null }\n",
		dir, dir))
}

// PsSearchExe appends PowerShell statements that search for exeName in
// C:\Users\*\.local\bin\ and %%APPDATA%%\npm, prepending matches to $env:PATH.
func PsSearchExe(b *strings.Builder, exeName string) {
	b.WriteString("foreach ($d in (Get-ChildItem 'C:\\Users' -Directory -ErrorAction SilentlyContinue)) {\n")
	b.WriteString(fmt.Sprintf(
		"  $p = Join-Path $d.FullName '.local\\bin'\n"+
			"  if (Test-Path (Join-Path $p '%s')) { $env:PATH = \"$p;$env:PATH\"; break }\n",
		exeName))
	b.WriteString("}\n")
	b.WriteString("if ($env:APPDATA) { $env:PATH = \"$env:APPDATA\\npm;$env:PATH\" }\n")
}

// PsQuoteArg wraps s in PowerShell single quotes, escaping embedded
// single quotes by doubling them. Use for command-line arguments only;
// here-string content must NOT be escaped — use PsWriteFileUTF8 instead.
func PsQuoteArg(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
}

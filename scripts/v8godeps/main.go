// Command v8godeps prepares the v8go native library for local builds/tests.
//
// v8go's Go module ships only deps/<GOOS>_<GOARCH>/libv8.a.zip (or v8.lib.zip);
// cgo needs the extracted library inside the module cache. This command:
//  1. ensures github.com/YaoApp/v8go is downloaded;
//  2. locates the module directory;
//  3. extracts libv8.a (or v8.lib on Windows) for the current platform.
//
// It is cross-platform (no unzip/make required) and is invoked by `make deps`.
package main

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const v8goModule = "github.com/YaoApp/v8go"

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "v8go deps: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	// Make sure the module is present in the module cache (best effort: if it
	// fails, `go list` below reports the actionable error).
	download := exec.Command("go", "mod", "download", v8goModule)
	download.Stdout = os.Stdout
	download.Stderr = os.Stderr
	_ = download.Run()

	dir, err := output("go", "list", "-m", "-f", "{{.Dir}}", v8goModule)
	if err != nil || dir == "" {
		return fmt.Errorf("cannot locate %s module (run: go mod download %s): %v", v8goModule, v8goModule, err)
	}

	plat := platformDir(runtime.GOOS, runtime.GOARCH)
	destDir := filepath.Join(dir, "deps", plat)
	libName := "libv8.a"
	zipName := "libv8.a.zip"
	if runtime.GOOS == "windows" {
		libName = "v8.lib"
		zipName = "v8.lib.zip"
	}
	libPath := filepath.Join(destDir, libName)
	zipPath := filepath.Join(destDir, zipName)

	if fileExists(libPath) {
		fmt.Printf("v8go native lib ready: %s\n", libPath)
		return nil
	}
	if !fileExists(zipPath) {
		// The committed archive is always named libv8.a.zip, even on Windows.
		zipPath = filepath.Join(destDir, "libv8.a.zip")
	}
	if !fileExists(zipPath) {
		return fmt.Errorf("missing %s (v8go module has no prebuilt lib for %s)", zipPath, plat)
	}

	if err := os.Chmod(destDir, 0o755); err != nil {
		// The Go module cache is read-only by default; best effort is enough on
		// Unix (owner can chmod). Report the real error if extraction fails.
		_ = err
	}
	if err := extractFromZip(zipPath, destDir, libName); err != nil {
		return err
	}
	if !fileExists(libPath) {
		return fmt.Errorf("extracted %s but %s not found", zipPath, libPath)
	}
	fmt.Printf("v8go native lib ready: %s\n", libPath)
	return nil
}

// platformDir maps GOOS/GOARCH to v8go's deps directory naming
// (amd64 is named x86_64, e.g. linux_amd64 -> linux_x86_64).
func platformDir(goos, goarch string) string {
	var arch string
	switch goarch {
	case "amd64":
		arch = "x86_64"
	case "arm64":
		arch = "arm64"
	default:
		arch = goarch
	}
	return goos + "_" + arch
}

func extractFromZip(zipPath, destDir, want string) error {
	zr, err := zip.OpenReader(zipPath)
	if err != nil {
		return fmt.Errorf("open %s: %w", zipPath, err)
	}
	defer zr.Close()

	for _, f := range zr.File {
		if filepath.Base(f.Name) != want {
			continue
		}
		dst := filepath.Join(destDir, filepath.Base(f.Name))
		rc, err := f.Open()
		if err != nil {
			return fmt.Errorf("open %s in zip: %w", f.Name, err)
		}
		out, err := os.Create(dst)
		if err != nil {
			rc.Close()
			return fmt.Errorf("create %s: %w", dst, err)
		}
		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return fmt.Errorf("extract %s: %w", dst, err)
		}
		_ = os.Chmod(dst, 0o644)
		return nil
	}
	return fmt.Errorf("%s not found in %s", want, zipPath)
}

func output(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

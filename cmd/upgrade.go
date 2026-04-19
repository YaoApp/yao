package cmd

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
	"github.com/yaoapp/yao/share"
)

const githubReleasesAPI = "https://api.github.com/repos/YaoApp/yao/releases/latest"
const cdnFallbackBase = "https://get.yaoapps.com/releases/yao"

// Upgrade command flags
var (
	upgradeYes    bool
	upgradeCheck  bool
	upgradeSource string
)

// githubRelease represents a GitHub release response
type githubRelease struct {
	TagName    string        `json:"tag_name"`
	Name       string        `json:"name"`
	Prerelease bool          `json:"prerelease"`
	Assets     []githubAsset `json:"assets"`
	HTMLURL    string        `json:"html_url"`
	Body       string        `json:"body"`
}

// githubAsset represents a single release asset
type githubAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
	Size               int64  `json:"size"`
}

// cdnLatest represents CDN latest.json format
//
//	{
//	  "version": "1.0.0",
//	  "released_at": "...",
//	  "assets": { "darwin-arm64": "https://...", ... }
//	}
type cdnLatest struct {
	Version    string            `json:"version"`
	ReleasedAt string            `json:"released_at"`
	Assets     map[string]string `json:"assets"`
}

// checkResult is the JSON payload emitted by `yao upgrade --check`
type checkResult struct {
	Current         string `json:"current"`
	Latest          string `json:"latest"`
	UpdateAvailable bool   `json:"update_available"`
	DownloadURL     string `json:"download_url,omitempty"`
	Source          string `json:"source"`
}

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: L("Upgrade yao to latest version"),
	Long:  L("Upgrade yao to latest version"),
	Run: func(cmd *cobra.Command, args []string) {
		latestVersion, downloadURL, err := resolveLatest()
		if err != nil {
			if upgradeCheck {
				emitCheckJSON(checkResult{
					Current: share.VERSION,
					Source:  upgradeSource,
				}, err)
				os.Exit(1)
			}
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		updateAvailable := compareVersions(latestVersion, share.VERSION) > 0

		if upgradeCheck {
			emitCheckJSON(checkResult{
				Current:         share.VERSION,
				Latest:          latestVersion,
				UpdateAvailable: updateAvailable,
				DownloadURL:     downloadURL,
				Source:          upgradeSource,
			}, nil)
			return
		}

		fmt.Printf("%s %s\n", color.WhiteString(L("Current version:")), color.CyanString(share.VERSION))
		fmt.Printf("%s %s\n", color.WhiteString(L("Latest version: ")), color.GreenString(latestVersion))

		if !updateAvailable {
			fmt.Println(color.GreenString(L("🎉Current version is the latest🎉")))
			return
		}

		if downloadURL == "" {
			fmt.Println(color.RedString(L("Fatal: %s"), fmt.Sprintf("asset not found for %s-%s", runtime.GOOS, runtime.GOARCH)))
			os.Exit(1)
		}

		if !upgradeYes {
			fmt.Printf("%s %s\n", color.WhiteString(L("Do you want to update to %s ? (y/n): "), latestVersion), "")
			fmt.Print("> ")
			input, err := bufio.NewReader(os.Stdin).ReadString('\n')
			if err != nil {
				fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
				os.Exit(1)
			}
			input = strings.TrimSpace(input)
			if input != "y" && input != "Y" {
				fmt.Println(color.YellowString(L("Canceled upgrade")))
				return
			}
		}

		exe, err := os.Executable()
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}
		exe, err = filepath.EvalSymlinks(exe)
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		fmt.Printf("%s %s\n", color.WhiteString(L("Downloading...")), color.CyanString(downloadURL))
		if err := downloadAndReplace(downloadURL, exe); err != nil {
			fmt.Println(color.RedString(L("Error occurred while updating binary: %s"), err.Error()))
			os.Exit(1)
		}

		fmt.Println(color.GreenString(L("🎉Successfully updated to version: %s🎉"), latestVersion))
	},
}

// resolveLatest fetches latest version info. Priority:
//  1. --source flag (explicit CDN URL)
//  2. GitHub Releases API (default)
//  3. Fallback to get.yaoapps.com CDN if GitHub fails (for users in China)
func resolveLatest() (string, string, error) {
	if upgradeSource != "" {
		return resolveFromCDN(upgradeSource)
	}
	ver, url, err := resolveFromGitHub()
	if err == nil {
		return ver, url, nil
	}
	fmt.Println(color.YellowString(L("GitHub unavailable (%s), trying CDN fallback..."), err.Error()))
	return resolveFromCDN(cdnFallbackBase)
}

// resolveFromCDN fetches latest.json from the given CDN base URL.
// The URL should point to the directory containing latest.json,
// e.g. "https://get.yaoapps.com/releases/yao".
func resolveFromCDN(base string) (string, string, error) {
	url := strings.TrimRight(base, "/") + "/latest.json"
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", "", err
	}
	req.Header.Set("User-Agent", fmt.Sprintf("yao/%s", share.VERSION))
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("fetch CDN latest.json failed: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("CDN returned status %d for %s", resp.StatusCode, url)
	}
	var data cdnLatest
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", "", fmt.Errorf("parse latest.json failed: %w", err)
	}
	key := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	dl := data.Assets[key]
	return strings.TrimPrefix(data.Version, "v"), dl, nil
}

// resolveFromGitHub keeps the original GitHub Releases API behavior.
func resolveFromGitHub() (string, string, error) {
	release, err := fetchLatestRelease()
	if err != nil {
		return "", "", err
	}
	latestVersion := strings.TrimPrefix(release.TagName, "v")
	assetName := buildAssetName(latestVersion)
	asset := findAsset(release.Assets, assetName)
	if asset == nil {
		return latestVersion, "", nil
	}
	return latestVersion, asset.BrowserDownloadURL, nil
}

// compareVersions returns >0 if a > b, 0 if equal, <0 if a < b.
// Uses dotted numeric comparison; falls back to string comparison for non-numeric parts.
func compareVersions(a, b string) int {
	a = strings.TrimPrefix(a, "v")
	b = strings.TrimPrefix(b, "v")
	if a == b {
		return 0
	}
	// Separate pre-release suffix (after '-')
	baseA, preA := splitVersion(a)
	baseB, preB := splitVersion(b)

	partsA := strings.Split(baseA, ".")
	partsB := strings.Split(baseB, ".")
	n := len(partsA)
	if len(partsB) > n {
		n = len(partsB)
	}
	for i := 0; i < n; i++ {
		var pa, pb string
		if i < len(partsA) {
			pa = partsA[i]
		}
		if i < len(partsB) {
			pb = partsB[i]
		}
		if c := compareNumeric(pa, pb); c != 0 {
			return c
		}
	}
	// Base parts equal: release version > pre-release version
	if preA == "" && preB != "" {
		return 1
	}
	if preA != "" && preB == "" {
		return -1
	}
	return strings.Compare(preA, preB)
}

func splitVersion(v string) (string, string) {
	if idx := strings.Index(v, "-"); idx >= 0 {
		return v[:idx], v[idx+1:]
	}
	return v, ""
}

func compareNumeric(a, b string) int {
	var na, nb int
	_, errA := fmt.Sscanf(a, "%d", &na)
	_, errB := fmt.Sscanf(b, "%d", &nb)
	if errA == nil && errB == nil {
		if na < nb {
			return -1
		}
		if na > nb {
			return 1
		}
		return 0
	}
	return strings.Compare(a, b)
}

// emitCheckJSON writes the check result as a single-line JSON to stdout.
func emitCheckJSON(r checkResult, err error) {
	if err != nil {
		payload := map[string]interface{}{
			"current": r.Current,
			"source":  r.Source,
			"error":   err.Error(),
		}
		b, _ := json.Marshal(payload)
		fmt.Println(string(b))
		return
	}
	b, _ := json.Marshal(r)
	fmt.Println(string(b))
}

// fetchLatestRelease fetches the latest release from GitHub API
func fetchLatestRelease() (*githubRelease, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	req, err := http.NewRequest("GET", githubReleasesAPI, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", fmt.Sprintf("yao/%s", share.VERSION))

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	var release githubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to parse release info: %w", err)
	}
	return &release, nil
}

// buildAssetName constructs the expected asset filename for the current platform
func buildAssetName(version string) string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// normalize arch names
	if goarch == "amd64" {
		goarch = "amd64"
	} else if goarch == "arm64" {
		goarch = "arm64"
	}

	return fmt.Sprintf("yao-%s-%s-%s", version, goos, goarch)
}

// findAsset finds the matching asset by name prefix
func findAsset(assets []githubAsset, name string) *githubAsset {
	for i, a := range assets {
		if a.Name == name {
			return &assets[i]
		}
	}
	return nil
}

// downloadAndReplace downloads the new binary and replaces the current executable
func downloadAndReplace(url, exePath string) error {
	client := &http.Client{Timeout: 10 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// write to a temp file in the same directory as the executable
	dir := filepath.Dir(exePath)
	tmpFile, err := os.CreateTemp(dir, ".yao-upgrade-*")
	if err != nil {
		return fmt.Errorf("failed to create temp file: %w", err)
	}
	tmpPath := tmpFile.Name()
	defer func() {
		tmpFile.Close()
		os.Remove(tmpPath)
	}()

	total := resp.ContentLength
	var downloaded int64
	buf := make([]byte, 32*1024)
	lastPrint := time.Now()

	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmpFile.Write(buf[:n]); werr != nil {
				return fmt.Errorf("write failed: %w", werr)
			}
			downloaded += int64(n)
			if time.Since(lastPrint) > 500*time.Millisecond || err == io.EOF {
				if total > 0 {
					pct := float64(downloaded) / float64(total) * 100
					fmt.Printf("\r  %s %.1f%%  (%d / %d MB)",
						color.CyanString(L("Progress:")),
						pct,
						downloaded/1024/1024,
						total/1024/1024,
					)
				} else {
					fmt.Printf("\r  %s %d MB downloaded", color.CyanString(L("Progress:")), downloaded/1024/1024)
				}
				lastPrint = time.Now()
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("download interrupted: %w", err)
		}
	}
	fmt.Println()

	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// make executable
	if err := os.Chmod(tmpPath, 0755); err != nil {
		return fmt.Errorf("failed to chmod: %w", err)
	}

	// atomically replace the executable
	if err := os.Rename(tmpPath, exePath); err != nil {
		// on some systems (cross-device) rename fails, fall back to copy
		if err2 := copyFile(tmpPath, exePath); err2 != nil {
			return fmt.Errorf("replace failed: %w (copy fallback: %v)", err, err2)
		}
	}

	return nil
}

// copyFile copies src to dst, used as fallback when rename fails cross-device
func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0755)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

func init() {
	upgradeCmd.Flags().BoolVarP(&upgradeYes, "yes", "y", false, L("Skip interactive confirmation"))
	upgradeCmd.Flags().BoolVar(&upgradeCheck, "check", false, L("Only check for updates and print JSON result"))
	upgradeCmd.Flags().StringVar(&upgradeSource, "source", "", L("Custom download source URL (e.g. https://get.yaoapps.com/releases/yao)"))
}

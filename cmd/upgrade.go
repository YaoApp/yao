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

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: L("Upgrade yao to latest version"),
	Long:  L("Upgrade yao to latest version"),
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("%s %s\n", color.WhiteString(L("Current version:")), color.CyanString(share.VERSION))
		fmt.Println(color.WhiteString(L("Checking latest version...")))

		release, err := fetchLatestRelease()
		if err != nil {
			fmt.Println(color.RedString(L("Fatal: %s"), err.Error()))
			os.Exit(1)
		}

		latestVersion := strings.TrimPrefix(release.TagName, "v")
		fmt.Printf("%s %s\n", color.WhiteString(L("Latest version: ")), color.GreenString(latestVersion))

		if latestVersion == share.VERSION {
			fmt.Println(color.GreenString(L("🎉Current version is the latest🎉")))
			os.Exit(0)
		}

		assetName := buildAssetName(latestVersion)
		asset := findAsset(release.Assets, assetName)
		if asset == nil {
			fmt.Println(color.RedString(L("Fatal: %s"), fmt.Sprintf("asset not found: %s", assetName)))
			fmt.Printf("%s %s\n", color.WhiteString(L("Available assets:")), "")
			for _, a := range release.Assets {
				if !strings.HasSuffix(a.Name, ".sha256") && !strings.HasSuffix(a.Name, ".zip") && !strings.HasSuffix(a.Name, ".tar.gz") {
					fmt.Printf("  - %s\n", color.YellowString(a.Name))
				}
			}
			os.Exit(1)
		}

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

		fmt.Printf("%s %s\n", color.WhiteString(L("Downloading...")), color.CyanString(asset.BrowserDownloadURL))
		if err := downloadAndReplace(asset.BrowserDownloadURL, exe); err != nil {
			fmt.Println(color.RedString(L("Error occurred while updating binary: %s"), err.Error()))
			os.Exit(1)
		}

		fmt.Println(color.GreenString(L("🎉Successfully updated to version: %s🎉"), latestVersion))
	},
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

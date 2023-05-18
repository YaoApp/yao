package get

import (
	"archive/zip"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/fs/system"
	"github.com/yaoapp/kun/log"
	"github.com/yaoapp/yao/widgets/app"
)

const (

	// Application application
	Application uint = iota

	// Widgets ? model & table & flow
	Widgets

	// Table table widget
	Table

	// Form form widget
	Form

	// Model model model
	Model

	// Flow data flow
	Flow
)

// Package package
type Package struct {
	Name   string
	Team   string
	Type   uint
	Remote string
	Origin string
	Temp   string
	Tag    string
	From   string
}

// New create a package via name
func New(repo string) (*Package, error) {

	team, name, tag, err := parse(repo)
	if err != nil {
		return nil, err
	}

	pkg := &Package{
		Origin: repo,
		Team:   team,
		Name:   name,
		Tag:    tag,
		Type:   Application,
	}

	url := pkg.InfraURL()
	if urlExists(url) {
		pkg.Remote = url
		pkg.From = "LetsInfra.com"
		return pkg, nil
	}

	// @Todo: Download from Github

	return nil, fmt.Errorf("%s not found", repo)
}

// InfraURL infra package url
func parse(repo string) (string, string, string, error) {

	tag := "latest"
	repo = strings.TrimSpace(repo)
	if !strings.Contains(repo, "/") {
		repo = fmt.Sprintf("yaoapp/%s", repo)
	}

	if strings.Contains(repo, "@") {
		arr := strings.Split(repo, "@")
		repo = arr[0]
		tag = arr[1]
	}

	arr := strings.Split(repo, "/")
	if len(arr) != 2 {
		return "", "", "", fmt.Errorf("REPO: %s format error", repo)
	}

	team := arr[0]
	name := arr[1]
	return team, name, tag, nil
}

// InfraURL infra package url
func (pkg *Package) InfraURL() string {
	return fmt.Sprintf("https://mirrors.yao.run/apps/%s/%s/%s.zip", pkg.Team, pkg.Name, pkg.Tag)
}

// GithubURL github package url
func (pkg *Package) GithubURL() string {
	return fmt.Sprintf("mirrors.letsinfra.com/apps/%s/%s/%s", pkg.Team, pkg.Name, pkg.Tag)
}

// urlExists check the http url is exists
func urlExists(url string) bool {
	resp, err := http.Get(url)
	if err != nil {
		return false
	}

	if resp.Body != nil {
		defer resp.Body.Close()
	}

	return resp.StatusCode == 200
}

// Download a package from remote
func (pkg *Package) Download() error {

	if pkg.Remote == "" {
		return fmt.Errorf("remote url is required")
	}

	root, err := os.MkdirTemp("", "*-yao-zip")
	if err != nil {
		return fmt.Errorf("Can't Create temp dir %s", err.Error())
	}

	name := fmt.Sprintf("%s-%s-%d.zip", pkg.Team, pkg.Name, time.Now().UnixMicro())
	file := filepath.Join(root, name)
	out, err := os.Create(file)
	defer out.Close()
	if err != nil {
		return fmt.Errorf("Can't Create file: %s", err.Error())
	}

	resp, err := http.Get(pkg.Remote)
	if err != nil {
		return fmt.Errorf("Download Error: %s", err.Error())
	}
	defer resp.Body.Close()

	_, err = io.Copy(out, resp.Body)
	if err != nil {
		return fmt.Errorf("Copy Error: %s", err.Error())
	}

	pkg.Temp = file
	return nil
}

// Validate a package files
func (pkg *Package) Validate() error {
	if pkg.Temp == "" {
		return fmt.Errorf("temp file not found")
	}
	return nil
}

// Unpack a package to current dir
func (pkg *Package) Unpack(dest string) (*app.DSL, error) {

	dest, err := filepath.Abs(dest)
	if err != nil {
		return nil, err
	}

	files, err := ioutil.ReadDir(dest)
	if err != nil {
		return nil, err
	}

	for _, f := range files {
		if !strings.HasPrefix(f.Name(), "logs") {
			return nil, fmt.Errorf("current folder shoud be empty")
		}
	}

	temp, err := os.MkdirTemp("", "*-yao-unzip")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(temp)

	// Read zip file
	r, err := zip.OpenReader(pkg.Temp)
	if err != nil {
		return nil, err
	}

	defer r.Close()
	defer os.Remove(pkg.Temp)

	path := ""
	for i, f := range r.File {
		if i == 0 {
			path = filepath.Join(temp, strings.TrimRight(f.Name, "/"))
		}
		err := extractFile(f, temp)
		if err != nil {
			return nil, err
		}
	}

	data, err := os.ReadFile(filepath.Join(path, "app.yao"))
	if err != nil {
		return nil, err
	}

	var setting app.DSL
	err = jsoniter.Unmarshal(data, &setting)
	if err != nil {
		return nil, err
	}

	fs := system.New("/")
	err = fs.Copy(path, dest)
	if err != nil {
		return nil, err
	}

	// Remove env
	fs.Remove(filepath.Join(dest, ".env"))
	return &setting, nil
}

// extractFile extract and save file to the dest path
func extractFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()

	path := filepath.Join(dest, f.Name)

	// Check for ZipSlip (Directory traversal)
	if !strings.HasPrefix(path, filepath.Clean(dest)+string(os.PathSeparator)) {
		return fmt.Errorf("illegal file path: %s", path)
	}

	if f.FileInfo().IsDir() {
		os.MkdirAll(path, f.Mode())
	} else {
		os.MkdirAll(filepath.Dir(path), f.Mode())
		f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			return err
		}
		defer func() {
			if err := f.Close(); err != nil {
				log.Error("repo unzip extractFile: %s", err.Error())
			}
		}()
		_, err = io.Copy(f, rc)
		if err != nil {
			return err
		}
	}
	return nil
}

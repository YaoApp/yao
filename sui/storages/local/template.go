package local

import (
	"fmt"
	"io"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/yaoapp/yao/sui/core"
	"golang.org/x/text/language"
)

// Assets get the assets treelist
func (tmpl *Template) Assets() []string {
	return nil
}

// Locales get the global locales
func (tmpl *Template) Locales() []core.SelectOption {

	supportLocales := []core.SelectOption{}
	path := filepath.Join(tmpl.Root, "__locales")
	if !tmpl.local.fs.IsDir(path) {
		return nil
	}

	dirs, err := tmpl.local.fs.ReadDir(path, false)
	if err != nil {
		return nil
	}

	for _, dir := range dirs {
		locale := filepath.Base(dir)
		label := language.Make(locale).String()
		supportLocales = append(supportLocales, core.SelectOption{
			Value: locale,
			Label: label,
		})
	}

	return supportLocales
}

// Themes get the global themes
func (tmpl *Template) Themes() []core.SelectOption {
	return tmpl.Template.Themes
}

// MediaSearch search the asset
func (tmpl *Template) MediaSearch(query url.Values, page int, pageSize int) (core.MediaSearchResult, error) {
	res := core.MediaSearchResult{Data: []core.Media{}, Page: page, PageSize: pageSize}

	total := 124
	pagecnt := int(math.Ceil(float64(total) / float64(pageSize)))
	for i := 0; i < pageSize; i++ {
		test := fmt.Sprintf("https://plus.unsplash.com/premium_photo-1671641797903-fd39ec702b16?auto=format&fit=crop&q=80&w=2334&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%%3D%%3D&id=%d", (page-1)*pageSize+i)
		thumb := fmt.Sprintf("https://plus.unsplash.com/premium_photo-1671641797903-fd39ec702b16?auto=format&fit=crop&q=80&w=100&ixlib=rb-4.0.3&ixid=M3wxMjA3fDB8MHxwaG90by1wYWdlfHx8fGVufDB8fHx8fA%%3D%%3D&id=%d", (page-1)*pageSize+i)
		res.Data = append(res.Data, core.Media{
			ID:     test,
			URL:    test,
			Thumb:  thumb,
			Type:   "image",
			Width:  100,
			Height: 100,
		})
	}

	res.Next = page + 1
	if (page+1)*pageSize >= total {
		res.Next = 0
	}

	res.Prev = page - 1
	if page == 1 {
		res.Prev = 0
	}

	res.Total = total
	res.PageCount = pagecnt

	return res, nil
}

// AssetUpload upload the asset
func (tmpl *Template) AssetUpload(reader io.Reader, name string) (string, error) {

	fingerprint := strings.ToUpper(uuid.NewString())
	dir := strings.Join([]string{string(os.PathSeparator), time.Now().Format("20060102")}, "")
	ext := filepath.Ext(name)
	file := filepath.Join(tmpl.Root, "__assets", "upload", dir, fmt.Sprintf("%s%s", fingerprint, ext))
	_, err := tmpl.local.fs.Write(file, reader, 0644)
	if err != nil {
		return "", err
	}
	return filepath.Join("upload", dir, fmt.Sprintf("%s%s", fingerprint, ext)), nil
}

// Asset get the asset
func (tmpl *Template) Asset(file string) (*core.Asset, error) {

	file = filepath.Join(tmpl.Root, "__assets", file)
	if exist, _ := tmpl.local.fs.Exists(file); exist {

		content, err := tmpl.local.fs.ReadFile(file)
		if err != nil {
			return nil, err
		}

		typ := "text/plain"
		switch filepath.Ext(file) {
		case ".css":
			typ = "text/css; charset=utf-8"
			break

		case ".js":
			typ = "application/javascript; charset=utf-8"
			break

		case ".ts":
			typ = "application/javascript; charset=utf-8"
			break

		case ".json":
			typ = "application/json; charset=utf-8"
			break

		case ".html":
			typ = "text/html; charset=utf-8"
			break

		default:
			typ, err = tmpl.local.fs.MimeType(file)
			if err != nil {
				return nil, err
			}
		}

		return &core.Asset{Type: typ, Content: content}, nil
	}

	return nil, fmt.Errorf("Asset %s not found", file)
}

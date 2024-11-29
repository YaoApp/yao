package core

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/PuerkitoBio/goquery"
	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/log"
)

// Get get the base info
func (page *Page) Get() *Page {
	return page
}

// SUI get the sui
func (page *Page) SUI() (SUI, error) {
	sui, has := SUIs[page.SuiID]
	if !has {
		return nil, fmt.Errorf("[sui] get page sui %s not found", page.SuiID)
	}
	return sui, nil
}

// Sid get the sid
func (page *Page) Sid() (string, error) {
	sui, err := page.SUI()
	if err != nil {
		return "", err
	}
	return sui.GetSid(), nil
}

// GetConfig get the config
func (page *Page) GetConfig() *PageConfig {

	if page.Codes.CONF.Code != "" {
		var config PageConfig
		err := jsoniter.UnmarshalFromString(page.Codes.CONF.Code, &config)
		if err == nil {
			page.Config = &config
		}
	}

	if page.Config == nil {
		page.Config = &PageConfig{
			Mock: &PageMock{Method: "GET"},
		}
	}

	if page.Config.Mock == nil {
		page.Config.Mock = &PageMock{Method: "GET"}
	}

	page.Config.Root = page.Root
	return page.Config
}

// ExportConfig export the config
func (page *Page) ExportConfig() string {
	if page.Config == nil {
		return fmt.Sprintf(`{"cacheStore": "%s"}`, page.CacheStore)
	}

	config, err := jsoniter.MarshalToString(map[string]interface{}{
		"title":      page.Config.Title,
		"guard":      page.Config.Guard,
		"cacheStore": page.CacheStore,
		"cache":      page.Config.Cache,
		"dataCache":  page.Config.DataCache,
		"api":        page.Config.API,
		"root":       page.Root,
	})

	if err != nil {
		log.Error("[sui] export page config error %s", err.Error())
		return ""
	}
	return config
}

// Data get the data （deprecated）
func (page *Page) Data(request *Request) (Data, map[string]interface{}, error) {

	setting := map[string]interface{}{
		"title": strings.ToUpper(page.Name),
	}

	if page.Codes.DATA.Code != "" {
		err := jsoniter.UnmarshalFromString(page.Codes.DATA.Code, &setting)
		if err != nil {
			return nil, nil, err
		}
	}
	return nil, setting, nil
}

// Exec get the data
func (page *Page) Exec(request *Request) (Data, error) {

	// Global data
	data := map[string]interface{}{}
	global := map[string]interface{}{}
	var err error
	if page.GlobalData != nil {
		global, err = request.ExecString(string(page.GlobalData))
		if err != nil {
			return nil, err
		}
	}

	if page.Codes.DATA.Code == "" {
		data["$global"] = global
		return data, nil
	}

	data, err = request.ExecString(page.Codes.DATA.Code)
	if err != nil {
		return nil, err
	}

	data["$global"] = global
	return data, nil
}

// RenderTitle render the title
func (page *Page) RenderTitle(data Data) string {

	if page.Config == nil {
		return "Untitled"
	}

	if page.Config.Title != "" {
		title, _ := data.Replace(page.Config.Title)
		return title
	}

	return "Untitled"
}

// Link get the link
func (page *Page) Link(r *Request) string {
	sui, has := SUIs[page.SuiID]
	if !has {
		log.Error("[sui] get page link %s not found", page.SuiID)
		return ""
	}

	root, err := sui.PublicRootWithSid(r.Sid)
	if err != nil {
		log.Error("[sui] get page link %s root error %s", page.SuiID, err.Error())
		return ""
	}

	parts := strings.Split(page.Route, "/")
	if len(parts) == 0 {
		log.Error("[sui] get page link %s path not found", page.SuiID)
		return ""
	}

	// Get the route
	paths := []string{root, "/"}
	for _, part := range parts {
		if part == "" {
			continue
		}
		if strings.HasPrefix(part, "[") && strings.HasSuffix(part, "]") {
			name := strings.TrimSuffix(strings.TrimPrefix(part, "["), "]")
			if name == "" {
				continue
			}

			if r == nil {
				continue
			}

			value, has := r.Params[name]
			if !has {
				paths = append(paths, name)
				continue
			}

			paths = append(paths, value)
			continue
		}
		paths = append(paths, part)
	}

	url := filepath.Join(paths...)
	if r.Query != nil {
		query := r.Query.Encode()
		if query != "" {
			url = url + "?" + query
		}
	}

	return url
}

// ReplaceDocument replace the document
func (page *Page) ReplaceDocument(doc *goquery.Document) {

	if page.Config == nil {
		return
	}

	if doc == nil {
		return
	}

	if page.Config.Title != "" {
		if doc.Find("title") != nil {
			doc.Find("title").SetText(page.Config.Title)
		}
	}

	if page.Config.Description != "" {
		if doc.Find("meta[name=description]") != nil {
			doc.Find("meta[name=description]").SetAttr("content", page.Config.Description)
		}
	}

	if page.Config.SEO != nil {

		if page.Config.SEO.Description != "" {
			if doc.Find("meta[name=description]") != nil {
				doc.Find("meta[name=description]").SetAttr("content", page.Config.SEO.Description)
			}
		}

		if page.Config.SEO.Keywords != "" {
			if doc.Find("meta[name=keywords]") != nil {
				sel := doc.Find("meta[name=keywords]")
				keywords := page.Config.SEO.Keywords
				if sel.AttrOr("content", "") != "" {
					keywords = keywords + "," + sel.AttrOr("content", "")
				}
				doc.Find("meta[name=keywords]").SetAttr("content", keywords)
			}
		}

		if page.Config.SEO.Title != "" {
			if doc.Find("meta[property='og:title']") != nil {
				doc.Find("meta[property='og:title']").SetAttr("content", page.Config.SEO.Title)
			}
		}

		if page.Config.SEO.Image != "" {
			if doc.Find("meta[property='og:image']") != nil {
				doc.Find("meta[property='og:image']").SetAttr("content", page.Config.SEO.Image)
			}
		}

		if page.Config.SEO.URL != "" {
			if doc.Find("meta[property='og:url']") != nil {
				doc.Find("meta[property='og:url']").SetAttr("content", page.Config.SEO.URL)
			}
		}
	}

}

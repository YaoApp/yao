package moapi

// *** WARNING ***
// Temporarily: change after the moapi is open source
//

import (
	"fmt"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/gou/http"
	"github.com/yaoapp/yao/config"
	"github.com/yaoapp/yao/share"
)

// Mirrors list all the mirrors
var cacheMirrors = []*Mirror{}
var cacheApps = []*App{}
var cacheMirrorsMap = map[string]*Mirror{}

// Models list all the models
var Models = []string{
	"gpt-4-1106-preview",
	"gpt-4-1106-vision-preview",
	"gpt-4",
	"gpt-4-32k",

	"gpt-3.5-turbo",
	"gpt-3.5-turbo-1106",
	"gpt-3.5-turbo-instruct",

	"dall-e-3",
	"dall-e-2",

	"tts-1",
	"tts-1-hd",

	"text-moderation-latest",
	"text-moderation-stable",

	"text-embedding-ada-002",
	"whisper-1",
}

// Load load the moapi
func Load(cfg config.Config) error {
	return registerAPI()
}

// Mirrors list all the mirrors
func Mirrors(cache bool) ([]*Mirror, error) {
	if cache && len(cacheMirrors) > 0 {
		return cacheMirrors, nil
	}

	bytes, err := httpGet("/api/moapi/mirrors")
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &cacheMirrors)
	if err != nil {
		return nil, err
	}

	for _, mirror := range cacheMirrors {
		cacheMirrorsMap[mirror.Host] = mirror
	}

	return cacheMirrors, nil
}

// Apps list all the apps
func Apps(cache bool) ([]*App, error) {
	if cache && len(cacheApps) > 0 {
		return cacheApps, nil
	}

	mirrors := SelectMirrors()
	bytes, err := httpGet("/api/moapi/apps", mirrors...)
	if err != nil {
		return nil, err
	}

	err = jsoniter.Unmarshal(bytes, &cacheApps)
	if err != nil {
		return nil, err
	}

	channel := Channel()
	if channel != "" {
		for i := range cacheApps {
			cacheApps[i].Homepage = cacheApps[i].Homepage + "?channel=" + channel
		}
	}

	return cacheApps, nil
}

// Homepage get the home page url with the invite code
func Homepage() string {
	channel := Channel()
	if channel == "" {
		return "https://store.moapi.ai"
	}
	return "https://store.moapi.ai" + "?channel=" + channel
}

// Channel get the channel
func Channel() string {

	return share.App.Moapi.Channel
}

// SelectMirrors select the mirrors
func SelectMirrors() []*Mirror {

	if share.App.Moapi.Mirrors == nil || len(share.App.Moapi.Mirrors) == 0 {
		return []*Mirror{}
	}

	_, err := Mirrors(true)
	if err != nil {
		return []*Mirror{}
	}

	// pick the mirrors
	var result []*Mirror
	for _, host := range share.App.Moapi.Mirrors {
		if mirror, ok := cacheMirrorsMap[host]; ok {
			if mirror.Status == "on" {
				result = append(result, mirror)
			}
		}
	}

	return result
}

// httpGet get the data from the api
func httpGet(api string, mirrors ...*Mirror) ([]byte, error) {
	return httpGetRetry(api, mirrors, 0)
}

func httpGetRetry(api string, mirrors []*Mirror, retryTimes int) ([]byte, error) {

	url := "https://" + share.MoapiHosts[retryTimes] + api
	if len(mirrors) > retryTimes {
		url = "https://" + mirrors[retryTimes].Host + api
	}

	secret := share.App.Moapi.Secret
	organization := share.App.Moapi.Organization

	http := http.New(url)
	http.SetHeader("Authorization", "Bearer "+secret)
	http.SetHeader("Content-Type", "application/json")
	http.SetHeader("Moapi-Organization", organization)

	resp := http.Get()
	if resp.Code >= 500 {
		if retryTimes > 3 {
			return nil, fmt.Errorf("Moapi Server Error: %s", resp.Data)
		}
		return httpGetRetry(api, mirrors, retryTimes+1)
	}

	return jsoniter.Marshal(resp.Data)
}

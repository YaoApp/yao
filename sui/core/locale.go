package core

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/yaoapp/gou/application"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/log"
	"gopkg.in/yaml.v3"
)

// Locales the locales
var Locales = map[string]map[string]*Locale{}

type localeData struct {
	name   string
	path   string
	locale *Locale
	cmd    uint8
}

var chLocale = make(chan *localeData, 1)

const (
	saveLocale uint8 = iota
	removeLocale
)

func init() {
	go localeWriter()
}

func localeWriter() {
	for {
		select {
		case data := <-chLocale:
			switch data.cmd {
			case saveLocale:
				if _, ok := Locales[data.name]; !ok {
					Locales[data.name] = map[string]*Locale{}
				}
				Locales[data.name][data.path] = data.locale

			case removeLocale:
				if _, ok := Locales[data.name]; ok {
					delete(Locales[data.name], data.path)
				}
			}
		}
	}
}

// Locale get the locale
func (parser *TemplateParser) Locale() *Locale {
	var locales map[string]*Locale = nil
	name, ok := parser.option.Locale.(string)
	if !ok {
		return nil
	}

	root := parser.option.Root
	route := parser.option.Route
	disableCache := parser.option.Preview || parser.option.Debug || parser.option.Editor || parser.option.DisableCache
	locales, ok = Locales[name]
	if !ok {
		locales = map[string]*Locale{}
	}

	locale, ok := locales[route]
	if ok && !disableCache {
		return locale
	}

	path := filepath.Join("public", parser.option.Root, ".locales", name, strings.TrimPrefix(route, root)+".yml")
	if exists, err := application.App.Exists(path); !exists {
		if err != nil {
			log.Error("[parser] %s Locale %s", route, err.Error())
		}
		return nil
	}

	// Load the locale
	locale = &Locale{Name: name}

	raw, err := application.App.Read(path)
	if err != nil {
		log.Error("[parser] %s Locale %s", route, err.Error())
		return nil
	}

	err = yaml.Unmarshal(raw, locale)
	if err != nil {
		log.Error("[parser] %s Locale %s", route, err.Error())
		return nil
	}

	if locale.Timezone == "" {
		locale.Timezone = GetSystemTimezone()
	}

	if locale.Direction == "" {
		locale.Direction = "ltr"
	}

	if parser.data != nil {
		parser.data["$timezone"] = locale.Timezone
		parser.data["$direction"] = locale.Direction
	}

	chLocale <- &localeData{name, route, locale, saveLocale}
	return locale
}

// MergeTranslations merge the translations
func (locale *Locale) MergeTranslations(translations []Translation, prefix ...string) {
	if locale.Keys == nil {
		locale.Keys = map[string]string{}
	}

	if locale.Messages == nil {
		locale.Messages = map[string]string{}
	}

	if locale.ScriptMessages == nil {
		locale.ScriptMessages = map[string]string{}
	}

	var reg *regexp.Regexp = nil
	if len(prefix) > 0 && prefix[0] != "" {
		reg = regexp.MustCompile(fmt.Sprintf(`^%s_([0-9]+)$`, prefix[0]))
	}

	for _, t := range translations {

		// Keep only the keys that start with the keyPrefix
		if reg != nil && !reg.MatchString(t.Key) {
			continue
		}

		message := t.Message
		if _, has := locale.Messages[message]; has {
			message = locale.Messages[message]
		}
		locale.Keys[t.Key] = message

		// Script messages
		if t.Type == "script" {
			locale.ScriptMessages[t.Message] = locale.Keys[t.Key]
			continue
		}

		msg, has := locale.Messages[t.Message]
		if has && msg != t.Message {
			continue
		}
		locale.Messages[t.Message] = t.Message
	}
}

// Merge merge the locale
func (locale *Locale) Merge(locale2 Locale) {

	if locale2.Keys != nil {
		if locale.Keys == nil {
			locale.Keys = map[string]string{}
		}
		for key, value := range locale2.Keys {
			if _, has := locale.Keys[key]; has {
				continue
			}
			locale.Keys[key] = value
		}
	}

	if locale2.Messages != nil {
		if locale.Messages == nil {
			locale.Messages = map[string]string{}
		}
		for key, value := range locale2.Messages {
			if _, has := locale.Messages[key]; has {
				continue
			}
			locale.Messages[key] = value
		}
	}
}

// ParseKeys match
func (locale *Locale) ParseKeys() {
	if locale.Keys == nil {
		locale.Keys = map[string]string{}
	}

	if locale.Messages == nil {
		locale.Messages = map[string]string{}
	}

	for key, msgKey := range locale.Keys {
		if message, has := locale.Messages[msgKey]; has {
			locale.Keys[key] = message
		}
	}
	return
}

// Fmt format the value
func (locale *Locale) Fmt(name string, value string) string {
	if locale.Formatter == "" {
		return value
	}

	pname := fmt.Sprintf("%s.%s", locale.Formatter, name)
	p, err := process.Of(pname, value, map[string]string{
		"name":      locale.Name,
		"timezone":  locale.Timezone,
		"direction": locale.Direction,
	})
	if err != nil {
		log.Error("[locale] %s %s", pname, err.Error())
		return value
	}

	res, err := p.Exec()
	if err != nil {
		log.Error("[locale] %s %s", pname, err.Error())
		return value
	}

	if v, ok := res.(string); ok {
		return v
	}

	log.Error("[locale] %s %s", pname, "The formatter must return a string")
	return value
}

// GetSystemTimezone get the system timezone
func GetSystemTimezone() string {
	now := time.Now()

	_, offset := now.Zone()

	hours := offset / 3600
	minutes := (offset % 3600) / 60

	sign := "+"
	if hours < 0 || minutes < 0 {
		sign = "-"
		hours = -hours
		minutes = -minutes
	}

	return fmt.Sprintf("%s%02d:%02d", sign, hours, minutes)
}

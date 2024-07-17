package core

import (
	"fmt"
	"regexp"
)

// MergeTranslations merge the translations
func (locale *Locale) MergeTranslations(translations []Translation, prefix ...string) {
	if locale.Keys == nil {
		locale.Keys = map[string]string{}
	}

	if locale.Messages == nil {
		locale.Messages = map[string]string{}
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

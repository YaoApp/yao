package str

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mozillazg/go-pinyin"
	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/maps"
)

// ProcessJoin utils.str.Join
func ProcessJoin(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	args := process.ArgsArray(0)
	sep := process.ArgsString(1)
	strs := []string{}
	for i := range args {
		strs = append(strs, fmt.Sprintf("%v", args[i]))
	}
	return strings.Join(strs, sep)
}

// ProcessJoinPath utils.str.JoinPath
func ProcessJoinPath(process *process.Process) interface{} {
	process.ValidateArgNums(2)
	paths := []string{}
	for _, arg := range process.Args {
		paths = append(paths, fmt.Sprintf("%v", arg))
	}
	return filepath.Join(paths...)
}

// ProcessUUID utils.str.uuid
func ProcessUUID(process *process.Process) interface{} {
	uuid := uuid.New()
	return uuid.String()
}

// ProcessPinyin utils.str.Pinyin converts Chinese characters to Pinyin
// Args:
//   - arg[0]: string, the Chinese characters to convert
//   - arg[1]: map (optional) configuration options
//     {
//     "tone": bool or string,  // true or "mark" for tone marks, "number" for numeric tones, false or "none" for no tones
//     "heteronym": bool, // whether to return multiple pronunciations for characters, default: false
//     "separator": string  // separator between pinyin, default: " "
//     }
func ProcessPinyin(process *process.Process) interface{} {
	process.ValidateArgNums(1)
	seed := process.ArgsString(0)

	// Create converter with settings
	a := pinyin.NewArgs()
	a.Style = pinyin.Normal // default style
	separator := " "        // default separator
	useToneNumber := false  // flag to track if we need to convert to tone numbers

	// Apply custom settings if provided
	if process.NumOfArgs() > 1 {
		confMap := process.ArgsMap(1, maps.MapStrAny{})

		// Check tone style
		toneVal, hasTone := confMap["tone"]
		if hasTone {
			// Handle different types of tone parameter
			switch v := toneVal.(type) {
			case bool:
				// Boolean: true = mark tones, false = no tones
				if v {
					a.Style = pinyin.Tone
				}
			case string:
				// String: "mark" = mark tones, "number" = numeric tones, "none" = no tones
				switch v {
				case "mark":
					a.Style = pinyin.Tone
				case "number":
					a.Style = pinyin.Tone2
					useToneNumber = true
				}
			}
		}

		// Check if heteronym is enabled
		heteronymVal, hasHeteronym := confMap["heteronym"]
		if hasHeteronym {
			if heteronym, ok := heteronymVal.(bool); ok && heteronym {
				a.Heteronym = true
			}
		}

		// Check custom separator
		if sep, ok := confMap["separator"].(string); ok {
			separator = sep
		}
	}

	// Convert to Pinyin
	result := pinyin.Pinyin(seed, a)

	// Fix the tone number position if needed (to handle "ha3o" -> "hao3")
	if useToneNumber {
		for i, pys := range result {
			for j, py := range pys {
				// This is a hacky fix to move the tone number to the end
				// since the library puts it after the vowel
				result[i][j] = fixToneNumberPosition(py)
			}
		}
	}

	// Process the pinyin results
	pinyinStr := make([]string, 0, len(result))
	for _, py := range result {
		if len(py) > 0 {
			if a.Heteronym && len(py) > 1 {
				// For heteronyms, use a pipe separator between different pronunciations
				charPinyins := strings.Join(py, "|")
				pinyinStr = append(pinyinStr, charPinyins)
			} else {
				pinyinStr = append(pinyinStr, py[0])
			}
		}
	}

	final := strings.Join(pinyinStr, separator)
	return final
}

// fixToneNumberPosition moves the tone number from after the vowel to the end of the syllable
func fixToneNumberPosition(s string) string {
	// Find the position of the first digit in the string
	for i, c := range s {
		if c >= '0' && c <= '9' {
			// If the digit is at the end already, return as is
			if i == len(s)-1 {
				return s
			}
			// Move the digit to the end
			return s[:i] + s[i+1:] + string(c)
		}
	}
	return s // No digit found, return as is
}

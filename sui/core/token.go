package core

import (
	"strings"
)

var propTokens = Tokens{
	{start: "{%", end: "%}"}, // {% xxx %}
	{start: "[{", end: "}]"}, // [{ xxx }]
}

var dataTokens = Tokens{
	{start: "{{", end: "}}"}, // {{ xxx }}
}

// Token the token
type Token struct {
	start string
	end   string
}

// Tokens the tokens
type Tokens []Token

// FindStringSubmatch returns a slice of strings holding the text of the
// leftmost match of the regular expression in s and the matches, if any, of
// its subexpressions, as defined by the 'Submatch' description in the
// package comment.
// A return value of nil indicates no match.
func (tokens Tokens) FindStringSubmatch(s string) []string {
	matches := []string{}
	for _, token := range tokens {
		startLen := len(token.start)
		endLen := len(token.end)
		stack := 0
		for i := 0; i <= len(s)-startLen; i++ {
			if s[i:i+startLen] == token.start {
				stack++
				if stack == 1 {
					for j := i + startLen; j <= len(s)-endLen; j++ {
						if s[j:j+endLen] == token.end {
							stack--
							if stack == 0 {
								matches = append(matches, s[i:j+endLen])
								i = j + endLen - 1
								break
							}
						} else if s[j:j+startLen] == token.start {
							stack++
						}
					}
				}
			}
		}
	}
	if len(matches) == 0 {
		return nil
	}
	return matches
}

// FindAllStringSubmatch is the 'All' version of FindStringSubmatch; it
// returns a slice of all successive matches of the expression, as defined by
// the 'All' description in the package comment.
// A return value of nil indicates no match.
func (tokens Tokens) FindAllStringSubmatch(s string, n int) [][]string {
	matches := [][]string{}
	for _, token := range tokens {
		startLen := len(token.start)
		endLen := len(token.end)
		stack := 0
		for i := 0; i <= len(s)-startLen; i++ {
			if s[i:i+startLen] == token.start {
				stack++
				if stack == 1 {
					for j := i + startLen; j <= len(s)-endLen; j++ {
						if s[j:j+endLen] == token.end {
							stack--
							if stack == 0 {
								matches = append(matches, []string{s[i : j+endLen], strings.TrimSpace(s[i+startLen : j])})
								i = j + endLen - 1
								break
							}
						} else if s[j:j+startLen] == token.start {
							stack++
						}
					}
				}
			}
		}
	}
	if len(matches) == 0 {
		return nil
	}
	return matches
}

// MatchString reports whether the string s
// contains any match of the regular expression re.
func (tokens Tokens) MatchString(s string) bool {
	for _, token := range tokens {
		startLen := len(token.start)
		endLen := len(token.end)
		stack := 0
		for i := 0; i <= len(s)-startLen; i++ {
			if s[i:i+startLen] == token.start {
				stack++
				if stack == 1 {
					for j := i + startLen; j <= len(s)-endLen; j++ {
						if s[j:j+endLen] == token.end {
							stack--
							if stack == 0 {
								return true
							}
						} else if s[j:j+startLen] == token.start {
							stack++
						}
					}
				}
			}
		}
	}
	return false
}

// ReplaceAllStringFunc returns a copy of src in which all matches of the
// Regexp have been replaced by the return value of function repl applied
// to the matched substring. The replacement returned by repl is substituted
// directly, without using Expand.
func (tokens Tokens) ReplaceAllStringFunc(src string, repl func(string) string) string {
	for _, token := range tokens {
		startLen := len(token.start)
		endLen := len(token.end)
		stack := 0
		i := 0
		for i <= len(src)-startLen {
			if i+startLen <= len(src) && src[i:i+startLen] == token.start {
				stack++
				if stack == 1 {
					for j := i + startLen; j <= len(src)-endLen; j++ {
						if j+endLen <= len(src) && src[j:j+endLen] == token.end {
							stack--
							if stack == 0 {
								match := src[i : j+endLen]
								replacement := repl(match)
								src = src[:i] + replacement + src[j+endLen:]
								i = i + len(replacement) - 1
								break
							}
						} else if j+startLen <= len(src) && src[j:j+startLen] == token.start {
							stack++
						}
					}
				}
			}
			i++
		}
	}
	return src
}

// ReplaceAllString returns a copy of src in which all matches of the
func (tokens Tokens) ReplaceAllString(src, repl string) string {
	return tokens.ReplaceAllStringFunc(src, func(string) string { return repl })
}

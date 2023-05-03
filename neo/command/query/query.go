package query

import (
	"regexp"
	"strings"
)

// Param the command param
type Param struct {
	Stack string `json:"stack,omitempty"`
	Path  string `json:"path,omitempty"`
}

// MatchStack match the stack
func (query Param) MatchStack(stack string) bool {

	if stack == "" || stack == "*" || query.Stack == "" {
		return true
	}

	if stack == query.Stack {
		return true
	}

	matched, _ := regexp.MatchString(strings.ReplaceAll(stack, "*", ".*"), query.Stack)
	return matched
}

// MatchPath match the path
func (query Param) MatchPath(path string) bool {
	if path == "" || path == "*" || query.Path == "" {
		return true
	}

	if path == query.Path {
		return true
	}

	matched, _ := regexp.MatchString(strings.ReplaceAll(path, "*", ".*"), query.Path)
	return matched
}

// MatchAny match the stack or path
func (query Param) MatchAny(stack, path string) bool {

	if path == "" || path == "-" {
		return query.MatchStack(stack)
	}

	if stack == "" || stack == "-" {
		return query.MatchPath(path)
	}

	return query.MatchStack(stack) || query.MatchPath(path)
}

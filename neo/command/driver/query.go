package driver

import (
	"regexp"
	"strings"
)

// MatchStack match the stack
func (query Query) MatchStack(stack string) bool {

	if query.Stack == "" || query.Stack == "*" || stack == "" {
		return true
	}

	if query.Stack == stack {
		return true
	}

	matched, _ := regexp.MatchString(strings.ReplaceAll(query.Stack, "*", ".*"), stack)
	return matched
}

// MatchPath match the path
func (query Query) MatchPath(path string) bool {
	if query.Path == "" || query.Path == "*" || path == "" {
		return true
	}

	if query.Path == path {
		return true
	}

	matched, _ := regexp.MatchString(strings.ReplaceAll(query.Path, "*", ".*"), path)
	return matched
}

// MatchAny match the stack or path
func (query Query) MatchAny(stack, path string) bool {
	if query.Path == "" || query.Path == "-" {
		return query.MatchStack(stack)
	}

	if query.Stack == "" || query.Stack == "-" {
		return query.MatchPath(path)
	}

	return query.MatchStack(stack) || query.MatchPath(path)
}

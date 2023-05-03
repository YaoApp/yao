package driver

import (
	"regexp"
	"strings"
)

// MatchStack match the stack
func (query Query) MatchStack(stack string) bool {

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
func (query Query) MatchPath(path string) bool {
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
func (query Query) MatchAny(stack, path string) bool {

	if path == "" || path == "-" {
		return query.MatchStack(stack)
	}

	if stack == "" || stack == "-" {
		return query.MatchPath(path)
	}

	return query.MatchStack(stack) || query.MatchPath(path)
}

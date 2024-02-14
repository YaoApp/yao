package pipe

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/expr-lang/expr"
	"github.com/expr-lang/expr/vm"
	"github.com/yaoapp/kun/log"
)

// If set the map value, should keep the space at the end of the statement
var stmtRe = regexp.MustCompile(`\{\{([\s\S]*?)\}\}`)
var options = []expr.Option{
	expr.AllowUndefinedVariables(),
}

// New create a new expression
func (data Data) New(stmt string) (*vm.Program, error) {

	stmt = stmtRe.ReplaceAllStringFunc(stmt, func(stmt string) string {
		matches := stmtRe.FindStringSubmatch(stmt)
		if len(matches) > 0 {
			stmt = strings.ReplaceAll(stmt, matches[0], matches[1])
		}
		return stmt
	})

	stmt = strings.TrimSpace(stmt)
	// &#39; => ' &#34; => "
	stmt = strings.ReplaceAll(stmt, "&#39;", "'")
	stmt = strings.ReplaceAll(stmt, "&#34;", "\"")
	return expr.Compile(stmt, append([]expr.Option{expr.Env(data)}, options...)...)
}

// Exec exec statement for the template
func (data Data) Exec(stmt string) (interface{}, error) {
	program, err := data.New(stmt)
	if err != nil {
		log.Warn("pipe: %s %s", stmt, err)
		return nil, nil
	}

	v, err := expr.Run(program, data)
	if err != nil {
		log.Warn("pipe: %s %s", stmt, err)
		return nil, nil
	}
	return v, nil
}

// ExecString exec statement for the template
func (data Data) ExecString(stmt string) (string, error) {

	res, err := data.Exec(stmt)
	if err != nil {
		return "", nil
	}

	if res == nil {
		return "", nil
	}

	if v, ok := res.(string); ok {
		return v, nil
	}
	return fmt.Sprintf("%v", res), nil
}

// IsExpression check if the statement is an expression
func IsExpression(stmt string) bool {
	return stmtRe.MatchString(stmt)
}

func (data Data) replace(value any) (any, error) {

	switch v := value.(type) {
	case string:
		return data.replaceAny(v)

	case []any:
		return data.replaceArray(v)

	case map[string]any:
		return data.replaceMap(v)

	case Input:
		return data.replaceArray(v)
	}

	return value, nil
}

func (data Data) replacePrompts(prompts []Prompt) ([]Prompt, error) {
	newPrompts := []Prompt{}
	for _, prompt := range prompts {
		content, err := data.replaceString(prompt.Content)
		if err != nil {
			return nil, err
		}
		role, err := data.replaceString(prompt.Role)
		if err != nil {
			return nil, err
		}
		prompt.Role = role
		prompt.Content = content
		newPrompts = append(newPrompts, prompt)
	}
	return newPrompts, nil
}

func (data Data) replaceAny(value string) (any, error) {

	if !IsExpression(value) {
		return value, nil
	}

	v, err := data.Exec(value)
	if err != nil {
		return "", err
	}
	return v, nil
}

// replaceString replace the string
func (data Data) replaceString(value string) (string, error) {

	if !IsExpression(value) {
		return value, nil
	}

	v, err := data.ExecString(value)
	if err != nil {
		return "", err
	}
	return v, nil
}

func (data Data) replaceMap(value map[string]any) (map[string]any, error) {
	newValue := map[string]any{}
	if value == nil {
		return newValue, nil
	}

	for k, v := range value {
		res, err := data.replace(v)
		if err != nil {
			return nil, err
		}
		newValue[k] = res
	}
	return newValue, nil
}

func (data Data) replaceArray(value []any) ([]any, error) {
	newValue := []any{}
	if value == nil {
		return newValue, nil
	}

	for _, v := range value {
		res, err := data.replace(v)
		if err != nil {
			return nil, err
		}
		newValue = append(newValue, res)
	}

	return newValue, nil
}

func (data Data) replaceInput(value Input) (Input, error) {
	return data.replaceArray(value)
}

func anyToInput(v any) Input {
	switch v := v.(type) {
	case Input:
		return v

	case []any:
		return v

	case []string:
		input := Input{}
		for _, s := range v {
			input = append(input, s)
		}
		return input

	default:
		return Input{v}
	}
}

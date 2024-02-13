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

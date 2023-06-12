package helper

import (
	"fmt"
	"regexp"
	"strings"

	jsoniter "github.com/json-iterator/go"
	"github.com/yaoapp/kun/any"
)

// ComputeFunc 计算函数
type ComputeFunc func(interface{}, interface{}) bool

// Computes 可用计算式
var Computes = map[string]ComputeFunc{
	"=": func(left interface{}, right interface{}) bool {
		return left == right
	},
	">": func(left interface{}, right interface{}) bool {
		return any.Of(left).CFloat64() > any.Of(right).CFloat64()
	},
	">=": func(left interface{}, right interface{}) bool {
		return any.Of(left).CFloat64() >= any.Of(right).CFloat64()
	},
	"<": func(left interface{}, right interface{}) bool {
		return any.Of(left).CFloat64() < any.Of(right).CFloat64()
	},
	"<=": func(left interface{}, right interface{}) bool {
		return any.Of(left).CFloat64() <= any.Of(right).CFloat64()
	},
	"!=": func(left interface{}, right interface{}) bool {
		return left != right
	},
	"hasprefix": func(left interface{}, right interface{}) bool {
		return strings.HasPrefix(fmt.Sprintf("%v", left), fmt.Sprintf("%v", right))
	},
	"hassuffix": func(left interface{}, right interface{}) bool {
		return strings.HasSuffix(fmt.Sprintf("%v", left), fmt.Sprintf("%v", right))
	},
	"contains": func(left interface{}, right interface{}) bool {
		return strings.Contains(fmt.Sprintf("%v", left), fmt.Sprintf("%v", right))
	},
	"match": func(left interface{}, right interface{}) bool {
		re := regexp.MustCompile(fmt.Sprintf("%v", right))
		return re.Match([]byte(fmt.Sprintf("%v", left)))
	},
	"is": func(left interface{}, right interface{}) bool {
		if is, ok := right.(string); ok {
			is = strings.ToLower(is)
			if is == "null" {
				return left == nil
			} else if is == "notnull" {
				return left != nil
			}
		}
		return false
	},
}

// Condition 判断条件
type Condition struct {
	Left    interface{} `json:"left"`
	Right   interface{} `json:"right"`
	Compute ComputeFunc `json:"-"`
	OP      string      `json:"op"`
	OR      bool        `json:"or"`
	Comment string      `json:"comment"`
}

// When 多项条件判断
func When(conds []Condition) bool {
	res := true
	for _, cond := range conds {
		if cond.OR {
			res = res || cond.Exec()
			continue
		}
		res = res && cond.Exec()
	}
	return res
}

// Exec 执行条件判断
func (cond Condition) Exec() bool {
	return cond.Compute(cond.Left, cond.Right)
}

// UnmarshalJSON for json marshalJSON
func (cond *Condition) UnmarshalJSON(data []byte) error {
	origin := map[string]interface{}{}
	err := jsoniter.Unmarshal(data, &origin)
	if err != nil {
		return err
	}
	*cond = ConditionOf(origin)
	return nil
}

// MarshalJSON for json marshalJSON
func (cond Condition) MarshalJSON() ([]byte, error) {
	return jsoniter.Marshal(cond.ToMap())
}

// ConditionOf 从 map[string]interface{}
func ConditionOf(input map[string]interface{}) Condition {
	cond := Condition{}
	for k, val := range input {
		key := strings.ToLower(k)
		// { "=": "foo" }
		if compute, has := Computes[key]; has {
			cond.Right = val
			cond.Compute = compute
			cond.OP = k
			continue
		}

		switch key {
		case "left":
			cond.Left = val
			continue
		case "right":
			cond.Right = val
			continue
		case "op":
			if val, ok := val.(string); ok {
				if compute, has := Computes[val]; has {
					cond.Compute = compute
					cond.OP = val
				}

			}
			continue
		case "or":
			if val, ok := val.(bool); ok {
				cond.OR = val
			}
			continue
		case "comment":
			if val, ok := val.(string); ok {
				cond.Comment = val
			}
			continue
		}

		// { "用户不存在": "bar"},
		cond.Comment = key
		cond.Left = val
	}

	return cond
}

// ToMap Condition 转换为 map[string]interface{}
func (cond Condition) ToMap() map[string]interface{} {
	res := map[string]interface{}{}
	if cond.OP != "" {
		res[cond.OP] = cond.Right
	}
	if cond.Comment != "" {
		res[cond.Comment] = cond.Left
	} else {
		res["left"] = cond.Left
		res["right"] = cond.Right
	}
	if cond.OR {
		res["or"] = true
	}
	return res
}

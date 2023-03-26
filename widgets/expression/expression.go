package expression

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/yaoapp/gou/process"
	"github.com/yaoapp/kun/any"
	"github.com/yaoapp/kun/maps"
)

// [ *]([0-9a-zA-Z_\-\.])[ *]
var regVar, _ = regexp.Compile(`([\\]*)\$[\.]*([\.0-9a-zA-Z_\-]*)\{[ ]*([0-9a-zA-Z_,\-\.\|\', ]+)[ ]*\}`)
var regNum, _ = regexp.Compile(`[0-9\.]+`)

// Export processes
func Export() error {
	exportProcess()
	return nil
}

// Replace with the given data
// ${label || comment}
// please input ${label || comment}
// where.${name}.eq
// ${name}
// $.SelectOption{option}
func Replace(ptr interface{}, data map[string]interface{}) error {
	if data == nil {
		return nil
	}

	ptrRef := reflect.ValueOf(ptr)
	if ptrRef.Kind() != reflect.Pointer {
		return fmt.Errorf("the value is %s, should be a pointer", ptrRef.Kind().String())
	}

	ref := ptrRef.Elem()
	kind := ref.Kind()
	data = any.Of(data).MapStr()

	switch kind {
	case reflect.String:
		new, replaced := replace(ref.String(), data)
		if _, ok := new.(string); replaced && ok {
			ptrRef.Elem().Set(reflect.ValueOf(new))
		}
		break

	case reflect.Map:
		keys := ref.MapKeys()
		for _, key := range keys {
			val := ref.MapIndex(key).Interface()
			Replace(&val, data)

			ref.SetMapIndex(key, reflect.ValueOf(val))
		}
		ptrRef.Elem().Set(ref)
		break

	case reflect.Slice:
		values := []interface{}{}
		for i := 0; i < ref.Len(); i++ {
			val := ref.Index(i).Interface()
			Replace(&val, data)
			values = append(values, val)
		}
		ptrRef.Elem().Set(reflect.ValueOf(values))
		break

	case reflect.Struct:
		for i := 0; i < ref.NumField(); i++ {
			if ref.Field(i).CanSet() {
				val := ref.Field(i).Interface()
				Replace(&val, data)
				ref.Field(i).Set(reflect.ValueOf(val).Convert(ref.Field(i).Type()))
			}
		}
		ptrRef.Elem().Set(ref)
		break

	case reflect.Interface:
		elmRef := ref.Elem()
		elmKind := elmRef.Kind()
		switch elmKind {
		case reflect.String:
			new, replaced := replace(ref.Elem().String(), data)
			if replaced {
				ptrRef.Elem().Set(reflect.ValueOf(new))
			}
			break

		case reflect.Map:
			keys := elmRef.MapKeys()
			for _, key := range keys {
				val := elmRef.MapIndex(key).Interface()
				Replace(&val, data)
				elmRef.SetMapIndex(key, reflect.ValueOf(val))
			}
			ptrRef.Elem().Set(elmRef)
			break

		case reflect.Slice:
			values := []interface{}{}
			for i := 0; i < elmRef.Len(); i++ {
				val := elmRef.Index(i).Interface()
				Replace(&val, data)
				values = append(values, val)
			}
			ptrRef.Elem().Set(reflect.ValueOf(values))
			break
		}

		break

	}

	return nil
}

func replace(value string, data maps.MapStrAny) (interface{}, bool) {
	matches := regVar.FindAllStringSubmatch(value, -1)
	length := len(matches)
	if length == 0 {
		return value, false
	}

	// "${ name }"
	if length == 1 && strings.TrimSpace(value) == strings.TrimSpace(matches[0][0]) {

		if matches[0][1] != "" {
			return value, false
		}

		if matches[0][2] != "" {
			// computeOf( "SelectOption", []string{"option"}, data )
			return computeOf(matches[0][2], strings.Split(matches[0][3], ","), data)
		}

		return valueOf(strings.TrimSpace(matches[0][3]), data) // valueOf( "name", data )
	}

	replaced := false
	// "${ name } ${ label || comment || 'value' || 0.618 } and ${label} \${name}"
	for _, match := range matches {
		if match[1] != "" {
			continue
		}

		if match[2] != "" {
			// computeOf( "SelectOption", []string{"option"}, data )
			if v, ok := computeOf(match[2], strings.Split(match[3], ","), data); ok {
				value = strings.ReplaceAll(value, strings.TrimSpace(match[0]), fmt.Sprintf("%v", v))
				replaced = true
			}
		}

		// valueOf( "name", data )
		if v, ok := valueOf(strings.TrimSpace(match[3]), data); ok {
			value = strings.ReplaceAll(value, strings.TrimSpace(match[0]), fmt.Sprintf("%v", v))
			replaced = true
		}
	}

	return value, replaced
}

func computeOf(processName string, argsvars []string, data maps.MapStrAny) (interface{}, bool) {
	args := []interface{}{}
	for _, name := range argsvars {
		arg, _ := valueOf(strings.TrimSpace(name), data)
		args = append(args, arg)
	}

	if !strings.Contains(processName, ".") {
		processName = fmt.Sprintf("yao.expression.%s", processName)
	}

	p, err := process.Of(processName, args...)
	if err != nil {
		return err.Error(), true
	}

	res, err := p.Exec()
	if err != nil {
		return err.Error(), true
	}

	return res, true
}

func valueOf(name string, data maps.MapStrAny) (interface{}, bool) {

	// label || comment || 'value' || 0.618 || 1
	if strings.Contains(name, "||") {
		names := strings.Split(name, "||")
		for _, name := range names {
			name := strings.TrimSpace(name)
			value, replaced := valueOf(name, data)
			if replaced {
				return value, true
			}
		}
	}

	//  'value'
	if strings.HasPrefix(name, "'") && strings.HasSuffix(name, "'") {
		return strings.Trim(name, "'"), true
	}

	//  0.618 / 1
	if regNum.MatchString(name) {
		return name, true
	}

	// label / comment
	if data.Has(name) {
		value := data.Get(name)
		if valstr, ok := value.(string); ok {
			//  ::value
			if strings.HasPrefix(valstr, "::") {
				return fmt.Sprintf("$L(%s)", strings.TrimPrefix(valstr, "::")), true
			}
		}
		return value, true
	}

	return nil, false
}

/*
 * Copyright 2019-present Ciena Corporation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package filter

import (
	"fmt"
	"reflect"
	"regexp"
	"strings"
)

type Operation int

const (
	UK Operation = iota
	EQ
	NE
	GT
	LT
	GE
	LE
	RE
)

func toOp(op string) Operation {
	switch op {
	case "=":
		return EQ
	case "!=":
		return NE
	case ">":
		return GT
	case "<":
		return LT
	case ">=":
		return GE
	case "<=":
		return LE
	case "~":
		return RE
	default:
		return UK
	}
}

type FilterTerm struct {
	Op    Operation
	Value string
	re    *regexp.Regexp
}

type Filter map[string]FilterTerm

var termRE = regexp.MustCompile(`^\s*([a-zA-Z_][.a-zA-Z0-9_]*)\s*(~|<=|>=|<|>|!=|=)\s*(.+)\s*$`)

// Parse parses a comma separated list of filter terms
func Parse(spec string) (Filter, error) {
	filter := make(map[string]FilterTerm)
	terms := strings.Split(spec, ",")
	var err error

	// Each term is in the form <key><op><value>
	for _, term := range terms {
		parts := termRE.FindAllStringSubmatch(term, -1)
		if parts == nil {
			return nil, fmt.Errorf("Unable to parse filter term '%s'", term)
		}
		ft := FilterTerm{
			Op:    toOp(parts[0][2]),
			Value: parts[0][3],
		}
		if ft.Op == RE {
			ft.re, err = regexp.Compile(ft.Value)
			if err != nil {
				return nil, fmt.Errorf("Unable to parse regexp filter value '%s'", ft.Value)
			}
		}
		filter[parts[0][1]] = ft
	}
	return filter, nil
}

func (f Filter) Process(data interface{}) (interface{}, error) {
	slice := reflect.ValueOf(data)
	if slice.Kind() != reflect.Slice {
		match, err := f.Evaluate(data)
		if err != nil {
			return nil, err
		}
		if match {
			return data, nil
		}
		return nil, nil
	}

	var result []interface{}

	for i := 0; i < slice.Len(); i++ {
		match, err := f.Evaluate(slice.Index(i).Interface())
		if err != nil {
			return nil, err
		}
		if match {
			result = append(result, slice.Index(i).Interface())
		}
	}

	return result, nil
}

// returns False if the filter does not match
// returns true if the filter does match or the operation is unsupported
func testField(v FilterTerm, field reflect.Value) bool {
	switch v.Op {
	case RE:
		if !v.re.MatchString(fmt.Sprintf("%v", field)) {
			return false
		}
	case EQ:
		// This seems to work for most comparisons
		if fmt.Sprintf("%v", field) != v.Value {
			return false
		}
	case NE:
		// This seems to work for most comparisons
		if fmt.Sprintf("%v", field) == v.Value {
			return false
		}
	default:
		// For unsupported operations, always pass
	}

	return true
}

func (f Filter) EvaluateTerm(k string, v FilterTerm, val reflect.Value, recurse bool) (bool, error) {
	// If we have been given a pointer, then deference it
	if val.Kind() == reflect.Ptr {
		val = reflect.Indirect(val)
	}

	// If the user gave us an explicitly named dotted field, then split it
	if strings.Contains(k, ".") {
		parts := strings.SplitN(k, ".", 2)
		if val.Kind() != reflect.Struct {
			return false, fmt.Errorf("Dotted field name specified in filter did not resolve to a valid field")
		}
		field := val.FieldByName(parts[0])
		if !field.IsValid() {
			return false, fmt.Errorf("Failed to find dotted field %s while filtering", parts[0])
		}
		return f.EvaluateTerm(parts[1], v, field, false)
	}

	if val.Kind() != reflect.Struct {
		return false, fmt.Errorf("Field name specified in filter did not resolve to a valid field")
	}

	field := val.FieldByName(k)
	if !field.IsValid() {
		return false, fmt.Errorf("Failed to find field %s while filtering", k)
	}

	// we might have a pointer to a struct at this time, so dereference it
	if field.Kind() == reflect.Ptr {
		field = reflect.Indirect(field)
	}

	if field.Kind() == reflect.Struct {
		return false, fmt.Errorf("Cannot filter on a field that is a struct")
	}

	if (field.Kind() == reflect.Slice) || (field.Kind() == reflect.Array) {
		// For an array, check to see if any item matches
		someMatch := false
		for i := 0; i < field.Len(); i++ {
			arrayElem := field.Index(i)
			if testField(v, arrayElem) {
				someMatch = true
			}
		}
		if !someMatch {
			//if recurse && val.Kind() == reflect.Struct {
			//    TODO: implement automatic recursion when the user did not
			//          use a dotted notation. Go through the list of fields
			//          in the struct, recursively check each one.
			//}
			return false, nil
		}
	} else {
		if !testField(v, field) {
			return false, nil
		}
	}

	return true, nil
}

func (f Filter) Evaluate(item interface{}) (bool, error) {
	val := reflect.ValueOf(item)

	for k, v := range f {
		matches, err := f.EvaluateTerm(k, v, val, true)
		if err != nil {
			return false, err
		}
		if !matches {
			// If any of the filter fail, the overall match fails
			return false, nil
		}
	}

	return true, nil
}

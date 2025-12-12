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
package order

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
)

type Operation int

const (
	ASC Operation = iota
	DSC
)

type SortTerm struct {
	Op   Operation
	Name string
}

func (o Operation) String() string {
	switch o {
	default:
		fallthrough
	case ASC:
		return "ASC"
	case DSC:
		return "DSC"
	}
}

type Sorter []SortTerm

func split(term string) SortTerm {
	st := SortTerm{}
	if len(term) > 0 {
		switch term[0] {
		case '+':
			fallthrough
		case '>':
			st.Op = ASC
			st.Name = term[1:]
		case '-':
			fallthrough
		case '<':
			st.Op = DSC
			st.Name = term[1:]
		default:
			st.Op = ASC
			st.Name = term
		}
	} else {
		st.Op = ASC
		st.Name = term
	}
	return st
}

// Parse parses a comma separated list of filter terms
func Parse(spec string) (Sorter, error) {
	terms := strings.Split(spec, ",")
	s := make([]SortTerm, 0)
	for _, term := range terms {
		s = append(s, split(term))
	}

	return s, nil
}

func (s Sorter) GetField(val reflect.Value, name string) (reflect.Value, error) {
	// If the user gave us an explicitly named dotted field, then split it
	if strings.Contains(name, ".") {
		parts := strings.SplitN(name, ".", 2)

		if val.Kind() != reflect.Struct {
			return val, fmt.Errorf("Dotted field name specified in filter did not resolve to a valid field")
		}

		field := val.FieldByName(parts[0])
		if !field.IsValid() {
			return field, fmt.Errorf("Failed to find dotted field %s while sorting", parts[0])
		}
		if field.Kind() == reflect.Ptr {
			field = reflect.Indirect(field)
		}
		return s.GetField(field, parts[1])
	}

	if val.Kind() != reflect.Struct {
		return val, fmt.Errorf("Dotted field name specified in filter did not resolve to a valid field")
	}

	field := val.FieldByName(name)
	if !field.IsValid() {
		return field, fmt.Errorf("Failed to find field %s while sorting", name)
	}

	// we might have a pointer to a struct at this time, so dereference it
	if field.Kind() == reflect.Ptr {
		field = reflect.Indirect(field)
	}

	if field.Kind() == reflect.Struct {
		return val, fmt.Errorf("Cannot sort on a field that is a struct")
	}

	return field, nil
}

func (s Sorter) Process(data interface{}) (interface{}, error) {
	slice := reflect.ValueOf(data)
	if slice.Kind() != reflect.Slice {
		return data, nil
	}

	var sortError error = nil

	sort.SliceStable(data, func(i, j int) bool {
		left := reflect.ValueOf(slice.Index(i).Interface())
		right := reflect.ValueOf(slice.Index(j).Interface())

		if left.Kind() == reflect.Ptr {
			left = reflect.Indirect(left)
		}

		if right.Kind() == reflect.Ptr {
			right = reflect.Indirect(right)
		}

		for _, term := range s {
			fleft, err := s.GetField(left, term.Name)
			if err != nil {
				sortError = err
				return false
			}

			fright, err := s.GetField(right, term.Name)
			if err != nil {
				sortError = err
				return false
			}

			switch fleft.Kind() {
			case reflect.Uint:
				fallthrough
			case reflect.Uint8:
				fallthrough
			case reflect.Uint16:
				fallthrough
			case reflect.Uint32:
				fallthrough
			case reflect.Uint64:
				ileft := fleft.Uint()
				iright := fright.Uint()
				switch term.Op {
				case ASC:
					if ileft < iright {
						return true
					} else if ileft > iright {
						return false
					}
				case DSC:
					if ileft > iright {
						return true
					} else if ileft < iright {
						return false
					}
				}
			case reflect.Int:
				fallthrough
			case reflect.Int8:
				fallthrough
			case reflect.Int16:
				fallthrough
			case reflect.Int32:
				fallthrough
			case reflect.Int64:
				ileft := fleft.Int()
				iright := fright.Int()
				switch term.Op {
				case ASC:
					if ileft < iright {
						return true
					} else if ileft > iright {
						return false
					}
				case DSC:
					if ileft > iright {
						return true
					} else if ileft < iright {
						return false
					}
				}
			default:
				sleft := fmt.Sprintf("%v", fleft)
				sright := fmt.Sprintf("%v", fright)
				diff := strings.Compare(sleft, sright)
				if term.Op != DSC {
					if diff == -1 {
						return true
					} else if diff == 1 {
						return false
					}
				} else {
					if diff == 1 {
						return true
					} else if diff == -1 {
						return false
					}
				}
			}
		}
		return false
	})

	if sortError != nil {
		return nil, sortError
	}

	return data, nil
}

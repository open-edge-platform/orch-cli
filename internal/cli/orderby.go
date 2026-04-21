// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"reflect"
	"sort"
	"strings"
	"sync"
	"unicode"
)

type orderByProbeFunc func(orderBy string) (bool, error)

var orderBySupportCache = struct {
	mu     sync.Mutex
	fields map[string][]string
}{fields: map[string][]string{}}

func camelToSnake(value string) string {
	if value == "" {
		return ""
	}

	out := make([]rune, 0, len(value)+4)
	for i, r := range value {
		if unicode.IsUpper(r) {
			if i > 0 {
				out = append(out, '_')
			}
			out = append(out, unicode.ToLower(r))
			continue
		}
		out = append(out, r)
	}

	return string(out)
}

func buildOrderByAliases(sample any) (map[string]string, []string) {
	t := reflect.TypeOf(sample)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	aliases := make(map[string]string)
	canonical := make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}

		name := strings.Split(tag, ",")[0]
		if name == "" || name == "-" {
			continue
		}

		if _, exists := aliases[name]; !exists {
			canonical = append(canonical, name)
		}

		aliases[name] = name
		aliases[strings.ToLower(name)] = name
		aliases[camelToSnake(name)] = name
		aliases[f.Name] = name
		aliases[strings.ToLower(f.Name)] = name
		aliases[camelToSnake(f.Name)] = name
	}

	sort.Strings(canonical)
	return aliases, canonical
}

func getSupportedOrderByFields(resourceKey string, sample any, probe orderByProbeFunc) ([]string, map[string]struct{}, error) {
	orderBySupportCache.mu.Lock()
	if cached, ok := orderBySupportCache.fields[resourceKey]; ok {
		orderBySupportCache.mu.Unlock()
		set := make(map[string]struct{}, len(cached))
		for _, field := range cached {
			set[field] = struct{}{}
		}
		return cached, set, nil
	}
	orderBySupportCache.mu.Unlock()

	_, canonical := buildOrderByAliases(sample)
	supported := make([]string, 0, len(canonical))
	for _, field := range canonical {
		accepted, err := probe(field)
		if err != nil {
			return nil, nil, err
		}
		if accepted {
			supported = append(supported, field)
		}
	}

	orderBySupportCache.mu.Lock()
	orderBySupportCache.fields[resourceKey] = supported
	orderBySupportCache.mu.Unlock()

	set := make(map[string]struct{}, len(supported))
	for _, field := range supported {
		set[field] = struct{}{}
	}
	return supported, set, nil
}

func normalizeOrderByWithAPIProbe(raw string, resourceKey string, sample any, probe orderByProbeFunc) (*string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	aliases, canonical := buildOrderByAliases(sample)

	terms := strings.Split(raw, ",")
	normalized := make([]string, 0, len(terms))
	apiFields := make([]string, 0, len(terms))

	for _, rawTerm := range terms {
		term := strings.TrimSpace(rawTerm)
		if term == "" {
			continue
		}

		field := term
		direction := ""
		switch term[0] {
		case '+':
			direction = "asc"
			field = strings.TrimSpace(term[1:])
		case '-':
			direction = "desc"
			field = strings.TrimSpace(term[1:])
		case '>', '<':
			return nil, fmt.Errorf("invalid --order-by term %q for API sorting: use plain field names or +/- prefixes", term)
		}

		if len(strings.Fields(field)) != 1 {
			return nil, fmt.Errorf("invalid --order-by term %q; use <field>, +<field>, or -<field>", term)
		}

		if field == "" {
			return nil, fmt.Errorf("invalid --order-by term %q; available fields: %s", term, strings.Join(canonical, ", "))
		}

		apiField, ok := aliases[field]
		if !ok {
			// Field not in model at all — fetch supported fields for accurate hints.
			supported, _, err := getSupportedOrderByFields(resourceKey, sample, probe)
			if err != nil {
				return nil, err
			}
			hintFields := supported
			if len(hintFields) == 0 {
				hintFields = canonical
			}
			return nil, fmt.Errorf("invalid --order-by field %q; available fields: %s (note: not all fields may support API-side sorting for JSON/YAML output)", field, strings.Join(hintFields, ", "))
		}

		apiFields = append(apiFields, apiField)

		if direction != "" {
			normalized = append(normalized, apiField+" "+direction)
		} else {
			normalized = append(normalized, apiField)
		}
	}

	if len(normalized) == 0 {
		return nil, nil
	}

	normalizedOrderBy := strings.Join(normalized, ",")
	accepted, err := probe(normalizedOrderBy)
	if err != nil {
		return nil, err
	}
	if accepted {
		return &normalizedOrderBy, nil
	}

	// Probe failed: build/consult cache only now to provide precise hints.
	supported, supportedSet, err := getSupportedOrderByFields(resourceKey, sample, probe)
	if err != nil {
		return nil, err
	}

	hintFields := supported
	if len(hintFields) == 0 {
		hintFields = canonical
	}

	for _, apiField := range apiFields {
		if _, ok := supportedSet[apiField]; !ok {
			return nil, fmt.Errorf("invalid --order-by field %q; available fields: %s (note: not all fields may support API-side sorting for JSON/YAML output)", apiField, strings.Join(hintFields, ", "))
		}
	}

	return nil, fmt.Errorf("invalid --order-by expression %q; available fields: %s (note: not all fields may support API-side sorting for JSON/YAML output)", raw, strings.Join(hintFields, ", "))
}

// buildClientSortAliases builds an alias map for client-side sorting.
// Unlike buildOrderByAliases (which maps to JSON tag names for the API), this maps all
// aliases to the Go struct field name (e.g. "Kind") because order.go uses reflect.FieldByName
// which requires the exact PascalCase struct field name. Hints are shown using JSON tag names.
func buildClientSortAliases(sample any) (aliases map[string]string, jsonHints []string) {
	t := reflect.TypeOf(sample)
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}

	aliases = make(map[string]string)
	jsonHints = make([]string, 0)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		tag := f.Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		jsonName := strings.Split(tag, ",")[0]
		if jsonName == "" || jsonName == "-" {
			continue
		}
		structName := f.Name // e.g. "DisplayName" — required by reflect.FieldByName in order.go

		jsonHints = append(jsonHints, jsonName)

		// Map all common variants to the struct field name
		aliases[jsonName] = structName
		aliases[strings.ToLower(jsonName)] = structName
		aliases[camelToSnake(jsonName)] = structName
		aliases[structName] = structName
		aliases[strings.ToLower(structName)] = structName
		aliases[camelToSnake(structName)] = structName
	}

	sort.Strings(jsonHints)
	return aliases, jsonHints
}

// normalizeOrderByForClientSorting validates order-by fields exist in the model without API probing.
// This is used for client-side sorting (table format) where the sorting is done locally on the fetched data.
// Returns a normalized order-by string using Go struct field names, as required by order.go.
func normalizeOrderByForClientSorting(raw string, sample any) (*string, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}

	aliases, hints := buildClientSortAliases(sample)

	terms := strings.Split(raw, ",")
	normalized := make([]string, 0, len(terms))

	for _, rawTerm := range terms {
		term := strings.TrimSpace(rawTerm)
		if term == "" {
			continue
		}

		prefix := ""
		field := term
		switch term[0] {
		case '+', '-', '>', '<':
			prefix = term[:1]
			field = strings.TrimSpace(term[1:])
		}

		if field == "" {
			return nil, fmt.Errorf("invalid --order-by term %q; available fields: %s", term, strings.Join(hints, ", "))
		}

		structField, ok := aliases[field]
		if !ok {
			return nil, fmt.Errorf("invalid --order-by field %q; available fields: %s", field, strings.Join(hints, ", "))
		}

		normalized = append(normalized, prefix+structField)
	}

	if len(normalized) == 0 {
		return nil, nil
	}

	normalizedOrderBy := strings.Join(normalized, ",")
	return &normalizedOrderBy, nil
}

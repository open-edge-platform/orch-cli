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
package format

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"text/tabwriter"
	"text/template"
	"text/template/parse"
)

var nameFinder = regexp.MustCompile(`\.([\._A-Za-z0-9]*)}}`)

type Format string

/* TrimAndPad
 *
 * Modify `s` so that it is exactly `l` characters long, removing
 * characters from the end, or adding spaces as necessary.
 */

func TrimAndPad(s string, l int) string {
	// TODO: support right justification if a negative number is passed
	if len(s) > l {
		s = s[:l]
	}
	return s + strings.Repeat(" ", l-len(s))
}

func CamelCaseToSpaces(s string) string {
	var result strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			result.WriteRune(' ')
		}
		result.WriteRune(r)
	}
	return result.String()
}

func headerLabelForField(field string) string {
	if field == "ApplicationReferences" {
		return "APPLICATION COUNT"
	}

	// Special case for DeployId to match legacy header
	if field == "DeployId" {
		return "DEPLOYMENT ID"
	}

	// Special case for ID field (avoid "I D" splitting)
	if field == "ID" {
		return "ID"
	}

	// Handle nested fields (e.g., "Status.State") by processing each part
	parts := strings.Split(field, ".")
	var processedParts []string
	for _, part := range parts {
		// Convert camelCase to spaces
		spaced := CamelCaseToSpaces(part)

		// Special handling for common ID patterns to avoid "I D" in headers
		// Replace " Id" at the end with " ID"
		if strings.HasSuffix(spaced, " Id") {
			spaced = strings.TrimSuffix(spaced, " Id") + " ID"
		}

		processedParts = append(processedParts, strings.ToUpper(spaced))
	}

	// Join with space instead of dot for better readability
	return strings.Join(processedParts, " ")
}

/* GetHeaderString
 *
 * From a template, extract the set of column names.
 */

func GetHeaderString(tmpl *template.Template, nameLimit int) string {
	var header string
	for _, n := range tmpl.Tree.Root.Nodes {
		switch n.Type() {
		case parse.NodeText:
			header += n.String()
		case parse.NodeString:
			header += n.String()
		case parse.NodeAction:
			found := nameFinder.FindStringSubmatch(n.String())
			if len(found) == 2 {
				if nameLimit > 0 {
					parts := strings.Split(found[1], ".")
					start := len(parts) - nameLimit
					if start < 0 {
						start = 0
					}
					header += headerLabelForField(strings.Join(parts[start:], "."))
				} else {
					header += headerLabelForField(found[1])
				}
			}
		}
	}
	return header
}

func (f Format) IsTable() bool {
	return strings.HasPrefix(string(f), "table")
}

func (f Format) Execute(writer io.Writer, withHeaders bool, nameLimit int, data interface{}) error {
	var tabWriter *tabwriter.Writer = nil
	format := f

	if f.IsTable() {
		if existingTabWriter, ok := writer.(*tabwriter.Writer); ok {
			tabWriter = existingTabWriter
		} else {
			tabWriter = tabwriter.NewWriter(writer, 0, 4, 4, ' ', 0)
		}
		format = Format(strings.TrimPrefix(string(f), "table"))
	}

	funcmap := template.FuncMap{
		"timestamp":       formatTimestamp,
		"since":           formatSince,
		"gosince":         formatGoSince,
		"deref":           formatDeref,
		"str":             formatString,
		"none":            formatStringOrNone,
		"fmttime":         formatTimeSimple,
		"formatTime":      formatTime,
		"statusIndicator": formatStatusIndicator,
		"statusMessage":   formatStatusMessage,
		"nodeCount":       formatNodeCount,
	}

	tmpl, err := template.New("output").Funcs(funcmap).Parse(string(format))
	if err != nil {
		return err
	}

	if f.IsTable() && withHeaders {
		header := GetHeaderString(tmpl, nameLimit)

		if _, err = tabWriter.Write([]byte(header)); err != nil {
			return err
		}
		if _, err = tabWriter.Write([]byte("\n")); err != nil {
			return err
		}

		slice := reflect.ValueOf(data)
		if slice.Kind() == reflect.Slice {
			for i := 0; i < slice.Len(); i++ {
				if err = tmpl.Execute(tabWriter, slice.Index(i).Interface()); err != nil {
					return err
				}
				if _, err = tabWriter.Write([]byte("\n")); err != nil {
					return err
				}
			}
		} else {
			if err = tmpl.Execute(tabWriter, data); err != nil {
				return err
			}
			if _, err = tabWriter.Write([]byte("\n")); err != nil {
				return err
			}
		}
		tabWriter.Flush()
		return nil
	}

	slice := reflect.ValueOf(data)
	if slice.Kind() == reflect.Slice {
		for i := 0; i < slice.Len(); i++ {
			if err = tmpl.Execute(writer, slice.Index(i).Interface()); err != nil {
				return err
			}
			if _, err = writer.Write([]byte("\n")); err != nil {
				return err
			}
		}
	} else {
		if err = tmpl.Execute(writer, data); err != nil {
			return err
		}
		if _, err = writer.Write([]byte("\n")); err != nil {
			return err
		}
	}
	return nil

}

// HeaderFields returns the list of header field names extracted from the
// provided format. It splits the header string on tab characters and trims
// whitespace. Returns an error if the template fails to parse.
func (f Format) HeaderFields(nameLimit int) ([]string, error) {
	funcmap := template.FuncMap{
		"timestamp":       formatTimestamp,
		"since":           formatSince,
		"gosince":         formatGoSince,
		"deref":           formatDeref,
		"str":             formatString,
		"none":            formatStringOrNone,
		"fmttime":         formatTimeSimple,
		"formatTime":      formatTime,
		"statusIndicator": formatStatusIndicator,
		"statusMessage":   formatStatusMessage,
		"nodeCount":       formatNodeCount,
	}

	// Trim table prefix so header text doesn't include the literal "table"
	formatStr := string(f)
	if strings.HasPrefix(formatStr, "table") {
		formatStr = strings.TrimPrefix(formatStr, "table")
	}

	// Parse the template to access its parse tree
	tmpl, err := template.New("output").Funcs(funcmap).Parse(formatStr)
	if err != nil {
		return nil, err
	}

	// Walk the template parse tree and extract raw field names (e.g., Name, DisplayName)
	var rawFields []string
	for _, n := range tmpl.Tree.Root.Nodes {
		switch n.Type() {
		case parse.NodeAction:
			found := nameFinder.FindStringSubmatch(n.String())
			if len(found) == 2 {
				rawFields = append(rawFields, found[1])
			}
		}
	}

	// Normalize to user-friendly aliases: lowercase and snake_case
	seen := make(map[string]struct{})
	var fields []string
	for _, rf := range rawFields {
		// respect nameLimit (take last N parts if dotted)
		parts := strings.Split(rf, ".")
		if nameLimit > 0 && len(parts) > nameLimit {
			parts = parts[len(parts)-nameLimit:]
		}
		field := strings.Join(parts, ".")

		// lowercase no-spaces alias
		lower := strings.ToLower(strings.ReplaceAll(field, " ", ""))
		if _, ok := seen[lower]; !ok {
			fields = append(fields, lower)
			seen[lower] = struct{}{}
		}

		// snake_case alias
		// simple camel->snake conversion
		snake := camelToSnake(field)
		if _, ok := seen[snake]; !ok {
			fields = append(fields, snake)
			seen[snake] = struct{}{}
		}
	}

	return fields, nil
}

// camelToSnake converts CamelCase/PascalCase to snake_case
func camelToSnake(s string) string {
	var b strings.Builder
	for i, r := range s {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteRune('_')
		}
		b.WriteRune(r)
	}
	return strings.ToLower(b.String())
}

/*
 * ExecuteFixedWidth
 *
 * Formats a table row using a set of fixed column widths. Used for streaming
 * output where column widths cannot be automatically determined because only
 * one line of the output is available at a time.
 *
 * Assumes the format uses tab as a field delimiter.
 *
 * columnWidths: struct that contains column widths
 * header: If true return the header. If false then evaluate data and return data.
 * data: Data to evaluate
 */

func (f Format) ExecuteFixedWidth(columnWidths interface{}, header bool, data interface{}) (string, error) {
	if !f.IsTable() {
		return "", errors.New("Fixed width is only available on table format")
	}

	outputAs := strings.TrimPrefix(string(f), "table")
	tmpl, err := template.New("output").Parse(string(outputAs))
	if err != nil {
		return "", fmt.Errorf("Failed to parse template: %v", err)
	}

	var buf bytes.Buffer
	var tabSepOutput string

	if header {
		// Caller wants the table header.
		tabSepOutput = GetHeaderString(tmpl, 1)
	} else {
		// Caller wants the data.
		err = tmpl.Execute(&buf, data)
		if err != nil {
			return "", fmt.Errorf("Failed to execute template: %v", err)
		}
		tabSepOutput = buf.String()
	}

	// Extract the column width constants by running the template on the
	// columnWidth structure. This will cause text.template to split the
	// column widths exactly like it did the output (i.e. separated by
	// tab characters)
	buf.Reset()
	err = tmpl.Execute(&buf, columnWidths)
	if err != nil {
		return "", fmt.Errorf("Failed to execute template on widths: %v", err)
	}
	tabSepWidth := buf.String()

	// Loop through the fields and widths, printing each field to the
	// preset width.
	output := ""
	outParts := strings.Split(tabSepOutput, "\t")
	widthParts := strings.Split(tabSepWidth, "\t")
	for i, outPart := range outParts {
		width, err := strconv.Atoi(widthParts[i])
		if err != nil {
			return "", fmt.Errorf("Failed to parse width %s: %v", widthParts[i], err)
		}
		output = output + TrimAndPad(outPart, width) + " "
	}

	// remove any trailing spaces
	output = strings.TrimRight(output, " ")

	return output, nil
}

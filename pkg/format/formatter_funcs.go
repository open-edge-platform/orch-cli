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
	"fmt"
	"reflect"
	"time"

	timestamppb "github.com/golang/protobuf/ptypes/timestamp"
)

// formats a Timestamp proto as a RFC3339 date string
func formatTimestamp(tsproto *timestamppb.Timestamp) (string, error) {
	if tsproto == nil {
		return "", nil
	}
	return tsproto.AsTime().Truncate(time.Second).Format(time.RFC3339), nil
}

// Computes the age of a timestamp and returns it in HMS format
func formatGoSince(ts time.Time) (string, error) {
	return time.Since(ts).Truncate(time.Second).String(), nil
}

// Computes the age of a timestamp and returns it in HMS format
func formatSince(tsproto *timestamppb.Timestamp) (string, error) {
	if tsproto == nil {
		return "", nil
	}
	return time.Since(tsproto.AsTime()).Truncate(time.Second).String(), nil
}

// Dereferences pointers recursively for safer template rendering.
// If a nil pointer is encountered, returns the zero value of the pointed-to type.
func formatDeref(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	rv := reflect.ValueOf(v)
	for rv.Kind() == reflect.Ptr {
		if rv.IsNil() {
			return reflect.Zero(rv.Type().Elem()).Interface()
		}
		rv = rv.Elem()
	}

	return rv.Interface()
}

// Renders a string pointer safely for templates.
func formatString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

// Renders a string pointer as "<none>" if nil or empty.
func formatStringOrNone(v *string) string {
	if v == nil || *v == "" {
		return "<none>"
	}
	return *v
}

// Formats a time.Time using ISO-8601 format without timezone.
func formatTimeSimple(t time.Time) string {
	return t.Format("2006-01-02T15:04:05")
}

// Extracts status indicator from GenericStatus-like objects.
// Returns a short indicator string (✓, ⨯, ?, ⏳) based on the indicator field.
func formatStatusIndicator(status interface{}) string {
	if status == nil {
		return "?"
	}

	// Use reflection to access Indicator field
	v := reflect.ValueOf(status)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "?"
		}
		v = v.Elem()
	}

	indicatorField := v.FieldByName("Indicator")
	if !indicatorField.IsValid() {
		return "?"
	}

	// Dereference if pointer
	if indicatorField.Kind() == reflect.Ptr {
		if indicatorField.IsNil() {
			return "?"
		}
		indicatorField = indicatorField.Elem()
	}

	// Get the string value of the indicator
	indicator := fmt.Sprintf("%v", indicatorField.Interface())

	switch indicator {
	case "STATUS_INDICATION_IDLE":
		return "✓"
	case "STATUS_INDICATION_ERROR":
		return "⨯"
	case "STATUS_INDICATION_IN_PROGRESS":
		return "⏳"
	case "STATUS_INDICATION_UNSPECIFIED":
		return "-"
	default:
		return "?"
	}
}

// Extracts status message from GenericStatus-like objects.
// Returns the message string or "<unknown>" if not available.
func formatStatusMessage(status interface{}) string {
	if status == nil {
		return "<unknown>"
	}

	// Use reflection to access Message field
	v := reflect.ValueOf(status)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "<unknown>"
		}
		v = v.Elem()
	}

	messageField := v.FieldByName("Message")
	if !messageField.IsValid() {
		return "<unknown>"
	}

	// Dereference if pointer
	if messageField.Kind() == reflect.Ptr {
		if messageField.IsNil() {
			return "<unknown>"
		}
		messageField = messageField.Elem()
	}

	message := fmt.Sprintf("%v", messageField.Interface())
	if message == "" {
		return "<unknown>"
	}
	return message
}

// Formats a node count, returning the count or "-" if nil.
func formatNodeCount(count *int) string {
	if count == nil {
		return "-"
	}
	return fmt.Sprintf("%d", *count)
}

// formatTime accepts various timestamp representations (unix int seconds, time.Time,
// protobuf Timestamp) and returns an ISO-like string. Returns empty string for nil.
func formatTime(v interface{}) string {
	if v == nil {
		return ""
	}

	switch t := v.(type) {
	case *int:
		if t == nil {
			return ""
		}
		return time.Unix(int64(*t), 0).UTC().Format(time.RFC3339)
	case int:
		return time.Unix(int64(t), 0).UTC().Format(time.RFC3339)
	case *time.Time:
		if t == nil {
			return ""
		}
		return t.UTC().Format("2006-01-02T15:04:05")
	case time.Time:
		return t.UTC().Format("2006-01-02T15:04:05")
	case *timestamppb.Timestamp:
		if t == nil {
			return ""
		}
		return t.AsTime().Truncate(time.Second).Format(time.RFC3339)
	default:
		return fmt.Sprint(v)
	}
}

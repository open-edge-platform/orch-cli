/*
 * Copyright 2019-2024 Open Networking Foundation (ONF) and the ONF Contributors
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
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestFilterIncludedStruct struct {
	Six string
}

type TestFilterStruct struct {
	One   string
	Two   string
	Three string
	Five  TestFilterIncludedStruct
	Seven *TestFilterIncludedStruct
}

func TestFilterList(t *testing.T) {
	f, err := Parse("One=a,Two=b,Five.Six=d,Seven.Six=e")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := []interface{}{
		TestFilterStruct{
			One:   "a",
			Two:   "b",
			Three: "c",
			Five:  TestFilterIncludedStruct{Six: "d"},
			Seven: &TestFilterIncludedStruct{Six: "e"},
		},
		TestFilterStruct{
			One:   "1",
			Two:   "2",
			Three: "3",
			Five:  TestFilterIncludedStruct{Six: "4"},
			Seven: &TestFilterIncludedStruct{Six: "5"},
		},
		TestFilterStruct{
			One:   "a",
			Two:   "b",
			Three: "z",
			Five:  TestFilterIncludedStruct{Six: "d"},
			Seven: &TestFilterIncludedStruct{Six: "e"},
		},
	}

	r, _ := f.Process(data)

	if _, ok := r.([]interface{}); !ok {
		t.Errorf("Expected list, but didn't get one")
	}

	if len(r.([]interface{})) != 2 {
		t.Errorf("Expected %d got %d", 2, len(r.([]interface{})))
	}

	if r.([]interface{})[0] != data[0] {
		t.Errorf("Filtered list did not match, item %d", 0)
	}
	if r.([]interface{})[1] != data[2] {
		t.Errorf("Filtered list did not match, item %d", 1)
	}
}

func TestFilterItem(t *testing.T) {
	f, err := Parse("One=a,Two=b")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
	}

	r, _ := f.Process(data)

	if r == nil {
		t.Errorf("Expected item, got nil")
	}

	if _, ok := r.([]interface{}); ok {
		t.Errorf("Expected item, but got list")
	}
}

func TestGoodFilters(t *testing.T) {
	var f Filter
	var err error
	f, err = Parse("One=a,Two=b")
	if err != nil {
		t.Errorf("1. Unable to parse filter: %s", err.Error())
	}
	if len(f) != 2 ||
		f["One"].Value != "a" ||
		f["One"].Op != EQ ||
		f["Two"].Value != "b" ||
		f["Two"].Op != EQ {
		t.Errorf("1. Filter did not parse correctly")
	}

	f, err = Parse("One=a")
	if err != nil {
		t.Errorf("2. Unable to parse filter: %s", err.Error())
	}
	if len(f) != 1 ||
		f["One"].Value != "a" ||
		f["One"].Op != EQ {
		t.Errorf("2. Filter did not parse correctly")
	}

	f, err = Parse("One<a")
	if err != nil {
		t.Errorf("3. Unable to parse filter: %s", err.Error())
	}
	if len(f) != 1 ||
		f["One"].Value != "a" ||
		f["One"].Op != LT {
		t.Errorf("3. Filter did not parse correctly")
	}

	f, err = Parse("One!=a")
	if err != nil {
		t.Errorf("4. Unable to parse filter: %s", err.Error())
	}
	if len(f) != 1 ||
		f["One"].Value != "a" ||
		f["One"].Op != NE {
		t.Errorf("4. Filter did not parse correctly")
	}
}

func TestBadFilters(t *testing.T) {
	_, err := Parse("One%a")
	if err == nil {
		t.Errorf("Parsed filter when it shouldn't have")
	}
}

func TestSingleRecord(t *testing.T) {
	f, err := Parse("One=d")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
	}

	r, err := f.Process(data)
	if err != nil {
		t.Errorf("Error processing data")
	}

	if r != nil {
		t.Errorf("expected no results, got some")
	}
}

// Invalid fields will throw an exception.
func TestInvalidField(t *testing.T) {
	f, err := Parse("Four=a")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
	}

	r, err := f.Process(data)
	assert.EqualError(t, err, "failed to find field Four while filtering")

	if r != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestInvalidDotted(t *testing.T) {
	f, err := Parse("Five.NonExistent=a")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
		Five:  TestFilterIncludedStruct{Six: "w"},
	}

	r, err := f.Process(data)
	assert.EqualError(t, err, "failed to find field NonExistent while filtering")

	if r != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestTrailingDot(t *testing.T) {
	f, err := Parse("Five.Six.=a")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
		Five:  TestFilterIncludedStruct{Six: "w"},
	}

	r, err := f.Process(data)
	assert.EqualError(t, err, "field name specified in filter did not resolve to a valid field")

	if r != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestDottedOnString(t *testing.T) {
	f, err := Parse("One.IsNotAStruct=a")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
		Five:  TestFilterIncludedStruct{Six: "w"},
	}

	r, err := f.Process(data)
	assert.EqualError(t, err, "field name specified in filter did not resolve to a valid field")

	if r != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestFilterOnStruct(t *testing.T) {
	f, err := Parse("Five=a")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
		Five:  TestFilterIncludedStruct{Six: "w"},
	}

	r, err := f.Process(data)
	assert.EqualError(t, err, "cannot filter on a field that is a struct")

	if r != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestFilterOnPointerStruct(t *testing.T) {
	f, err := Parse("Seven=a")
	if err != nil {
		t.Errorf("Unable to parse filter: %s", err.Error())
	}

	data := TestFilterStruct{
		One:   "a",
		Two:   "b",
		Three: "c",
		Seven: &TestFilterIncludedStruct{Six: "w"},
	}

	r, err := f.Process(data)
	assert.EqualError(t, err, "cannot filter on a field that is a struct")

	if r != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestREFilter(t *testing.T) {
	var f Filter
	var err error
	f, err = Parse("One~a")
	if err != nil {
		t.Errorf("Unable to parse RE expression")
	}
	if len(f) != 1 {
		t.Errorf("filter parsed incorrectly")
	}

	data := []interface{}{
		TestFilterStruct{
			One:   "a",
			Two:   "b",
			Three: "c",
		},
		TestFilterStruct{
			One:   "1",
			Two:   "2",
			Three: "3",
		},
		TestFilterStruct{
			One:   "a",
			Two:   "b",
			Three: "z",
		},
	}

	if _, err = f.Process(data); err != nil {
		t.Errorf("Error processing data")
	}
}

func TestBadRE(t *testing.T) {
	_, err := Parse("One~(qs*")
	if err == nil {
		t.Errorf("Expected RE parse error, got none")
	}
}

func TestQuoteStripping(t *testing.T) {
	tests := []struct {
		name  string
		spec  string
		field string
		want  string
	}{
		{"single quotes stripped", "One='hello'", "One", "hello"},
		{"double quotes stripped", `One="hello"`, "One", "hello"},
		{"no quotes unchanged", "One=hello", "One", "hello"},
		{"single quotes on NE op", "One!='hello'", "One", "hello"},
		{"single quotes on regex op", "One~'he.*'", "One", "he.*"},
		{"unmatched left quote not stripped", "One='hello", "One", "'hello"},
		{"unmatched right quote not stripped", "One=hello'", "One", "hello'"},
		{"mismatched quotes not stripped", `One='hello"`, "One", `'hello"`},
		{"empty quoted string", "One=''", "One", ""},
		{"value with spaces inside quotes", "One='hello world'", "One", "hello world"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f, err := Parse(tt.spec)
			assert.NoError(t, err)
			assert.Equal(t, tt.want, f[tt.field].Value)
		})
	}
}

func TestQuoteStrippingFilterMatch(t *testing.T) {
	data := []interface{}{
		TestFilterStruct{One: "hello", Two: "b", Three: "c"},
		TestFilterStruct{One: "world", Two: "b", Three: "c"},
	}

	// With quotes — should behave exactly like without quotes.
	fQuoted, err := Parse("One='hello'")
	assert.NoError(t, err)
	fUnquoted, err := Parse("One=hello")
	assert.NoError(t, err)

	rQuoted, err := fQuoted.Process(data)
	assert.NoError(t, err)
	rUnquoted, err := fUnquoted.Process(data)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(rQuoted.([]interface{})))
	assert.Equal(t, rUnquoted.([]interface{})[0], rQuoted.([]interface{})[0])
}

func TestQuoteStrippingRegexMatch(t *testing.T) {
	data := []interface{}{
		TestFilterStruct{One: "pg-test-40", Two: "b", Three: "c"},
		TestFilterStruct{One: "pg-test-41", Two: "b", Three: "c"},
		TestFilterStruct{One: "pg-other-40", Two: "b", Three: "c"},
	}

	// Quoted regex should match same items as unquoted.
	fQuoted, err := Parse("One~'pg-test-4.'")
	assert.NoError(t, err)
	fUnquoted, err := Parse("One~pg-test-4.")
	assert.NoError(t, err)

	rQuoted, err := fQuoted.Process(data)
	assert.NoError(t, err)
	rUnquoted, err := fUnquoted.Process(data)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(rQuoted.([]interface{})))
	assert.Equal(t, len(rUnquoted.([]interface{})), len(rQuoted.([]interface{})))
}

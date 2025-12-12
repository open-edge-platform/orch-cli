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

package order

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

type SortIncludedStruct struct {
	Seven string
}

type SortTestStruct struct {
	Id    int
	One   string
	Two   string
	Three uint
	Four  int
	Six   SortIncludedStruct
	Eight *SortIncludedStruct
}

var testSetOne = []SortTestStruct{
	{
		Id:    0,
		One:   "a",
		Two:   "x",
		Three: 10,
		Four:  1,
		Six:   SortIncludedStruct{Seven: "o"},
		Eight: &SortIncludedStruct{Seven: "o"},
	},
	{
		Id:    1,
		One:   "a",
		Two:   "c",
		Three: 1,
		Four:  10,
		Six:   SortIncludedStruct{Seven: "p"},
		Eight: &SortIncludedStruct{Seven: "p"},
	},
	{
		Id:    2,
		One:   "a",
		Two:   "b",
		Three: 2,
		Four:  1000,
		Six:   SortIncludedStruct{Seven: "q"},
		Eight: &SortIncludedStruct{Seven: "q"},
	},
	{
		Id:    3,
		One:   "a",
		Two:   "a",
		Three: 3,
		Four:  100,
		Six:   SortIncludedStruct{Seven: "r"},
		Eight: &SortIncludedStruct{Seven: "r"},
	},
	{
		Id:    4,
		One:   "b",
		Two:   "a",
		Three: 3,
		Four:  0,
		Six:   SortIncludedStruct{Seven: "s"},
		Eight: &SortIncludedStruct{Seven: "s"},
	},
}

var testSetTwo = []SortTestStruct{
	{
		Id:    0,
		One:   "a",
		Two:   "x",
		Three: 10,
		Four:  10,
	},
	{
		Id:    1,
		One:   "a",
		Two:   "y",
		Three: 1,
		Four:  1,
	},
}

func Verify(v []SortTestStruct, order []int) bool {
	for i, item := range v {
		if item.Id != order[i] {
			return false
		}
	}
	return true
}

func TestSort(t *testing.T) {
	s, err := Parse("+One,-Two")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if !Verify(o.([]SortTestStruct), []int{0, 1, 2, 3, 4}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortASC(t *testing.T) {
	s, err := Parse("+One,Two")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetTwo)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if !Verify(o.([]SortTestStruct), []int{0, 1}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortUintASC(t *testing.T) {
	s, err := Parse("Three,One")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if !Verify(o.([]SortTestStruct), []int{1, 2, 3, 4, 0}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortUintDSC(t *testing.T) {
	s, err := Parse("-Three,One")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if !Verify(o.([]SortTestStruct), []int{0, 3, 4, 2, 1}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortUintDSC2(t *testing.T) {
	s, err := Parse("-Three,One")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetTwo)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if !Verify(o.([]SortTestStruct), []int{0, 1}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortIntASC(t *testing.T) {
	s, err := Parse("Four,One")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}
	if !Verify(o.([]SortTestStruct), []int{4, 0, 1, 3, 2}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortIntDSC(t *testing.T) {
	s, err := Parse("-Four,One")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}
	if !Verify(o.([]SortTestStruct), []int{2, 3, 1, 0, 4}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortIntDSC2(t *testing.T) {
	s, err := Parse("-Four,One")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetTwo)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}
	if !Verify(o.([]SortTestStruct), []int{0, 1}) {
		t.Errorf("incorrect sort")
	}
}

func TestOperString(t *testing.T) {
	if ASC.String() != "ASC" {
		t.Errorf("ASC to string failed")
	}
	if DSC.String() != "DSC" {
		t.Errorf("DSC to string failed")
	}
	var o Operation = 5 // Invalid
	if o.String() != "ASC" {
		t.Errorf("to string default failed")
	}
}

func TestSortSingle(t *testing.T) {
	s, err := Parse("-Four,One")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne[0])
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if o == nil {
		t.Errorf("expected value, got nil")
	}

	r, ok := o.(SortTestStruct)
	if !ok {
		t.Errorf("Unexpected result type")
	}

	if r.Id != testSetOne[0].Id {
		t.Errorf("results don't match input")
	}
}

func TestSortDotted(t *testing.T) {
	s, err := Parse("+Six.Seven")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if !Verify(o.([]SortTestStruct), []int{0, 1, 2, 3, 4}) {
		t.Errorf("incorrect sort")
	}
}

func TestSortDottedPointer(t *testing.T) {
	s, err := Parse("+Eight.Seven")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	if err != nil {
		t.Errorf("Sort failed: %s", err.Error())
	}

	if !Verify(o.([]SortTestStruct), []int{0, 1, 2, 3, 4}) {
		t.Errorf("incorrect sort")
	}
}

func TestInvalidDotted(t *testing.T) {
	s, err := Parse("+Six.Nonexistent")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	assert.EqualError(t, err, "Failed to find field Nonexistent while sorting")
	if o != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestDotOnString(t *testing.T) {
	s, err := Parse("+One.IsNotAStruct")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	assert.EqualError(t, err, "Dotted field name specified in filter did not resolve to a valid field")
	if o != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestSortOnStuct(t *testing.T) {
	s, err := Parse("+Six")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	assert.EqualError(t, err, "Cannot sort on a field that is a struct")
	if o != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestSortOnPointerStuct(t *testing.T) {
	s, err := Parse("+Eight")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	assert.EqualError(t, err, "Cannot sort on a field that is a struct")
	if o != nil {
		t.Errorf("expected no results, got some")
	}
}

func TestTrailingDot(t *testing.T) {
	s, err := Parse("+Six.Seven.")
	if err != nil {
		t.Errorf("Unable to parse sort specification")
	}
	o, err := s.Process(testSetOne)
	assert.EqualError(t, err, "Dotted field name specified in filter did not resolve to a valid field")
	if o != nil {
		t.Errorf("expected no results, got some")
	}
}

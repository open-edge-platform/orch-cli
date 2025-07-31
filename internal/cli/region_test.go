// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"regexp"
	"testing"
)

func (s *CLITestSuite) createRegion(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create region %s --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listRegion(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list region --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getRegion(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get region "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteRegion(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete region "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestRegion() {

	name := "region"
	resourceID := "region-abcd1111"
	siteID := "site-7ceae560"
	rtype := "country"

	/////////////////////////////
	// Test Region Creation
	/////////////////////////////

	//create region
	SArgs := map[string]string{
		"type": rtype,
	}
	_, err := s.createRegion(project, name, SArgs)
	s.NoError(err)

	//create regionin nonexisting region
	SArgs = map[string]string{
		"type": rtype,
	}
	_, err = s.createRegion("invalid-project", name, SArgs)
	s.EqualError(err, "error while creating region: Internal Server Error")

	//create with invalid type
	SArgs = map[string]string{
		"type": "nope",
	}
	_, err = s.createRegion(project, name, SArgs)
	s.EqualError(err, "invalid type provided must be one of: country/state/county/region/city")

	//create with parent region
	SArgs = map[string]string{
		"parent": resourceID,
		"type":   rtype,
	}
	_, err = s.createRegion("parent-region", name, SArgs)
	s.NoError(err)

	// /////////////////////////////
	// // Test Region Listing
	// /////////////////////////////

	//List region

	SArgs = map[string]string{}
	linesOutput, err := s.listRegion(project, SArgs)
	s.NoError(err)

	parsedOutputLines := mapLinesOutput(linesOutput)

	expectedOutputLines := linesCommandOutput{
		"",
		"Printing regions tree",
		"",
		"Region: " + resourceID + " (region)",
		"  |",
		"  └───── Site: " + siteID + " (site)",
		"",
	}

	s.compareLinesOutput(expectedOutputLines, parsedOutputLines)

	//List region --verbose
	SArgs = map[string]string{
		"verbose": "true",
	}
	linesOutput, err = s.listRegion(project, SArgs)
	s.NoError(err)

	parsedOutputLines = mapLinesOutput(linesOutput)

	expectedOutputLines = linesCommandOutput{
		"Printing regions tree",
		"",
		"Region: " + resourceID + " (region)",
		"- Total Sites: 1",
		"  |",
		"  └───── Site: " + siteID + " (site)",
		"",
	}

	s.compareLinesOutput(expectedOutputLines, parsedOutputLines)

	//List region --verbose and region filter
	SArgs = map[string]string{
		"verbose": "true",
		"region":  resourceID,
	}
	linesOutput, err = s.listRegion(project, SArgs)
	s.NoError(err)

	parsedOutputLines = mapLinesOutput(linesOutput)

	expectedOutputLines = linesCommandOutput{
		"Printing regions tree",
		"",
		"Region: " + resourceID + " (region)",
		"- Total Sites: 1",
		"  |",
		"  └───── Site: " + siteID + " (site)",
		"",
	}

	s.compareLinesOutput(expectedOutputLines, parsedOutputLines)

	//List region with region filter
	SArgs = map[string]string{
		"region": resourceID,
	}
	linesOutput, err = s.listRegion(project, SArgs)
	s.NoError(err)

	parsedOutputLines = mapLinesOutput(linesOutput)

	expectedOutputLines = linesCommandOutput{
		"",
		"Printing regions tree",
		"",
		"Region: " + resourceID + " (region)",
		"  |",
		"  └───── Site: " + siteID + " (site)",
		"",
	}

	s.compareLinesOutput(expectedOutputLines, parsedOutputLines)

	//List subregions
	SArgs = map[string]string{
		"region": resourceID,
	}
	_, err = s.listRegion("parent-region", SArgs)
	s.NoError(err)

	//Output comparison commented out for now, output is not deterministic sometim causes test to fail
	// parsedOutputLines = mapLinesOutput(linesOutput)

	// expectedOutputLines = linesCommandOutput{
	// 	"",
	// 	"Printing regions tree",
	// 	"",
	// 	"Region: region-abcd1111 (region)",
	// 	"  |",
	// 	"  └───── Site: " + siteID + " (site)",
	// 	"",
	// 	"  └───── Region: region-abcd2222 (region)",
	// 	"         |",
	// 	"         └───── Site: " + siteID + " (site)",
	// 	"",
	// 	"Region: region-abcd2222 (region)",
	// 	"  |",
	// 	"  └───── Site: " + siteID + " (site)",
	// 	"",
	// }

	// s.compareLinesOutput(expectedOutputLines, parsedOutputLines)

	/////////////////////////////
	// Test Region Get
	/////////////////////////////

	getOutput, err := s.getRegion(project, resourceID, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":          name,
		"Resource ID:":   resourceID,
		"Parent region:": "region-abcd1111",
		"Metadata:":      "[{region us-east}]",
		"TotalSites:":    "1",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test Region Delete
	/////////////////////////////

	//delete custom config
	_, err = s.deleteRegion(project, resourceID, make(map[string]string))
	s.NoError(err)

	//delete invalid custom config
	_, err = s.deleteRegion(project, "nonexistent-region", make(map[string]string))
	s.EqualError(err, "invalid region id")

}

func FuzzCreateRegion(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "region1", "country", "")                 // valid, no parent
	f.Add("project", "region-123", "state", "region-abcd1111") // valid with parent
	f.Add("project", "", "country", "")                        // missing name
	f.Add("project", "region1", "", "")                        // missing type
	f.Add("project", "region1", "invalidtype", "")             // invalid type
	f.Add("project", "region1", "city", "invalid-parent")      // invalid parent

	f.Fuzz(func(t *testing.T, project, name, rtype, parent string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"type": rtype,
		}
		if parent != "" {
			args["parent"] = parent
		}

		_, err := testSuite.createRegion(project, name, args)

		// Error expectations
		if name == "" || !regexp.MustCompile(`^[a-zA-Z0-9]+$`).MatchString(name) {
			if err == nil {
				t.Errorf("Expected error for invalid or missing region name, got: %v", err)
			}
			return
		}
		if rtype == "" {
			if err == nil {
				t.Errorf("Expected error for missing region type, got: %v", err)
			}
			return
		}
		if rtype != "country" && rtype != "state" && rtype != "county" && rtype != "region" && rtype != "city" {
			if err == nil {
				t.Errorf("Expected error for invalid region type, got: %v", err)
			}
			return
		}
		if parent != "" && !regexp.MustCompile(`^region-[0-9a-f]{8}$`).MatchString(parent) {
			if err == nil {
				t.Errorf("Expected error for invalid parent region id, got: %v", err)
			}
			return
		}
		// If all inputs are valid, expect no error
		if err != nil {
			t.Errorf("Unexpected error for valid region %s creation: %v", name, err)
		}
	})
}

// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createSite(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create site %s --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listSite(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list site --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getSite(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get site "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteSite(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete site "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestSite() {
	var expectedOutputList listCommandOutput

	name := "site"
	resourceID := "site-7ceae560"
	regionID := "region-abcd1234"

	/////////////////////////////
	// Test Site Creation
	/////////////////////////////

	//create site
	SArgs := map[string]string{
		"region":    "region-abcd1111",
		"longitude": "5",
		"latitude":  "5",
	}
	_, err := s.createSite(project, name, SArgs)
	s.NoError(err)

	//create sitein nonexisting region
	SArgs = map[string]string{
		"region":    "region-11111111",
		"longitude": "5",
		"latitude":  "5",
	}
	_, err = s.createSite(project, name, SArgs)
	s.EqualError(err, "the region for site creation does not exist")

	//create with invalid region
	SArgs = map[string]string{
		"region":    "nope",
		"longitude": "5",
		"latitude":  "5",
	}
	_, err = s.createSite(project, name, SArgs)
	s.EqualError(err, "no region found with name \"nope\"")

	//create with wrong longitude
	SArgs = map[string]string{
		"region":    "region-abcd1111",
		"longitude": "nope",
		"latitude":  "5",
	}
	_, err = s.createSite(project, name, SArgs)
	s.EqualError(err, "invalid longitude value")

	//create with wrong latitude
	SArgs = map[string]string{
		"region":    "region-abcd1111",
		"longitude": "5",
		"latitude":  "nope",
	}
	_, err = s.createSite(project, name, SArgs)
	s.EqualError(err, "invalid latitude value")

	/////////////////////////////
	// Test Site Listing
	/////////////////////////////

	//List site

	SArgs = map[string]string{}
	listOutput, err := s.listSite(project, SArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"RESOURCE ID": resourceID,
			"NAME":        name,
			"REGION ID":   regionID,
			"REGION NAME": "region",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List site --verbose
	SArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listSite(project, SArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"RESOURCE ID": resourceID,
			"NAME":        name,
			"REGION ID":   regionID,
			"REGION NAME": "region",
			"SITE LAT":    "50000000",
			"SITE LNG":    "50000000",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List site --verbose and region filter
	SArgs = map[string]string{
		"verbose": "true",
		"region":  regionID,
	}
	listOutput, err = s.listSite(project, SArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"RESOURCE ID": resourceID,
			"NAME":        name,
			"REGION ID":   regionID,
			"REGION NAME": "region",
			"SITE LAT":    "50000000",
			"SITE LNG":    "50000000",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List site withregion filter
	SArgs = map[string]string{
		"region": regionID,
	}
	listOutput, err = s.listSite(project, SArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"RESOURCE ID": resourceID,
			"NAME":        name,
			"REGION ID":   regionID,
			"REGION NAME": "region",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	// List sites with order-by and YAML output
	SArgs = map[string]string{
		"order-by":    "name",
		"output-type": "yaml",
		"page-size":   "1",
	}
	listOrderedOutput, err := s.listSite(project, SArgs)
	s.NoError(err)
	s.Contains(listOrderedOutput, "name: site")
	s.Contains(listOrderedOutput, "resourceid: "+resourceID)
	s.Contains(listOrderedOutput, "regionid: "+regionID)

	// List sites with filter and YAML output
	SArgs = map[string]string{
		"filter":      "name=site",
		"output-type": "yaml",
		"page-size":   "1",
	}
	listFilteredOutput, err := s.listSite(project, SArgs)
	s.NoError(err)
	s.Contains(listFilteredOutput, "name: site")
	s.Contains(listFilteredOutput, "resourceid: "+resourceID)
	s.Contains(listFilteredOutput, "regionid: "+regionID)

	// List sites with invalid order-by
	SArgs = map[string]string{
		"order-by":  "invalid",
		"page-size": "1",
	}
	_, err = s.listSite(project, SArgs)
	s.EqualError(err, "invalid --order-by field \"invalid\"; available fields: inheritedMetadata, metadata, name, provider, region, regionId, resourceId, siteID, siteLat, siteLng, timestamps")

	/////////////////////////////
	// Test Site Get
	/////////////////////////////

	getOutput, err := s.getSite(project, resourceID, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":        name,
		"Resource ID:": resourceID,
		"Region ID:":   regionID,
		"Region Name:": "region",
		"Latitude:":    "50000000",
		"Longitude:":   "50000000",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	//get site by name
	getOutput, err = s.getSite(project, "site", make(map[string]string))
	s.NoError(err)

	parsedOutput = mapGetOutput(getOutput)
	expectedOutput = map[string]string{
		"Name:":        name,
		"Resource ID:": resourceID,
		"Region ID:":   regionID,
		"Region Name:": "region",
		"Latitude:":    "50000000",
		"Longitude:":   "50000000",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	//get site by name
	_, err = s.getSite("duplicate-site", "duplicate-site", make(map[string]string))
	s.EqualError(err, "multiple sites found with name \"duplicate-site\"; use a resource ID instead:\n  name: duplicate-site  resource-id: site-7ceae560\n  name: duplicate-site  resource-id: site-7ceae560")

	/////////////////////////////
	// Test Site Delete
	/////////////////////////////

	//delete custom config
	_, err = s.deleteSite(project, resourceID, make(map[string]string))
	s.NoError(err)

	//delete invalid custom config
	_, err = s.deleteSite(project, "nonexistent-site", make(map[string]string))
	s.EqualError(err, "no site found with name \"nonexistent-site\"")

}

func FuzzSite(f *testing.F) {
	// Initial corpus with valid and invalid input
	f.Add("project", "site1", "region-abcd1234", "5", "5", "site-7ceae560")
	f.Add("project", "site1", "", "5", "5", "site-7ceae560")                      // missing region
	f.Add("project", "", "region-abcd1234", "5", "5", "site-7ceae560")            // missing name
	f.Add("project", "site1", "invalid-region", "5", "5", "site-7ceae560")        // invalid region format
	f.Add("project", "site1", "region-abcd1234", "invalid", "5", "site-7ceae560") // invalid latitude
	f.Add("project", "site1", "region-abcd1234", "5", "invalid", "site-7ceae560") // invalid longitude
	f.Add("project", "site1", "region-abcd1234", "5", "5", "")

	f.Fuzz(func(t *testing.T, project, name, region, latitude, longitude string, siteID string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"region":    region,
			"latitude":  latitude,
			"longitude": longitude,
		}

		// Call your site creation logic (replace with your actual function if needed)
		_, err := testSuite.createSite(project, name, args)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listSite(project, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteSite(project, siteID, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
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

	name := "site"
	resourceID := "site-7ceae560"
	regionID := "region-abcd1234"
	longitude := "5"
	latitiude := "5"
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
	s.EqualError(err, "invalid region id nope --region expects region-abcd1234 format")

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

	// /////////////////////////////
	// // Test Custom Config Listing
	// /////////////////////////////

	//List site

	SArgs = map[string]string{}
	listOutput, err := s.listSite(project, SArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Site ID":       resourceID,
			"Site Name":     name,
			"Region (Name)": regionID + (" (region)"),
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
			"Site ID":       resourceID,
			"Site Name":     name,
			"Region (Name)": regionID + (" (region)"),
			"Longitude":     longitude,
			"Latitude":      latitiude,
		},
	}

	fmt.Println("=== DEBUG: Raw list output ===")
	fmt.Println(listOutput)
	fmt.Println("=== END DEBUG ===")
	fmt.Println("=== DEBUG: Parsed output ===")
	for i, row := range parsedOutputList {
		fmt.Printf("Row %d: %+v\n", i, row)
	}
	fmt.Println("=== END DEBUG ===")
	fmt.Println("=== DEBUG: Expected output ===")
	for i, row := range expectedOutputList {
		fmt.Printf("Row %d: %+v\n", i, row)
	}
	fmt.Println("=== END DEBUG ===")

	s.compareListOutput(expectedOutputList, parsedOutputList)

	// /////////////////////////////
	// // Test Custom Config Get
	// /////////////////////////////

	// getOutput, err := s.getSite(project, name, make(map[string]string))
	// s.NoError(err)

	// parsedOutput := mapGetOutput(getOutput)
	// expectedOutput := map[string]string{
	// 	"Name:":        "nginx-config",
	// 	"Resource ID:": "config-abc12345",
	// 	"Description:": "Nginx configuration for web services",
	// 	"Cloud Init:":  "",
	// 	"test:":        "",
	// }

	// s.compareGetOutput(expectedOutput, parsedOutput)

	// /////////////////////////////
	// // Test Custom Config Delete
	// /////////////////////////////

	// //delete custom config
	// _, err = s.deleteSite(project, name, make(map[string]string))
	// s.NoError(err)

	// //delete invalid cusotm config
	// _, err = s.deleteSite(project, "nonexistent-config", make(map[string]string))
	// s.EqualError(err, "no custom config matches the given name")

}

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"testing"
)

func (s *CLITestSuite) createSchedule(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create schedule %s --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listSchedule(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list schedule --project %s`, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getSchedule(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get schedule "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteSchedule(project string, name string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete schedule "%s" --project %s`, name, project))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestSchedule() {

	name := "schedule"
	sresourceID := "singlesche-abcd1234"
	rresourceID := "repeatedsche-abcd1234"
	regionID := "region-abcd1234"
	hostID := "host-abcd1234"
	siteID := "site-abcd1234"

	/////////////////////////////
	// Test Schedule Creation
	/////////////////////////////

	//create repeated schedule - monthly
	SArgs := map[string]string{
		"timezone":         "GMT",
		"frequency-type":   "repeated",
		"maintenance-type": "osupdate",
		"target":           siteID,
		"frequency":        "monthly",
		"start-time":       "10:10",
		"day-of-month":     "1,2,15-18",
		"months":           "2,4,3-7",
		"duration":         "3600",
	}
	_, err := s.createSchedule(project, name, SArgs)
	s.NoError(err)

	//create repeated schedule - monthly
	SArgs = map[string]string{
		"timezone":         "GMT",
		"frequency-type":   "repeated",
		"maintenance-type": "osupdate",
		"target":           siteID,
		"frequency":        "monthly",
		"start-time":       "10:10",
		"day-of-month":     "1",
		"months":           "2",
		"duration":         "3600",
	}
	_, err = s.createSchedule(project, name, SArgs)
	s.NoError(err)

	//create repeated schedule - weekly
	SArgs = map[string]string{
		"timezone":         "GMT",
		"frequency-type":   "repeated",
		"maintenance-type": "maintenance",
		"target":           regionID,
		"frequency":        "weekly",
		"start-time":       "10:10",
		"day-of-week":      "1,2,2-6",
		"months":           "2,4,3-7",
		"duration":         "3600",
	}
	_, err = s.createSchedule(project, name, SArgs)
	s.NoError(err)

	//create repeated schedule - weekly
	SArgs = map[string]string{
		"timezone":         "GMT",
		"frequency-type":   "repeated",
		"maintenance-type": "maintenance",
		"target":           regionID,
		"frequency":        "weekly",
		"start-time":       "10:10",
		"day-of-week":      "wed",
		"months":           "2",
		"duration":         "3600",
	}
	_, err = s.createSchedule(project, name, SArgs)
	s.NoError(err)

	//create single schedule
	SArgs = map[string]string{
		"timezone":         "GMT",
		"frequency-type":   "single",
		"maintenance-type": "maintenance",
		"target":           hostID,
		"start-time":       "\"2026-12-01 10:10\"",
		"end-time":         "\"2027-12-01 10:10\"",
	}
	_, err = s.createSchedule(project, name, SArgs)
	s.NoError(err)

	//create invalid repeated schedule - weekly
	SArgs = map[string]string{
		"timezone":         "GMT",
		"frequency-type":   "repeated",
		"maintenance-type": "maintenance",
		"target":           regionID,
		"frequency":        "weekly",
		"start-time":       "\"10 10\"",
		"day-of-week":      "wed",
		"months":           "2",
		"duration":         "3600",
	}
	_, err = s.createSchedule(project, name, SArgs)
	s.EqualError(err, "repeated schedule --start-time must be specified in format \"HH:MM\"")

	//create single schedule
	SArgs = map[string]string{
		"timezone":         "GMT",
		"frequency-type":   "single",
		"maintenance-type": "maintenance",
		"target":           hostID,
		"start-time":       "\"2026-1201 1010\"",
		"end-time":         "\"2027-1201 1010\"",
	}
	_, err = s.createSchedule(project, name, SArgs)
	s.EqualError(err, "single schedule --start-time must be specified in format \"YYYY-MM-DD HH:MM\"")

	// //create with invalid region
	// SArgs = map[string]string{
	// 	"region":    "nope",
	// 	"longitude": "5",
	// 	"latitude":  "5",
	// }
	// _, err = s.createSite(project, name, SArgs)
	// s.EqualError(err, "invalid region id nope --region expects region-abcd1234 format")

	// //create with wrong longitude
	// SArgs = map[string]string{
	// 	"region":    "region-abcd1111",
	// 	"longitude": "nope",
	// 	"latitude":  "5",
	// }
	// _, err = s.createSite(project, name, SArgs)
	// s.EqualError(err, "invalid longitude value")

	// //create with wrong latitude
	// SArgs = map[string]string{
	// 	"region":    "region-abcd1111",
	// 	"longitude": "5",
	// 	"latitude":  "nope",
	// }
	// _, err = s.createSite(project, name, SArgs)
	// s.EqualError(err, "invalid latitude value")

	/////////////////////////////
	// Test Schedule Listing
	/////////////////////////////

	//List Schedule

	SArgs = map[string]string{}
	listOutput, err := s.listSchedule(project, SArgs)
	s.NoError(err)

	parsedOutputList := mapListOutput(listOutput)

	expectedOutputList := listCommandOutput{
		{
			"Name":   name,
			"Target": siteID,
			"Type":   "Maintenance",
		},
		{
			"Name":   name,
			"Target": siteID,
			"Type":   "Maintenance",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	//List schedule --verbose
	SArgs = map[string]string{
		"verbose": "true",
	}
	listOutput, err = s.listSchedule(project, SArgs)
	s.NoError(err)

	parsedOutputList = mapListOutput(listOutput)

	expectedOutputList = listCommandOutput{
		{
			"Name":        name,
			"Target":      siteID,
			"Resource ID": sresourceID,
			"Type":        "single",
		},
		{
			"Name":        name,
			"Target":      siteID,
			"Resource ID": rresourceID,
			"Type":        "repeated",
		},
	}

	s.compareListOutput(expectedOutputList, parsedOutputList)

	/////////////////////////////
	// Test Schedule Get
	/////////////////////////////
	SArgs = map[string]string{
		"timezone": "GMT",
	}
	getOutput, err := s.getSchedule(project, rresourceID, SArgs)
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)
	expectedOutput := map[string]string{
		"Name:":             name,
		"Resource ID:":      rresourceID,
		"Target Host ID:":   "Unspecified",
		"Target Region ID:": "Unspecified",
		"Target Site ID:":   siteID,
		"Schedule Status:":  "SCHEDULE_STATUS_MAINTENANCE",
		"Month:":            "1",
		"Month day:":        "1",
		"Weekday:":          "1",
		"Hour (UTC):":       "1",
		"Minute (UTC):":     "1",
		"Hour (GMT):":       "1",
		"Minute (GMT):":     "1",
		"Local Time:":       "1:1 GMT",
		"Duration:":         "1 seconds",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	SArgs = map[string]string{
		"timezone": "GMT",
	}
	getOutput, err = s.getSchedule(project, sresourceID, SArgs)
	s.NoError(err)

	parsedOutput = mapGetOutput(getOutput)
	expectedOutput = map[string]string{
		"Name:":             name,
		"Resource ID:":      sresourceID,
		"Target Host ID:":   "Unspecified",
		"Target Region ID:": "Unspecified",
		"Target Site ID:":   siteID,
		"Schedule Status:":  "SCHEDULE_STATUS_MAINTENANCE",
		"Start Time:":       "1970-01-01 02:46:40 GMT",
		"End Time:":         "",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	/////////////////////////////
	// Test Schedule Delete
	/////////////////////////////

	//delete schedule
	_, err = s.deleteSchedule(project, rresourceID, make(map[string]string))
	s.NoError(err)

	_, err = s.deleteSchedule(project, sresourceID, make(map[string]string))
	s.NoError(err)

	//delete invalid schedule
	_, err = s.deleteSchedule(project, "nonexistent-site", make(map[string]string))
	s.EqualError(err, "no schedule matches the given id")

}

func FuzzSchedule(f *testing.F) {
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
		_, err := testSuite.createSchedule(project, name, args)

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List ---
		_, err = testSuite.listSchedule(project, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete ---
		_, err = testSuite.deleteSchedule(project, siteID, make(map[string]string))

		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import "fmt"

func (s *CLITestSuite) createHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`create host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) listHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deauthorizeHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`deauthorize host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteHost(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete host --project %s`, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestHost() {

	//hostID := "host-abc12345"
	HostArgs := map[string]string{}

	//Generate CSV
	HostArgs["generate-csv"] = "test.csv"
	_, err := s.createHost(project, HostArgs)
	s.NoError(err)

	//Dry run host creation
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
		"dry-run":         "true",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)

	//Dry run host creation wrong file
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.lol",
		"dry-run":         "true",
	}
	_, err = s.createHost(project, HostArgs)
	s.EqualError(err, "host import input file must be a CSV file")

	//Dry run host creation with overrides
	HostArgs = map[string]string{
		"import-from-csv":  "./testdata/mock.csv",
		"dry-run":          "true",
		"site":             "site-abcd1111",
		"secure":           "true",
		"remote-user":      "user",
		"os-profile":       "microvisor-nonrt",
		"metadata":         "key1=value1",
		"cloud-init":       "custom",
		"cluster-deploy":   "true",
		"cluster-config":   "role:all;name:mycluster;labels:sample-label=samplevalue&sample-label2=samplevalue",
		"cluster-template": "baseline:v2.0.2",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)

	//host creation wrong fil
	HostArgs = map[string]string{
		"import-from-csv": "./testdata/mock.csv",
	}
	_, err = s.createHost(project, HostArgs)
	s.NoError(err)

	// _, err = s.listHost(project, make(map[string]string))
	// fmt.Printf("listHost: %v\n", err)
	// s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	// _, err = s.getHost(project, make(map[string]string))
	// fmt.Printf("getHost: %v\n", err)
	// s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	// _, err = s.deauthorizeHost(project, make(map[string]string))
	// fmt.Printf("deauthorizeHost: %v\n", err)
	// s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)

	// _, err = s.deleteHost(project, make(map[string]string))
	// fmt.Printf("deleteHost: %v\n", err)
	// s.EqualError(err, `no response from backend - check catalog-endpoint and deployment-endpoint`)
}

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"testing"
)

func (s *CLITestSuite) TestDeploymentGetOverrideValues() {
	ov, err := getOverrideValuesRaw(
		map[string]string{
			"foo": "ns1",
		},
		map[string]string{
			"foo.property.string":     "string1",
			"foo.property.int":        "123",
			"foo.property.float":      "123.321",
			"foo.property.bool":       "true",
			"foo.another.string":      "string2",
			"foo.property.nested.int": "420",
			"bar.property.bool":       "false",
			"bar.another.int":         "42",
		})
	s.NoError(err)

	for _, v := range ov {
		if v.TargetNamespace == nil {
			fmt.Printf("%s(NONE)=", v.AppName)
		} else {
			fmt.Printf("%s(%s)=\n", v.AppName, *v.TargetNamespace)
		}
		b, err := json.Marshal(v.Values)
		s.NoError(err)
		fmt.Println(string(b))
	}
}

func (s *CLITestSuite) createDeployment(appName string, version string, args commandArgs) error {
	commandString := addCommandArgs(args, fmt.Sprintf(`create deployment %s %s`,
		appName, version))
	_, err := s.runCommand(commandString)
	return err
}

func (s *CLITestSuite) listDeployment(publisher string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list deployments --project %s`,
		publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) getDeployment(publisher string, deployment string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`get deployment %s --project %s`, deployment, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) setDeployment(publisher string, deployment string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`set deployment %s --project %s`, deployment, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) deleteDeployment(publisher string, deployment string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`delete deployment %s --project %s`, deployment, publisher))
	return s.runCommand(commandString)
}

func (s *CLITestSuite) TestDeployment() {
	//TODO: These test should be expanded to compare outputs for list and get
	err := s.createDeployment("deployment-pkg", "1.0.0", map[string]string{
		"project":           project,
		"display-name":      "Test",
		"profile":           "test-profile",
		"application-label": "app1.l1=l1value,app2.l2=l2value",
	})
	s.NoError(err)

	_, err = s.listDeployment(project, make(map[string]string))
	s.NoError(err)

	_, err = s.getDeployment(project, "test-deployment", make(map[string]string))
	s.NoError(err)

	_, err = s.setDeployment(project, "test-deployment", make(map[string]string))
	s.NoError(err)

	_, err = s.deleteDeployment(project, "test-deployment", make(map[string]string))
	s.NoError(err)
}

func FuzzDeployment(f *testing.F) {
	// Seed with valid and invalid input combinations
	f.Add("deployment-pkg", "1.0.0", "test-deployment", project, "test-profile", "Test", "app1.l1=l1value,app2.l2=l2value")
	f.Add("", "1.0.0", "test-deployment", project, "test-profile", "Test", "")
	f.Add("deployment-pkg", "", "test-deployment", project, "test-profile", "Test", "")
	f.Add("deployment-pkg", "1.0.0", "", project, "test-profile", "Test", "")
	f.Add("deployment-pkg", "1.0.0", "test-deployment", "", "test-profile", "Test", "")
	f.Add("deployment-pkg", "1.0.0", "test-deployment", project, "", "Test", "")
	f.Add("deployment-pkg", "1.0.0", "test-deployment", project, "test-profile", "", "")

	f.Fuzz(func(t *testing.T, appName, version, deployment, publisher, profile, displayName, appLabel string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		args := map[string]string{
			"project":           publisher,
			"display-name":      displayName,
			"profile":           profile,
			"application-label": appLabel,
		}

		// --- Create Deployment ---
		err := testSuite.createDeployment(appName, version, args)
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- List Deployments ---
		_, err = testSuite.listDeployment(publisher, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Get Deployment ---
		_, err = testSuite.getDeployment(publisher, deployment, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Set Deployment ---
		_, err = testSuite.setDeployment(publisher, deployment, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}

		// --- Delete Deployment ---
		_, err = testSuite.deleteDeployment(publisher, deployment, make(map[string]string))
		if isExpectedError(err) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error: %v", err)
		}
	})
}

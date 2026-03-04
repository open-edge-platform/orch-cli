// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
)

func (s *CLITestSuite) listFeatures(project string, args commandArgs) (string, error) {
	commandString := addCommandArgs(args, fmt.Sprintf(`list features  --project %s`, project))
	return s.runCommand(commandString)
}
func (s *CLITestSuite) TestFeatures() {
	getOutput, err := s.listFeatures(project, make(map[string]string))
	s.NoError(err)

	parsedOutput := mapGetOutput(getOutput)

	expectedOutput := map[string]string{

		"Edge Orchestrator Features:":              "",
		"application-orchestration":                "enabled",
		"cluster-orchestration":                    "enabled",
		"edge-infrastructure-manager":              "enabled",
		"edge-infrastructure-manager.day2":         "enabled",
		"edge-infrastructure-manager.onboarding":   "enabled",
		"edge-infrastructure-manager.oob":          "enabled",
		"edge-infrastructure-manager.provisioning": "enabled",
		"edge-infrastructure-manager.oxm-profile":  "enabled",
		"multitenancy":                             "enabled",
		"orchestrator-observability":               "enabled",
	}

	s.compareGetOutput(expectedOutput, parsedOutput)

	//Disable a feature using config command
	_, err = s.runCommand(fmt.Sprintf(`config set orchestrator.features.edge-infrastructure-manager.onboarding.installed "false" --project %s`, project))
	s.NoError(err)

	getOutput, err = s.listFeatures(project, make(map[string]string))
	s.NoError(err)

	parsedOutput = mapGetOutput(getOutput)

	expectedOutput = map[string]string{

		"Edge Orchestrator Features:":              "",
		"application-orchestration":                "enabled",
		"cluster-orchestration":                    "enabled",
		"edge-infrastructure-manager":              "enabled",
		"edge-infrastructure-manager.day2":         "enabled",
		"edge-infrastructure-manager.onboarding":   "disabled",
		"edge-infrastructure-manager.oob":          "enabled",
		"edge-infrastructure-manager.provisioning": "enabled",
		"edge-infrastructure-manager.oxm-profile":  "enabled",
		"multitenancy":                             "enabled",
		"orchestrator-observability":               "enabled",
	}
	s.compareGetOutput(expectedOutput, parsedOutput)

	_, err = s.runCommand(fmt.Sprintf(`config set features.edge-infrastructure-manager.onboarding.installed true --project %s`, project))
	s.NoError(err)
}

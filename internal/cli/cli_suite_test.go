// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"github.com/open-edge-platform/app-orch-catalog/pkg/restClient"
	restproxy "github.com/open-edge-platform/app-orch-catalog/pkg/restProxy"
	authmock "github.com/open-edge-platform/cli/internal/cli/mocks/auth"
	catalogmock "github.com/open-edge-platform/cli/internal/cli/mocks/catalog"
	clustermock "github.com/open-edge-platform/cli/internal/cli/mocks/cluster"
	deploymentmock "github.com/open-edge-platform/cli/internal/cli/mocks/deployment"
	inframock "github.com/open-edge-platform/cli/internal/cli/mocks/infra"
	rpsmock "github.com/open-edge-platform/cli/internal/cli/mocks/rps"
	tenancymock "github.com/open-edge-platform/cli/internal/cli/mocks/tenancy"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	verboseOutput  = true
	simpleOutput   = false
	timestampRegex = `^[0-9-]*T[0-9:]*$`
	kcTest         = "http://unit-test-keycloak/realms/master"
)

type commandArgs map[string]string
type commandOutput map[string]map[string]string
type listCommandOutput []map[string]string
type linesCommandOutput []string

type CLITestSuite struct {
	suite.Suite
	proxy restproxy.MockRestProxy
}

func (s *CLITestSuite) SetupSuite() {
	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")

	// In your SetupSuite method, replace the existing timestamp line with:
	timestamp, _ := time.Parse(time.RFC3339, "2025-01-15T10:30:00Z")
	// Helper function to create timestamp pointers

	mctrl := gomock.NewController(s.T())

	// Setup all mocks
	auth.KeycloakFactory = authmock.CreateKeycloakMock(&s.Suite, mctrl)

	CatalogFactory = catalogmock.CreateCatalogMock(mctrl)
	InfraFactory = inframock.CreateInfraMock(mctrl, timestamp)
	ClusterFactory = clustermock.CreateClusterMock(mctrl)
	RpsFactory = rpsmock.CreateRpsMock(mctrl)
	DeploymentFactory = deploymentmock.CreateDeploymentMock(mctrl)
	TenancyFactory = tenancymock.CreateTenancyMock(mctrl)
}

func (s *CLITestSuite) TearDownSuite() {
	auth.KeycloakFactory = nil
	CatalogFactory = nil
	InfraFactory = nil
	ClusterFactory = nil
	RpsFactory = nil
	DeploymentFactory = nil
	TenancyFactory = nil

	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")
}

func (s *CLITestSuite) SetupTest() {
	s.proxy = restproxy.NewMockRestProxy(s.T())
	s.NotNil(s.proxy)
	err := s.login("u", "p")
	s.NoError(err)
}

func (s *CLITestSuite) TearDownTest() {
	s.NoError(s.proxy.Close())
	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")
}

func TestCLI(t *testing.T) {
	//t.Skip("defunct; to be reworked")
	suite.Run(t, &CLITestSuite{})
}

func (s *CLITestSuite) compareOutput(expected commandOutput, actual commandOutput) {
	for expectedK, expectedMap := range expected {
		actualMap := actual[expectedK]

		// Make sure there are no extra entries
		s.Equal(len(expectedMap), len(actualMap))

		// Make sure the entries match
		for k, v := range expectedMap {
			s.NotNil(actualMap[k])
			matchPattern := v
			if v != timestampRegex {
				matchPattern = regexp.QuoteMeta(v)
			}
			matches, _ := regexp.MatchString(matchPattern, actualMap[k])
			if !matches {
				s.True(matches, "Values don't match for %s", k)
			}
			s.True(matches, "Values don't match for %s", k)
		}
	}
}

func (s *CLITestSuite) compareListOutput(expected []map[string]string, actual []map[string]string) {
	s.Equal(len(expected), len(actual), "Number of rows should match")

	for i, expectedRow := range expected {
		if i >= len(actual) {
			s.Fail("Missing row at index %d", i)
			continue
		}

		actualRow := actual[i]

		// Make sure there are no extra entries
		s.Equal(len(expectedRow), len(actualRow), "Row %d should have same number of fields", i)

		// Make sure the entries match
		for k, v := range expectedRow {
			s.Contains(actualRow, k, "Row %d should contain field %s", i, k)
			// Use exact string comparison instead of regex
			s.Equal(v, actualRow[k], "Row %d field %s: expected '%s' but got '%s'", i, k, v, actualRow[k])
		}
	}
}

func (s *CLITestSuite) compareGetOutput(expected map[string]string, actual map[string]string) {
	// Make sure there are no extra entries
	s.Equal(len(expected), len(actual), "Number of fields should match")

	// Make sure the entries match
	for key, expectedValue := range expected {
		s.Contains(actual, key, "Should contain field %s", key)
		if actualValue, exists := actual[key]; exists {
			s.Equal(expectedValue, actualValue, "Field %s should match", key)
		}
	}
}

func (s *CLITestSuite) compareLinesOutput(expected linesCommandOutput, actual linesCommandOutput) {
	s.Equal(len(expected), len(actual), "Number of lines should match")

	for i, expectedLine := range expected {
		if i >= len(actual) {
			s.Fail("Missing line at index %d", i)
			continue
		}

		actualLine := actual[i]

		// Use exact string comparison for line content
		s.Equal(expectedLine, actualLine, "Line %d: expected '%s' but got '%s'", i, expectedLine, actualLine)
	}
}

func parseArgs(input string) []string {
	var args []string
	var current strings.Builder
	inQuotes := false

	for _, char := range input {
		switch char {
		case '"':
			inQuotes = !inQuotes
		case ' ':
			if !inQuotes {
				if current.Len() > 0 {
					args = append(args, current.String())
					current.Reset()
				}
			} else {
				current.WriteRune(char)
			}
		default:
			current.WriteRune(char)
		}
	}

	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

func (s *CLITestSuite) runCommand(commandArgs string) (string, error) {
	c := s.proxy.RestClient().ClientInterface.(*restClient.Client)
	cmd := getRootCmd()

	// Use custom parser instead of strings.Fields
	args := parseArgs(commandArgs)

	args = append(args, "--debug-headers")
	args = append(args, "--api-endpoint")
	args = append(args, c.Server)
	cmd.SetArgs(args)
	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	err := cmd.Execute()
	cmdOutput := stdout.String()
	return cmdOutput, err
}

func addCommandArgs(args commandArgs, commandString string) string {
	for argName, argValue := range args {
		if argValue == "" {
			commandString += fmt.Sprintf(" --%s", argName)
		} else {
			commandString += fmt.Sprintf(" --%s=%s", argName, argValue)
		}
	}
	return commandString
}

func mapCliOutput(output string) map[string]map[string]string {
	retval := make(map[string]map[string]string)
	lines := strings.Split(output, "\n")
	var headers []string

	for i, line := range lines {
		if i == 0 {
			// First line is the headers
			headers = strings.Split(line, "|")
			// Clean up headers
			for j := range headers {
				headers[j] = strings.TrimSpace(headers[j])
			}
		} else if line == "" {
			break
		} else {
			// Split data line by | instead of whitespace to match headers
			fields := strings.Split(line, "|")

			// Clean up fields
			for j := range fields {
				fields[j] = strings.TrimSpace(fields[j])
			}

			if len(fields) == 0 {
				continue
			}

			key := fields[0]
			retval[key] = make(map[string]string)

			// Only process fields that have corresponding headers
			maxFields := len(headers)
			if len(fields) < maxFields {
				maxFields = len(fields)
			}

			for fieldNumber := 0; fieldNumber < maxFields; fieldNumber++ {
				if fieldNumber < len(headers) && fieldNumber < len(fields) {
					headerKey := headers[fieldNumber]
					retval[key][headerKey] = fields[fieldNumber]
				}
			}
		}
	}
	return retval
}

func mapListOutput(output string) listCommandOutput {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) < 2 {
		return listCommandOutput{}
	}

	headerLine := lines[0]

	// Try to detect if this is space-separated (like site output) or pipe-separated (like host output)
	if strings.Contains(headerLine, "|") {
		// Pipe-separated format (existing host tests)
		return parsePipeSeparatedOutput(lines)
	}
	// Space-separated format (new site tests)
	return parseSpaceSeparatedOutput(lines)

}

func mapLinesOutput(output string) linesCommandOutput {
	lines := strings.Split(output, "\n")
	result := linesCommandOutput{}

	// Remove only the trailing newline if present, but preserve internal structure
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	for _, line := range lines {
		result = append(result, line)
	}

	return result
}

func parsePipeSeparatedOutput(lines []string) listCommandOutput {
	var headers []string
	result := listCommandOutput{}

	for i, line := range lines {
		if i == 0 {
			// First line is the headers
			headers = strings.Split(line, "|")
			// Clean up headers
			for j := range headers {
				headers[j] = strings.TrimSpace(headers[j])
			}
		} else if strings.TrimSpace(line) == "" {
			continue
		} else {
			// Split data line by |
			fields := strings.Split(line, "|")

			// Clean up fields
			for j := range fields {
				fields[j] = strings.TrimSpace(fields[j])
			}

			if len(fields) == 0 {
				continue
			}

			row := make(map[string]string)

			// Process fields that have corresponding headers
			maxFields := len(headers)
			if len(fields) < maxFields {
				maxFields = len(fields)
			}

			for fieldNumber := 0; fieldNumber < maxFields; fieldNumber++ {
				if fieldNumber < len(headers) && fieldNumber < len(fields) {
					headerKey := headers[fieldNumber]
					row[headerKey] = fields[fieldNumber]
				}
			}

			result = append(result, row)
		}
	}
	return result
}

func parseSpaceSeparatedOutput(lines []string) listCommandOutput {
	if len(lines) < 2 {
		return listCommandOutput{}
	}

	headerLine := lines[0]

	// Simple approach: find column positions by looking for gaps of 2+ spaces
	headers := []string{}
	positions := []int{}

	// Split by multiple spaces to get rough column boundaries
	parts := strings.Split(headerLine, "  ") // Split by 2+ spaces
	currentPos := 0

	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			// Find where this header actually starts in the original line
			headerStart := strings.Index(headerLine[currentPos:], trimmed)
			if headerStart >= 0 {
				actualStart := currentPos + headerStart
				headers = append(headers, trimmed)
				positions = append(positions, actualStart)
				currentPos = actualStart + len(trimmed)
			}
		}
	}

	result := listCommandOutput{}

	// Parse data rows using detected positions
	for _, line := range lines[1:] {
		if strings.TrimSpace(line) == "" {
			continue
		}

		row := make(map[string]string)

		for i, header := range headers {
			start := positions[i]
			var end int
			if i < len(positions)-1 {
				end = positions[i+1]
			} else {
				end = len(line)
			}

			if start < len(line) {
				if end > len(line) {
					end = len(line)
				}
				value := strings.TrimSpace(line[start:end])
				row[header] = value
			}
		}

		result = append(result, row)
	}

	return result
}

func mapGetOutput(output string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Ignore lines that start with more than 2 dashes
		if strings.HasPrefix(line, "---") {
			continue
		}

		// Handle lines that contain pipe separators
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(strings.Join(parts[1:], "|")) // Join all remaining parts

				// Remove quotes from value if present
				value = strings.Trim(value, `"`)

				// Handle host format lines that start with "-   |"
				if strings.HasPrefix(line, "-   |") {
					// For host format: "-   |Host Resurce ID:   | host-abc12345"
					// Remove the "-   |" prefix from the line, then extract key
					content := strings.TrimPrefix(line, "-   |")
					contentParts := strings.Split(content, "|")
					if len(contentParts) >= 2 {
						hostKey := strings.TrimSpace(contentParts[0])
						hostValue := strings.TrimSpace(strings.Join(contentParts[1:], "|")) // Join all remaining parts
						hostValue = strings.Trim(hostValue, `"`)
						result["-   "+hostKey] = hostValue
					}
				} else {
					// Handle OS profile format and other formats
					// For OS profile format: "Name:               | Edge Microvisor Toolkit"
					result[key] = value
				}
			}
		} else {
			// Handle section headers (lines ending with ":")
			if strings.HasSuffix(line, ":") && !strings.Contains(line, "|") {
				result[line] = ""
			} else {
				// Handle standalone values (like memory values or table data)
				// Check if this looks like a numeric value or table data
				if strings.TrimSpace(line) != "" {
					// For cases like "Total (GB)" or "16" - treat as key with empty value
					result[line] = ""
				}
			}
		}
	}

	return result
}

func mapVerboseCliOutput(output string) map[string]map[string]string {
	retval := make(map[string]map[string]string)
	lines := strings.Split(output, "\n")

	newOne := true
	key := ""

	for _, line := range lines {
		if line == "" {
			newOne = true
			continue
		}
		fields := strings.SplitN(line, ":", 2)
		value := strings.TrimSpace(fields[1])
		if newOne {
			newOne = false
			key = value
			retval[key] = make(map[string]string)
		}
		retval[key][fields[0]] = value
	}
	return retval
}

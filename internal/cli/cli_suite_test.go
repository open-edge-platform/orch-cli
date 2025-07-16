// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/open-edge-platform/app-orch-catalog/pkg/restClient"
	restproxy "github.com/open-edge-platform/app-orch-catalog/pkg/restProxy"
	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	infra "github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/open-edge-platform/orch-library/go/pkg/openidconnect"
	"github.com/spf13/cobra"
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

type CLITestSuite struct {
	suite.Suite
	proxy restproxy.MockRestProxy
}

func (s *CLITestSuite) SetupSuite() {
	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")
	viper.Set(auth.TrustCertField, "")

	// In your SetupSuite method, replace the existing timestamp line with:
	timestamp, _ := time.Parse(time.RFC3339, "2025-01-15T10:30:00Z")
	// Helper function to create timestamp pointers
	timestampPtr := func(t time.Time) *infra.GoogleProtobufTimestamp {
		return (*infra.GoogleProtobufTimestamp)(&t)
	}

	mctrl := gomock.NewController(s.T())

	kcTokenEndpoint := fmt.Sprintf("%s/protocol/openid-connect/token", kcTest)

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "u",
		"typ":      "Refresh",
		"azp":      "system-client",
		"iss":      kcTest,
		"nbf":      time.Now(),
	})

	rt, err := token.SignedString([]byte("test-key"))
	s.NoError(err)

	auth.KeycloakFactory = func(ctx context.Context, _ string) (openidconnect.ClientWithResponsesInterface, error) {
		mockClient := openidconnect.NewMockClientWithResponsesInterface(mctrl)

		mockClient.EXPECT().GetWellKnownOpenidConfigurationWithResponse(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ ...openidconnect.RequestEditorFn) (*openidconnect.GetWellKnownOpenidConfigurationResponse, error) {
				return &openidconnect.GetWellKnownOpenidConfigurationResponse{
					JSON200: &openidconnect.WellKnownResponse{
						TokenEndpoint: &kcTokenEndpoint,
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().PostProtocolOpenidConnectTokenWithFormdataBodyWithResponse(gomock.Any(), auth.GrantTypeMatcher{GrantType: "password"}, gomock.Any()).DoAndReturn(
			func(_ context.Context, body openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody, _ ...openidconnect.RequestEditorFn) (*openidconnect.PostProtocolOpenidConnectTokenResponse, error) {
				s.NotNil(body.Username)
				s.NotNil(body.Password)
				s.NotNil(body.ClientId)
				s.Nil(body.RefreshToken)

				resp := new(openidconnect.PostProtocolOpenidConnectTokenResponse)
				resp.HTTPResponse = &http.Response{
					StatusCode: 200,
					Status:     "OK",
				}
				at := "test access token after login"
				expireSec := 60
				tokenResponse := openidconnect.TokenResponse{
					AccessToken:      &at,
					DeviceSecret:     nil,
					ExpiresIn:        &expireSec,
					IdToken:          nil,
					RefreshExpiresIn: &expireSec,
					RefreshToken:     &rt,
					Scope:            nil,
					TokenType:        nil,
				}
				resp.JSON200 = &tokenResponse

				return resp, nil
			}).AnyTimes()

		mockClient.EXPECT().PostProtocolOpenidConnectTokenWithFormdataBodyWithResponse(gomock.Any(), auth.GrantTypeMatcher{GrantType: "refresh_token"}, gomock.Any()).DoAndReturn(
			func(_ context.Context, body openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody, _ ...openidconnect.RequestEditorFn) (*openidconnect.PostProtocolOpenidConnectTokenResponse, error) {
				s.Nil(body.Username)
				s.Nil(body.Password)
				s.NotNil(body.ClientId)
				s.NotNil(body.RefreshToken)

				resp := new(openidconnect.PostProtocolOpenidConnectTokenResponse)
				resp.HTTPResponse = &http.Response{
					StatusCode: 200,
					Status:     "OK",
				}
				at := "test access token after refresh"
				expireSec := 60
				tokenResponse := openidconnect.TokenResponse{
					AccessToken:      &at,
					DeviceSecret:     nil,
					ExpiresIn:        &expireSec,
					IdToken:          nil,
					RefreshExpiresIn: &expireSec,
					RefreshToken:     &rt,
					Scope:            nil,
					TokenType:        nil,
				}
				resp.JSON200 = &tokenResponse

				return resp, nil
			}).AnyTimes()

		return mockClient, nil
	}

	// Mock the CatalogFactory
	CatalogFactory = func(cmd *cobra.Command) (context.Context, catapi.ClientWithResponsesInterface, string, error) {
		mockClient := catapi.NewMockClientWithResponsesInterface(mctrl)

		// Mock ListRegistries
		mockClient.EXPECT().CatalogServiceListRegistriesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *catapi.CatalogServiceListRegistriesParams, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceListRegistriesResponse, error) {
				stringPtr := func(s string) *string { return &s }
				resp := &catapi.CatalogServiceListRegistriesResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.ListRegistriesResponse{
						Registries: []catapi.Registry{
							{
								Name:        "test-registry",
								DisplayName: stringPtr("Test Registry"),
								Description: stringPtr("Test registry description"),
								Type:        "HELM",
								RootUrl:     "https://test-registry.example.com",
							},
						},
					},
				}
				return resp, nil
			},
		).AnyTimes()

		// Mock GetRegistry
		mockClient.EXPECT().CatalogServiceGetRegistryWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, name string, params *catapi.CatalogServiceGetRegistryParams, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetRegistryResponse, error) {
				stringPtr := func(s string) *string { return &s }
				resp := &catapi.CatalogServiceGetRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.GetRegistryResponse{
						Registry: catapi.Registry{
							Name:        "test-registry",
							DisplayName: stringPtr("Test Registry"),
							Description: stringPtr("Test registry description"),
							Type:        "HELM",
							RootUrl:     "https://test-registry.example.com",
						},
					},
				}
				return resp, nil
			},
		).AnyTimes()

		// Mock ListRegistries
		mockClient.EXPECT().CatalogServiceListRegistriesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *catapi.CatalogServiceListRegistriesParams, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceListRegistriesResponse, error) {
				stringPtr := func(s string) *string { return &s }
				resp := &catapi.CatalogServiceListRegistriesResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.ListRegistriesResponse{
						Registries: []catapi.Registry{
							{
								Name:        "test-registry",
								DisplayName: stringPtr("Test Registry"),
								Description: stringPtr("Test registry description"),
								Type:        "HELM",
								RootUrl:     "https://test-registry.example.com",
							},
						},
					},
				}
				return resp, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceCreateRegistryWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body interface{}, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateRegistryResponse, error) {
				stringPtr := func(s string) *string { return &s }
				resp := &catapi.CatalogServiceCreateRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200: &catapi.CreateRegistryResponse{
						Registry: catapi.Registry{
							Name:        "test-registry",
							DisplayName: stringPtr("Test Registry"),
							Description: stringPtr("Test registry description"),
							Type:        "HELM",
							RootUrl:     "https://test-registry.example.com",
						},
					},
				}
				return resp, nil
			},
		).AnyTimes()
		// Add more methods as needed:
		// - CatalogServiceCreateRegistryWithResponse
		// - CatalogServiceUpdateRegistryWithResponse
		// - CatalogServiceDeleteRegistryWithResponse

		ctx := context.Background()
		projectName := "test-project"
		return ctx, mockClient, projectName, nil
	}

	// Mock the InfraFactory
	InfraFactory = func(cmd *cobra.Command) (context.Context, infra.ClientWithResponsesInterface, string, error) {
		mockInfraClient := infra.NewMockClientWithResponsesInterface(mctrl)

		// Helper function for string pointers
		stringPtr := func(s string) *string { return &s }

		// Get the project name from the command flags
		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project" // Default fallback
		}

		// Mock ListOperatingSystems (used by list, get, create, delete commands)
		mockInfraClient.EXPECT().OperatingSystemServiceListOperatingSystemsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.OperatingSystemServiceListOperatingSystemsParams, reqEditors ...infra.RequestEditorFn) (*infra.OperatingSystemServiceListOperatingSystemsResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &infra.OperatingSystemServiceListOperatingSystemsResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.OperatingSystemServiceListOperatingSystemsResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListOperatingSystemsResponse{
							OperatingSystemResources: []infra.OperatingSystemResource{
								{
									Name:              stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
									Architecture:      stringPtr("x86_64"),
									SecurityFeature:   (*infra.SecurityFeature)(stringPtr("SECURITY_FEATURE_NONE")),
									ProfileName:       stringPtr("microvisor-nonrt"),
									RepoUrl:           stringPtr("files-edge-orch/repository/microvisor/non_rt/"),
									OsResourceID:      stringPtr("test-os-resource-id"),
									ImageId:           stringPtr("3.0.20250504"),
									ImageUrl:          stringPtr("files-edge-orch/repository/microvisor/non_rt/artifact.raw.gz"),
									OsType:            (*infra.OsType)(stringPtr("OPERATING_SYSTEM_TYPE_IMMUTABLE")),
									OsProvider:        (*infra.OsProviderKind)(stringPtr("OPERATING_SYSTEM_PROVIDER_INFRA")),
									PlatformBundle:    stringPtr(""),
									Sha256:            "abc123def456",
									ProfileVersion:    stringPtr("3.0.20250504"),
									KernelCommand:     stringPtr("console=ttyS0, root=/dev/sda1"),
									UpdateSources:     &[]string{"https://updates.example.com"},
									InstalledPackages: stringPtr("wget\ncurl\nvim"),
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
								},
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock CreateOperatingSystem (used by create command)
		mockInfraClient.EXPECT().OperatingSystemServiceCreateOperatingSystemWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.OperatingSystemServiceCreateOperatingSystemJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.OperatingSystemServiceCreateOperatingSystemResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.OperatingSystemServiceCreateOperatingSystemResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil

				default:
					return &infra.OperatingSystemServiceCreateOperatingSystemResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "Created"},
						JSON200: &infra.OperatingSystemResource{
							Name:            body.Name,
							Architecture:    body.Architecture,
							SecurityFeature: body.SecurityFeature,
							ProfileName:     body.ProfileName,
							RepoUrl:         body.RepoUrl,
							OsResourceID:    stringPtr("test-os-resource-id-new"),
							ImageId:         body.ImageId,
							ImageUrl:        body.ImageUrl,
							OsType:          body.OsType,
							OsProvider:      body.OsProvider,
							PlatformBundle:  stringPtr(""),
							Sha256:          body.Sha256,
							ProfileVersion:  stringPtr("1.0.0"),
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock DeleteOperatingSystem (used by delete command)
		mockInfraClient.EXPECT().OperatingSystemServiceDeleteOperatingSystemWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, osResourceId string, reqEditors ...infra.RequestEditorFn) (*infra.OperatingSystemServiceDeleteOperatingSystemResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.OperatingSystemServiceDeleteOperatingSystemResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.OperatingSystemServiceDeleteOperatingSystemResponse{
						HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
					}, nil
				}
			},
		).AnyTimes()

		// Add more infrastructure service mocks as needed:
		// - Host management operations
		// - Network operations
		// - Storage operations
		// etc.

		ctx := context.Background()
		return ctx, mockInfraClient, projectName, nil
	}
}

func (s *CLITestSuite) TearDownSuite() {
	auth.KeycloakFactory = nil
	CatalogFactory = nil
	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")
	viper.Set(auth.TrustCertField, "")
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
	viper.Set(auth.TrustCertField, "")
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
			matches, _ := regexp.MatchString(v, actualMap[k])
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
			matches, _ := regexp.MatchString(v, actualRow[k])
			s.True(matches, "Row %d field %s: expected '%s' to match '%s'", i, k, actualRow[k], v)
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
		commandString = commandString + fmt.Sprintf(` --%s %s `, argName, argValue)
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

func mapListOutput(output string) []map[string]string {
	var retval []map[string]string
	lines := strings.Split(output, "\n")
	var headers []string
	headerFound := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		if !headerFound {
			// First non-empty line is the headers
			headers = strings.Split(line, "|")
			// Clean up headers
			for j := range headers {
				headers[j] = strings.TrimSpace(headers[j])
			}
			headerFound = true
		} else {
			// Split data line by | to match headers
			fields := strings.Split(line, "|")

			// Clean up fields
			for j := range fields {
				fields[j] = strings.TrimSpace(fields[j])
			}

			if len(fields) == 0 {
				continue
			}

			row := make(map[string]string)

			// Only process fields that have corresponding headers
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

			retval = append(retval, row)
		}
	}
	return retval
}

func mapGetOutput(output string) map[string]string {
	result := make(map[string]string)
	lines := strings.Split(output, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Split by the first | character
		parts := strings.SplitN(line, "|", 2)
		if len(parts) == 2 {
			key := strings.TrimSpace(parts[0])
			value := strings.TrimSpace(parts[1])

			// Remove trailing colon from key
			key = strings.TrimSuffix(key, ":")

			// Remove quotes from value if present
			if strings.HasPrefix(value, `"`) && strings.HasSuffix(value, `"`) {
				value = strings.Trim(value, `"`)
			}

			result[key] = value
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

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
	cluster "github.com/open-edge-platform/cli/pkg/rest/cluster"
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

		// Mock CustomConfigServiceListCustomConfigsWithResponse (used by list custom configs command)
		mockInfraClient.EXPECT().CustomConfigServiceListCustomConfigsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.CustomConfigServiceListCustomConfigsParams, reqEditors ...infra.RequestEditorFn) (*infra.CustomConfigServiceListCustomConfigsResponse, error) {
				switch projectName {
				case "nonexistent-project", "nonexistent-init":
					return &infra.CustomConfigServiceListCustomConfigsResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.NotFound
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.CustomConfigServiceListCustomConfigsResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListCustomConfigsResponse{
							CustomConfigs: []infra.CustomConfigResource{
								{
									Name:        "nginx-config",
									Config:      "test:",
									Description: stringPtr("Nginx configuration for web services"),
									ResourceId:  stringPtr("config-abc12345"),
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
								},
							},
							HasNext:       false,
							TotalElements: 1,
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock CustomConfigServiceCreateCustomConfigWithResponse (used by create custom config command)
		mockInfraClient.EXPECT().CustomConfigServiceCreateCustomConfigWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.CustomConfigServiceCreateCustomConfigJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.CustomConfigServiceCreateCustomConfigResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.CustomConfigServiceCreateCustomConfigResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "duplicate-config-project":
					return &infra.CustomConfigServiceCreateCustomConfigResponse{
						HTTPResponse: &http.Response{StatusCode: 409, Status: "Conflict"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Custom config with same name already exists"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.AlreadyExists
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.CustomConfigServiceCreateCustomConfigResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.CustomConfigResource{
							Name:        body.Name,
							Config:      body.Config,
							Description: body.Description,
							ResourceId:  stringPtr("config-abc12345"),
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock CustomConfigServiceDeleteCustomConfigWithResponse (used by delete custom config command)
		mockInfraClient.EXPECT().CustomConfigServiceDeleteCustomConfigWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, configName string, reqEditors ...infra.RequestEditorFn) (*infra.CustomConfigServiceDeleteCustomConfigResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.CustomConfigServiceDeleteCustomConfigResponse{
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
					switch configName {
					case "nonexistent-config", "invalid-config-name":
						return &infra.CustomConfigServiceDeleteCustomConfigResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("Custom config not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.CustomConfigServiceDeleteCustomConfigResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock LocalAccountServiceListLocalAccountsWithResponse (used by list local accounts command)
		mockInfraClient.EXPECT().LocalAccountServiceListLocalAccountsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.LocalAccountServiceListLocalAccountsParams, reqEditors ...infra.RequestEditorFn) (*infra.LocalAccountServiceListLocalAccountsResponse, error) {
				switch projectName {
				case "nonexistent-project", "nonexistent-user":
					return &infra.LocalAccountServiceListLocalAccountsResponse{
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
					return &infra.LocalAccountServiceListLocalAccountsResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListLocalAccountsResponse{
							LocalAccounts: []infra.LocalAccountResource{
								{
									ResourceId:     stringPtr("account-abc12345"),
									LocalAccountID: stringPtr("account-abc12345"), // Deprecated alias
									Username:       "admin",
									SshKey:         "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC7... admin@example.com",
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
								},
							},
							TotalElements: 1,
							HasNext:       false,
						},
					}, nil
				}
			},
		).AnyTimes()

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
									OsResourceID:      stringPtr("os-1234abcd"),
									ResourceId:        stringPtr("os-1234abcd"),
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
							OsResourceID:    stringPtr("os-1234abcd"),
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

		// Mock ListHosts (used by list, get, create, delete commands)
		mockInfraClient.EXPECT().HostServiceListHostsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.HostServiceListHostsParams, reqEditors ...infra.RequestEditorFn) (*infra.HostServiceListHostsResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &infra.HostServiceListHostsResponse{
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
					return &infra.HostServiceListHostsResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListHostsResponse{
							Hosts: []infra.HostResource{
								{
									ResourceId:      stringPtr("host-abc12345"),
									Name:            "edge-host-001",
									Hostname:        stringPtr("edge-host-001.example.com"),
									Note:            stringPtr("Edge computing host"),
									CpuArchitecture: stringPtr("x86_64"),
									CpuCores:        (*int)(func() *int { i := 8; return &i }()),
									CpuModel:        stringPtr("Intel(R) Xeon(R) CPU E5-2670 v3"),
									CpuSockets:      (*int)(func() *int { i := 2; return &i }()),
									CpuThreads:      (*int)(func() *int { i := 32; return &i }()),
									MemoryBytes:     stringPtr("17179869184"), // 16GB in bytes
									SerialNumber:    stringPtr("1234567890"),
									Uuid:            stringPtr("550e8400-e29b-41d4-a716-446655440000"),
									ProductName:     stringPtr("ThinkSystem SR650"),
									BiosVendor:      stringPtr("Lenovo"),
									BiosVersion:     stringPtr("TEE142L-2.61"),
									BiosReleaseDate: stringPtr("03/25/2023"),
									BmcIp:           stringPtr("192.168.1.101"),
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
									Instance: &infra.InstanceResource{
										ResourceId: stringPtr("instance-abcd1234"),
										Name:       stringPtr("edge-instance-001"),
										HostID:     stringPtr("host-abc12345"),
										InstanceID: stringPtr("instance-abcd1234"),
										WorkloadMembers: &[]infra.WorkloadMember{
											{
												ResourceId:       stringPtr("workload-abcd1234"),
												WorkloadId:       stringPtr("workload-abcd1234"),
												InstanceId:       stringPtr("instance-abc12345"),
												WorkloadMemberId: stringPtr("workload-abcd1234"),
												Kind:             infra.WORKLOADMEMBERKINDCLUSTERNODE,
												Workload: &infra.WorkloadResource{
													ResourceId: stringPtr("workload-abcd1234"),
													WorkloadId: stringPtr("workload-abcd1234"),
													Name:       stringPtr("Edge Kubernetes Cluster"),
													Kind:       infra.WORKLOADKINDCLUSTER,
													Status:     stringPtr("Running"),
													ExternalId: stringPtr("k8s-cluster-east-001"),
													Timestamps: &infra.Timestamps{
														CreatedAt: timestampPtr(timestamp),
														UpdatedAt: timestampPtr(timestamp),
													},
												},
												Timestamps: &infra.Timestamps{
													CreatedAt: timestampPtr(timestamp),
													UpdatedAt: timestampPtr(timestamp),
												},
											},
										},
										CustomConfig: &[]infra.CustomConfigResource{
											{
												Name:        "nginx-config",
												Config:      "server {\n    listen 80;\n    server_name example.com;\n    location / {\n        proxy_pass http://backend;\n    }\n}",
												Description: stringPtr("Nginx configuration for web services"),
												ResourceId:  stringPtr("config-abc12345"),
												Timestamps: &infra.Timestamps{
													CreatedAt: timestampPtr(timestamp),
													UpdatedAt: timestampPtr(timestamp),
												},
											},
										},
										Os: &infra.OperatingSystemResource{
											Name: stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
										},
										CurrentOs: &infra.OperatingSystemResource{
											Name: stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
										},
									},
								},
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock CreateHost (used by create command)
		mockInfraClient.EXPECT().HostServiceCreateHostWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.HostServiceCreateHostJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.HostServiceCreateHostResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.HostServiceCreateHostResponse{
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
					return &infra.HostServiceCreateHostResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.HostResource{
							ResourceId:                  stringPtr("host-new-12345"),
							Name:                        body.Name,
							Hostname:                    body.Hostname,
							Note:                        body.Note,
							CpuArchitecture:             body.CpuArchitecture,
							CpuCores:                    body.CpuCores,
							CpuModel:                    body.CpuModel,
							CpuSockets:                  body.CpuSockets,
							CpuThreads:                  body.CpuThreads,
							CpuCapabilities:             body.CpuCapabilities,
							CpuTopology:                 body.CpuTopology,
							MemoryBytes:                 body.MemoryBytes,
							SerialNumber:                body.SerialNumber,
							Uuid:                        body.Uuid,
							ProductName:                 body.ProductName,
							BiosVendor:                  body.BiosVendor,
							BiosVersion:                 body.BiosVersion,
							BiosReleaseDate:             body.BiosReleaseDate,
							BmcIp:                       body.BmcIp,
							BmcKind:                     body.BmcKind,
							CurrentState:                (*infra.HostState)(stringPtr("HOST_STATE_ONBOARDING")),
							CurrentPowerState:           (*infra.PowerState)(stringPtr("POWER_STATE_OFF")),
							CurrentAmtState:             (*infra.AmtState)(stringPtr("AMT_STATE_UNKNOWN")),
							HostStatus:                  stringPtr("Provisioning"),
							HostStatusIndicator:         (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_WORKING")),
							OnboardingStatus:            stringPtr("Onboarding in progress"),
							OnboardingStatusIndicator:   (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_WORKING")),
							PowerStatus:                 stringPtr("Powered off"),
							PowerStatusIndicator:        (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							RegistrationStatus:          stringPtr("Registering"),
							RegistrationStatusIndicator: (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_WORKING")),
							SiteId:                      body.SiteId,
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock DeleteHost (used by delete command)
		mockInfraClient.EXPECT().HostServiceDeleteHostWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, hostId string, reqEditors ...infra.RequestEditorFn) (*infra.HostServiceDeleteHostResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.HostServiceDeleteHostResponse{
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
					return &infra.HostServiceDeleteHostResponse{
						HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
					}, nil
				}
			},
		).AnyTimes()

		// Mock GetHost (used by get command)
		mockInfraClient.EXPECT().HostServiceGetHostWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, hostId string, reqEditors ...infra.RequestEditorFn) (*infra.HostServiceGetHostResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.HostServiceGetHostResponse{
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
					switch hostId {
					case "host-11111111", "non-existent-host", "invalid-host-id":
						return &infra.HostServiceGetHostResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("Host not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.HostServiceGetHostResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.HostResource{
								ResourceId:                  stringPtr(hostId),
								Name:                        "edge-host-001",
								Hostname:                    stringPtr("edge-host-001.example.com"),
								Note:                        stringPtr("Edge computing host"),
								CpuArchitecture:             stringPtr("x86_64"),
								CpuCores:                    (*int)(func() *int { i := 8; return &i }()),
								CpuModel:                    stringPtr("Intel(R) Xeon(R) CPU E5-2670 v3"),
								CpuSockets:                  (*int)(func() *int { i := 2; return &i }()),
								CpuThreads:                  (*int)(func() *int { i := 32; return &i }()),
								MemoryBytes:                 stringPtr("17179869184"), // 16GB in bytes
								SerialNumber:                stringPtr("1234567890"),  // Match ListHosts
								Uuid:                        stringPtr("550e8400-e29b-41d4-a716-446655440000"),
								ProductName:                 stringPtr("ThinkSystem SR650"),
								BiosVendor:                  stringPtr("Lenovo"),
								BiosVersion:                 stringPtr("TEE142L-2.61"),
								BiosReleaseDate:             stringPtr("03/25/2023"),
								BmcIp:                       stringPtr("192.168.1.101"),
								BmcKind:                     (*infra.BaremetalControllerKind)(stringPtr("BAREMETAL_CONTROLLER_KIND_IPMI")),
								CurrentState:                (*infra.HostState)(stringPtr("HOST_STATE_ONBOARDED")),
								CurrentPowerState:           (*infra.PowerState)(stringPtr("POWER_STATE_ON")),
								CurrentAmtState:             (*infra.AmtState)(stringPtr("AMT_STATE_PROVISIONED")),
								HostStatus:                  stringPtr("Running"),
								HostStatusIndicator:         (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								OnboardingStatus:            stringPtr("Onboarded successfully"),
								OnboardingStatusIndicator:   (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								PowerStatus:                 stringPtr("Powered on"),
								PowerStatusIndicator:        (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								RegistrationStatus:          stringPtr("Registered"),
								RegistrationStatusIndicator: (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								SiteId:                      stringPtr("site-abc123"),
								Instance: &infra.InstanceResource{
									ResourceId: stringPtr("instance-abcd1234"),
									InstanceID: stringPtr("instance-abcd1234"),
								},
								Timestamps: &infra.Timestamps{
									CreatedAt: timestampPtr(timestamp),
									UpdatedAt: timestampPtr(timestamp),
								},
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock RegisterHost (used by create command)
		mockInfraClient.EXPECT().HostServiceRegisterHostWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.HostServiceRegisterHostJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.HostServiceRegisterHostResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.HostServiceRegisterHostResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "duplicate-host-project":
					// Simulate FailedPrecondition error for duplicate host registration
					return &infra.HostServiceRegisterHostResponse{
						HTTPResponse: &http.Response{StatusCode: 409, Status: "Conflict"},
						Body:         []byte(`{"code":"FailedPrecondition","message":"Host with same serial number and UUID already exists"}`),
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Host with same serial number and UUID already exists"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.FailedPrecondition
								return &code
							}(),
						},
					}, nil
				default:
					// Generate a new host ID based on serial number or UUID
					hostID := "host-1111abcd"
					hostName := hostID
					if body.Name != nil && *body.Name != "" {
						hostName = *body.Name
					}

					return &infra.HostServiceRegisterHostResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.HostResource{
							ResourceId:                  stringPtr(hostID),
							Name:                        hostName,
							SerialNumber:                body.SerialNumber,
							Uuid:                        body.Uuid,
							CurrentState:                (*infra.HostState)(stringPtr("HOST_STATE_REGISTERED")),
							HostStatus:                  stringPtr("registered"),
							HostStatusIndicator:         (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							OnboardingStatus:            stringPtr("Registered successfully"),
							OnboardingStatusIndicator:   (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							PowerStatus:                 stringPtr("Unknown"),
							PowerStatusIndicator:        (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							RegistrationStatus:          stringPtr("Registered"),
							RegistrationStatusIndicator: (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							CurrentPowerState:           (*infra.PowerState)(stringPtr("POWER_STATE_UNKNOWN")),
							CurrentAmtState:             (*infra.AmtState)(stringPtr("AMT_STATE_UNKNOWN")),
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock InvalidateHost (used by invalidate command)
		mockInfraClient.EXPECT().HostServiceInvalidateHostWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, hostId string, params *infra.HostServiceInvalidateHostParams, reqEditors ...infra.RequestEditorFn) (*infra.HostServiceInvalidateHostResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.HostServiceInvalidateHostResponse{
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
					switch hostId {
					case "host-11111111", "non-existent-host", "invalid-host-id":
						return &infra.HostServiceInvalidateHostResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("Host not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.HostServiceInvalidateHostResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock PatchHost (used by patch command)
		mockInfraClient.EXPECT().HostServicePatchHostWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, hostId string, body infra.HostServicePatchHostJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.HostServicePatchHostResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.HostServicePatchHostResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "host-not-found-project":
					return &infra.HostServicePatchHostResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Host not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.NotFound
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.HostServicePatchHostResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.HostResource{
							ResourceId: stringPtr(hostId),
							Name: func() string {
								if body.Name != "" {
									return body.Name
								}
								return "edge-host-001"
							}(),
							Hostname: func() *string {
								if body.Hostname != nil {
									return body.Hostname
								}
								return stringPtr("edge-host-001.example.com")
							}(),
							Note: func() *string {
								if body.Note != nil {
									return body.Note
								}
								return stringPtr("Edge computing host")
							}(),
							CpuArchitecture: func() *string {
								if body.CpuArchitecture != nil {
									return body.CpuArchitecture
								}
								return stringPtr("x86_64")
							}(),
							CpuCores: func() *int {
								if body.CpuCores != nil {
									return body.CpuCores
								}
								i := 8
								return &i
							}(),
							CpuModel: func() *string {
								if body.CpuModel != nil {
									return body.CpuModel
								}
								return stringPtr("Intel(R) Xeon(R) CPU E5-2670 v3")
							}(),
							CpuSockets: func() *int {
								if body.CpuSockets != nil {
									return body.CpuSockets
								}
								i := 2
								return &i
							}(),
							CpuThreads: func() *int {
								if body.CpuThreads != nil {
									return body.CpuThreads
								}
								i := 32
								return &i
							}(),
							CpuCapabilities: body.CpuCapabilities,
							CpuTopology:     body.CpuTopology,
							MemoryBytes: func() *string {
								if body.MemoryBytes != nil {
									return body.MemoryBytes
								}
								return stringPtr("17179869184")
							}(),
							SerialNumber: func() *string {
								if body.SerialNumber != nil {
									return body.SerialNumber
								}
								return stringPtr("SN123456789")
							}(),
							Uuid: func() *string {
								if body.Uuid != nil {
									return body.Uuid
								}
								return stringPtr("550e8400-e29b-41d4-a716-446655440000")
							}(),
							ProductName: func() *string {
								if body.ProductName != nil {
									return body.ProductName
								}
								return stringPtr("ThinkSystem SR650")
							}(),
							BiosVendor: func() *string {
								if body.BiosVendor != nil {
									return body.BiosVendor
								}
								return stringPtr("Lenovo")
							}(),
							BiosVersion: func() *string {
								if body.BiosVersion != nil {
									return body.BiosVersion
								}
								return stringPtr("TEE142L-2.61")
							}(),
							BiosReleaseDate: func() *string {
								if body.BiosReleaseDate != nil {
									return body.BiosReleaseDate
								}
								return stringPtr("03/25/2023")
							}(),
							BmcIp: func() *string {
								if body.BmcIp != nil {
									return body.BmcIp
								}
								return stringPtr("192.168.1.101")
							}(),
							BmcKind: func() *infra.BaremetalControllerKind {
								if body.BmcKind != nil {
									return body.BmcKind
								}
								return (*infra.BaremetalControllerKind)(stringPtr("BAREMETAL_CONTROLLER_KIND_IPMI"))
							}(),

							// System-managed fields (not patchable)
							CurrentState:      (*infra.HostState)(stringPtr("HOST_STATE_ONBOARDED")),
							CurrentPowerState: (*infra.PowerState)(stringPtr("POWER_STATE_ON")),
							CurrentAmtState:   (*infra.AmtState)(stringPtr("AMT_STATE_PROVISIONED")),

							// User-controlled desired states
							DesiredState: func() *infra.HostState {
								if body.DesiredState != nil {
									return body.DesiredState
								}
								return (*infra.HostState)(stringPtr("HOST_STATE_ONBOARDED"))
							}(),
							DesiredPowerState: func() *infra.PowerState {
								if body.DesiredPowerState != nil {
									return body.DesiredPowerState
								}
								return (*infra.PowerState)(stringPtr("POWER_STATE_ON"))
							}(),
							DesiredAmtState: func() *infra.AmtState {
								if body.DesiredAmtState != nil {
									return body.DesiredAmtState
								}
								return (*infra.AmtState)(stringPtr("AMT_STATE_PROVISIONED"))
							}(),

							// Status fields (system-managed)
							HostStatus:                  stringPtr("Running"),
							HostStatusIndicator:         (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							OnboardingStatus:            stringPtr("Onboarded successfully"),
							OnboardingStatusIndicator:   (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							PowerStatus:                 stringPtr("Powered on"),
							PowerStatusIndicator:        (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
							RegistrationStatus:          stringPtr("Registered"),
							RegistrationStatusIndicator: (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),

							// User-controlled fields
							PowerCommandPolicy: body.PowerCommandPolicy,
							SiteId: func() *string {
								if body.SiteId != nil {
									return body.SiteId
								}
								return stringPtr("site-abc123")
							}(),
							Metadata: body.Metadata,

							// Timestamps
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock ListSites (used by list, get, create, delete commands)
		mockInfraClient.EXPECT().SiteServiceListSitesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, resourceId string, params *infra.SiteServiceListSitesParams, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceListSitesResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &infra.SiteServiceListSitesResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "nonexistent-site":
					return &infra.SiteServiceListSitesResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListSitesResponse{
							Sites:         []infra.SiteResource{},
							TotalElements: 0,
						},
					}, nil
				default:
					return &infra.SiteServiceListSitesResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListSitesResponse{
							Sites: []infra.SiteResource{
								{
									ResourceId: stringPtr("site-7ceae560"),
									SiteID:     stringPtr("site-7ceae560"), // Deprecated alias
									Name:       stringPtr("Edge Site East"),
									RegionId:   stringPtr("region-abcd1234"),
									SiteLat:    (*int32)(func() *int32 { lat := int32(404783900); return &lat }()),  // 40.4783900° N (NYC) in E7 format
									SiteLng:    (*int32)(func() *int32 { lng := int32(-740020000); return &lng }()), // -74.0020000° W (NYC) in E7 format
									Metadata: &[]infra.MetadataItem{
										{Key: "environment", Value: "production"},
										{Key: "datacenter", Value: "nyc-east-1"},
									},
									InheritedMetadata: &[]infra.MetadataItem{
										{Key: "region", Value: "us-east"},
										{Key: "zone", Value: "east-coast"},
									},
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
								},
							},
							TotalElements: 1,
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock CreateSite (used by create command)
		mockInfraClient.EXPECT().SiteServiceCreateSiteWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.SiteServiceCreateSiteJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceCreateSiteResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.SiteServiceCreateSiteResponse{
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
					return &infra.SiteServiceCreateSiteResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.SiteResource{
							ResourceId:        stringPtr("site-new-456"),
							SiteID:            stringPtr("site-new-456"), // Deprecated alias
							Name:              body.Name,
							RegionId:          body.RegionId,
							SiteLat:           body.SiteLat,
							SiteLng:           body.SiteLng,
							Metadata:          body.Metadata,
							InheritedMetadata: body.InheritedMetadata,
							Provider:          body.Provider,
							Region:            body.Region,
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock DeleteSite (used by delete command)
		mockInfraClient.EXPECT().SiteServiceDeleteSiteWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, siteId string, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceDeleteSiteResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.SiteServiceDeleteSiteResponse{
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
					return &infra.SiteServiceDeleteSiteResponse{
						HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
					}, nil
				}
			},
		).AnyTimes()

		// Mock GetSite (used by get command)
		mockInfraClient.EXPECT().SiteServiceGetSiteWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, region string, siteId string, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceGetSiteResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.SiteServiceGetSiteResponse{
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
					return &infra.SiteServiceGetSiteResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.SiteResource{
							ResourceId: stringPtr(siteId),
							SiteID:     stringPtr(siteId), // Deprecated alias
							Name:       stringPtr("Edge Site East"),
							RegionId:   stringPtr("region-abcd1111"),
							SiteLat:    (*int32)(func() *int32 { lat := int32(404783900); return &lat }()),  // 40.4783900° N (NYC) in E7 format
							SiteLng:    (*int32)(func() *int32 { lng := int32(-740020000); return &lng }()), // -74.0020000° W (NYC) in E7 format
							Metadata: &[]infra.MetadataItem{
								{Key: "environment", Value: "production"},
								{Key: "datacenter", Value: "nyc-east-1"},
							},
							InheritedMetadata: &[]infra.MetadataItem{
								{Key: "region", Value: "us-east"},
								{Key: "zone", Value: "east-coast"},
							},
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock UpdateSite (used by update command)
		mockInfraClient.EXPECT().SiteServiceUpdateSiteWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, siteId string, body infra.SiteServiceUpdateSiteJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceUpdateSiteResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.SiteServiceUpdateSiteResponse{
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
					return &infra.SiteServiceUpdateSiteResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.SiteResource{
							ResourceId:        stringPtr(siteId),
							SiteID:            stringPtr(siteId), // Deprecated alias
							Name:              body.Name,
							RegionId:          body.RegionId,
							SiteLat:           body.SiteLat,
							SiteLng:           body.SiteLng,
							Metadata:          body.Metadata,
							InheritedMetadata: body.InheritedMetadata,
							Provider:          body.Provider,
							Region:            body.Region,
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock CreateInstance (used by create command)
		mockInfraClient.EXPECT().InstanceServiceCreateInstanceWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.InstanceServiceCreateInstanceJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.InstanceServiceCreateInstanceResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.InstanceServiceCreateInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "invalid-host-project":
					return &infra.InstanceServiceCreateInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Host not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.NotFound
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.InstanceServiceCreateInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.InstanceResource{
							ResourceId:   stringPtr("instance-abcd1234"),
							Name:         body.Name,
							CurrentState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_PROVISIONING")),
							DesiredState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
							Kind:         (*infra.InstanceKind)(stringPtr("INSTANCE_KIND_OPERATING_SYSTEM")),
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock DeleteInstance (used by delete command)
		mockInfraClient.EXPECT().InstanceServiceDeleteInstanceWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, instanceId string, reqEditors ...infra.RequestEditorFn) (*infra.InstanceServiceDeleteInstanceResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.InstanceServiceDeleteInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "instance-not-found-project":
					return &infra.InstanceServiceDeleteInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Instance not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.NotFound
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.InstanceServiceDeleteInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
					}, nil
				}
			},
		).AnyTimes()

		// Mock GetInstance (used by get command) - Optional but helpful for completeness
		mockInfraClient.EXPECT().InstanceServiceGetInstanceWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, instanceId string, reqEditors ...infra.RequestEditorFn) (*infra.InstanceServiceGetInstanceResponse, error) {
				switch projectName {
				case "invalid-project", "invalid-instance":
					return &infra.InstanceServiceGetInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "instance-not-found-project":
					return &infra.InstanceServiceGetInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Instance not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.NotFound
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.InstanceServiceGetInstanceResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.InstanceResource{
							ResourceId:   stringPtr(instanceId),
							Name:         stringPtr("edge-instance-001"),
							CurrentState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
							DesiredState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
							Kind:         (*infra.InstanceKind)(stringPtr("INSTANCE_KIND_OPERATING_SYSTEM")),
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
							CustomConfig: &[]infra.CustomConfigResource{
								{
									Name:        "nginx-config",
									Config:      "server {\n    listen 80;\n    server_name example.com;\n    location / {\n        proxy_pass http://backend;\n    }\n}",
									Description: stringPtr("Nginx configuration for web services"),
									ResourceId:  stringPtr("config-abc12345"),
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
								},
							},
							Os: &infra.OperatingSystemResource{
								Name: stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
							},
							CurrentOs: &infra.OperatingSystemResource{
								Name: stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock ListInstances (used by list command) - Optional but helpful for completeness
		mockInfraClient.EXPECT().InstanceServiceListInstancesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.InstanceServiceListInstancesParams, reqEditors ...infra.RequestEditorFn) (*infra.InstanceServiceListInstancesResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &infra.InstanceServiceListInstancesResponse{
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
					return &infra.InstanceServiceListInstancesResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListInstancesResponse{
							Instances: []infra.InstanceResource{
								{
									ResourceId:   stringPtr("instance-abcd1234"),
									InstanceID:   stringPtr("instance-abcd1234"),
									Name:         stringPtr("edge-instance-001"),
									CurrentState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
									DesiredState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
									Kind:         (*infra.InstanceKind)(stringPtr("INSTANCE_KIND_OPERATING_SYSTEM")),
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
									WorkloadMembers: &[]infra.WorkloadMember{
										{
											ResourceId:       stringPtr("workload-abcd1234"),
											WorkloadId:       stringPtr("workload-abcd1234"),
											InstanceId:       stringPtr("instance-abc12345"),
											WorkloadMemberId: stringPtr("workload-abcd1234"),
											Kind:             infra.WORKLOADMEMBERKINDCLUSTERNODE,
											Workload: &infra.WorkloadResource{
												ResourceId: stringPtr("workload-abcd1234"),
												WorkloadId: stringPtr("workload-abcd1234"),
												Name:       stringPtr("Edge Kubernetes Cluster"),
												Kind:       infra.WORKLOADKINDCLUSTER,
												Status:     stringPtr("Running"),
												ExternalId: stringPtr("k8s-cluster-east-001"),
												Timestamps: &infra.Timestamps{
													CreatedAt: timestampPtr(timestamp),
													UpdatedAt: timestampPtr(timestamp),
												},
											},
											Timestamps: &infra.Timestamps{
												CreatedAt: timestampPtr(timestamp),
												UpdatedAt: timestampPtr(timestamp),
											},
										},
									},
									CustomConfig: &[]infra.CustomConfigResource{
										{
											Name:        "nginx-config",
											Config:      "server {\n    listen 80;\n    server_name example.com;\n    location / {\n        proxy_pass http://backend;\n    }\n}",
											Description: stringPtr("Nginx configuration for web services"),
											ResourceId:  stringPtr("config-abc12345"),
											Timestamps: &infra.Timestamps{
												CreatedAt: timestampPtr(timestamp),
												UpdatedAt: timestampPtr(timestamp),
											},
										},
									},
									Os: &infra.OperatingSystemResource{
										Name: stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
									},
									CurrentOs: &infra.OperatingSystemResource{
										Name: stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
									},
								},
								{
									ResourceId:   stringPtr("instance-abcd5678"),
									Name:         stringPtr("edge-instance-002"),
									CurrentState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
									DesiredState: (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
									Kind:         (*infra.InstanceKind)(stringPtr("INSTANCE_KIND_OPERATING_SYSTEM")),
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

		ctx := context.Background()
		return ctx, mockInfraClient, projectName, nil
	}

	// Mock the ClusterFactory
	ClusterFactory = func(cmd *cobra.Command) (context.Context, cluster.ClientWithResponsesInterface, string, error) {
		mockClusterClient := cluster.NewMockClientWithResponsesInterface(mctrl)

		// Helper function for string pointers
		stringPtr := func(s string) *string { return &s }

		// Get the project name from the command flags
		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project" // Default fallback
		}

		// Mock GetV2ProjectsProjectNameTemplatesNameVersionsVersionWithResponse (used by get template command)
		mockClusterClient.EXPECT().GetV2ProjectsProjectNameTemplatesNameVersionsVersionWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, templateName, version string, reqEditors ...cluster.RequestEditorFn) (*cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse, error) {
				fmt.Printf("The name of the template is %s", templateName)
				switch projectName {
				case "nonexistent-project":
					return &cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Not Found"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Project not found"),
						},
					}, nil
				default:
					switch templateName {
					case "nonexistent-template":
						return &cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse{
							HTTPResponse: &http.Response{StatusCode: 500, Status: "Not Found"},
							JSON500: &cluster.ProblemDetails{
								Message: stringPtr("Template not found"),
							},
						}, nil
					default:
						return &cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &cluster.TemplateInfo{
								Name:    templateName,
								Version: version,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock GetV2ProjectsProjectNameTemplatesWithResponse (used by list templates command)
		mockClusterClient.EXPECT().GetV2ProjectsProjectNameTemplatesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *cluster.GetV2ProjectsProjectNameTemplatesParams, reqEditors ...cluster.RequestEditorFn) (*cluster.GetV2ProjectsProjectNameTemplatesResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &cluster.GetV2ProjectsProjectNameTemplatesResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Project not found"),
						},
					}, nil
				default:
					return &cluster.GetV2ProjectsProjectNameTemplatesResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &cluster.TemplateInfoList{
							TemplateInfoList: &[]cluster.TemplateInfo{
								{
									Name:              "default-template",
									Version:           "v1.0.0",
									KubernetesVersion: "v1.28.0",
									Description:       stringPtr("Default Kubernetes cluster template"),
								},
								{
									Name:              "ha-template",
									Version:           "v1.1.0",
									KubernetesVersion: "v1.28.0",
									Description:       stringPtr("High availability cluster template"),
								},
							},
							TotalElements: func() *int32 { count := int32(2); return &count }(),
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock PostV2ProjectsProjectNameClustersWithResponse (used by create cluster command)
		mockClusterClient.EXPECT().PostV2ProjectsProjectNameClustersWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body cluster.PostV2ProjectsProjectNameClustersJSONRequestBody, reqEditors ...cluster.RequestEditorFn) (*cluster.PostV2ProjectsProjectNameClustersResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &cluster.PostV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Not Found"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Project not found"),
						},
					}, nil
				case "duplicate-cluster-project":
					return &cluster.PostV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Conflict"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Cluster with same name already exists"),
						},
					}, nil
				default:
					return &cluster.PostV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON201:      stringPtr("cluster-12345"), // Return cluster ID as string
					}, nil
				}
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockClusterClient, projectName, nil
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

		// Handle lines that contain pipe separators
		if strings.Contains(line, "|") {
			parts := strings.Split(line, "|")
			if len(parts) >= 2 {
				key := strings.TrimSpace(parts[0])
				value := strings.TrimSpace(parts[1])

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
						hostValue := strings.TrimSpace(contentParts[1])
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

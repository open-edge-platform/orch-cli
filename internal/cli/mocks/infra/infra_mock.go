// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package infra

import (
	"context"
	"net/http"
	"time"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"

	infra "github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

// Return the mock factory function instead of assigning directly
func CreateInfraMock(mctrl *gomock.Controller, timestamp time.Time) interfaces.InfraFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, infra.ClientWithResponsesInterface, string, error) {
		mockInfraClient := infra.NewMockClientWithResponsesInterface(mctrl)

		timestampPtr := func(t time.Time) *infra.GoogleProtobufTimestamp {
			return &t
		}

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
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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

		mockInfraClient.EXPECT().LocalAccountServiceCreateLocalAccountWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, body infra.LocalAccountServiceCreateLocalAccountJSONRequestBody, _ ...infra.RequestEditorFn) (*infra.LocalAccountServiceCreateLocalAccountResponse, error) {
				return &infra.LocalAccountServiceCreateLocalAccountResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200: &infra.LocalAccountResource{
						ResourceId: body.ResourceId,
						Username:   body.Username,
						SshKey:     body.SshKey,
						Timestamps: &infra.Timestamps{ /* fill as needed */ },
					},
				}, nil
			},
		).AnyTimes()

		// Mock LocalAccountServiceGetLocalAccountWithResponse (used by get local account command)
		mockInfraClient.EXPECT().LocalAccountServiceGetLocalAccountWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, accountID string, _ ...infra.RequestEditorFn) (*infra.LocalAccountServiceGetLocalAccountResponse, error) {
				return &infra.LocalAccountServiceGetLocalAccountResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &infra.LocalAccountResource{
						ResourceId: &accountID,
						Username:   "admin",
						SshKey:     "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIEK8F2qJ5K8F2qJ5K8F2qJ5K8F2qJ5K8F2qJ5K8F2qJ5 testkey@example.com",
						Timestamps: &infra.Timestamps{},
					},
				}, nil
			},
		).AnyTimes()

		// Mock LocalAccountServiceDeleteLocalAccountWithResponse (used by delete local account command)
		mockInfraClient.EXPECT().LocalAccountServiceDeleteLocalAccountWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ string, _ ...infra.RequestEditorFn) (*infra.LocalAccountServiceDeleteLocalAccountResponse, error) {
				return &infra.LocalAccountServiceDeleteLocalAccountResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		// Mock LocalAccountServiceListLocalAccountsWithResponse (used by list local accounts command)
		mockInfraClient.EXPECT().LocalAccountServiceListLocalAccountsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.LocalAccountServiceListLocalAccountsParams, reqEditors ...infra.RequestEditorFn) (*infra.LocalAccountServiceListLocalAccountsResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
									ResourceId:     stringPtr("localaccount-abc12345"),
									LocalAccountID: stringPtr("localaccount-abc12345"), // Deprecated alias
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
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
									Description:       stringPtr(""),
									Metadata:          stringPtr(""),
									Timestamps: &infra.Timestamps{
										CreatedAt: timestampPtr(timestamp),
										UpdatedAt: timestampPtr(timestamp),
									},
									ExistingCves: stringPtr(`[{"cve_id":"CVE-2021-1234","priority":"HIGH","affected_packages":["fluent-bit-3.1.9-11.emt3.x86_64"]}]`),
									FixedCves:    stringPtr(`[{"cve_id":"CVE-2021-5678","priority":"MEDIUM","affected_packages":["curl-7.68.0-1ubuntu2.24"]}]`),
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
				_ = ctx          // Acknowledge we're not using it
				_ = reqEditors   // Acknowledge we're not using it
				_ = osResourceId // Acknowledge we're not using it
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
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
									HostStatus:      stringPtr("Running"),
									ResourceId:      stringPtr("host-abc12345"),
									Name:            "edge-host-001",
									Hostname:        stringPtr("edge-host-001.example.com"),
									Note:            stringPtr("Edge computing host"),
									CpuArchitecture: stringPtr("x86_64"),
									CpuCores:        func() *int { i := 8; return &i }(),
									CpuModel:        stringPtr("Intel(R) Xeon(R) CPU E5-2670 v3"),
									CpuSockets:      func() *int { i := 2; return &i }(),
									CpuThreads:      func() *int { i := 32; return &i }(),
									MemoryBytes:     stringPtr("17179869184"), // 16GB in bytes
									SerialNumber:    stringPtr("1234567890"),
									Uuid:            stringPtr("550e8400-e29b-41d4-a716-446655440000"),
									ProductName:     stringPtr("ThinkSystem SR650"),
									BiosVendor:      stringPtr("Lenovo"),
									BiosVersion:     stringPtr("TEE142L-2.61"),
									BiosReleaseDate: stringPtr("03/25/2023"),
									BmcIp:           stringPtr("192.168.1.101"),
									SiteId:          stringPtr("site-abcd1234"),
									Site: &infra.SiteResource{
										ResourceId: stringPtr("site-abcd1234"),
										Name:       stringPtr("site"),
										Timestamps: &infra.Timestamps{
											CreatedAt: timestampPtr(timestamp),
											UpdatedAt: timestampPtr(timestamp),
										},
									},
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
										ProvisioningStatus: stringPtr("PROVISIONING_STATUS_COMPLETED"),
									},
								},
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		mockInfraClient.EXPECT().InstanceServicePatchInstanceWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ string, _ infra.InstanceServicePatchInstanceJSONRequestBody, _ ...infra.RequestEditorFn) (*infra.InstanceServicePatchInstanceResponse, error) {
				return &infra.InstanceServicePatchInstanceResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					// Add JSON200 or other fields if your code expects them
				}, nil
			},
		).AnyTimes()

		// Mock CreateHost (used by create command)
		mockInfraClient.EXPECT().HostServiceCreateHostWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.HostServiceCreateHostJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.HostServiceCreateHostResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
							DesiredAmtState:             (*infra.AmtState)(stringPtr("AMT_STATE_UNKNOWN")),
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				_ = hostId     // Acknowledge we're not using it
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				_ = hostId     // Acknowledge we're not using it
				stamp := 1

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
					case "host-abcd1000":
						return &infra.HostServiceGetHostResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSON200: &infra.HostResource{
								ResourceId: stringPtr(hostId),
								Name:       "edge-host-002",
								Hostname:   stringPtr("edge-host-002.example.com"),
								Instance: &infra.InstanceResource{
									ResourceId: stringPtr("instance-abcd1234"),
									InstanceID: stringPtr("instance-abcd1234"),
									UpdatePolicy: &infra.OSUpdatePolicy{
										ResourceId: stringPtr("updatepolicy-abc12345"),
									},
								},
							},
						}, nil
					case "host-abcd1001":
						return &infra.HostServiceGetHostResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.HostResource{
								ResourceId: stringPtr(hostId),
								Name:       "edge-host-002",
								Hostname:   stringPtr("edge-host-002.example.com"),
								Instance: &infra.InstanceResource{
									ResourceId: stringPtr("instance-abcd1234"),
									InstanceID: stringPtr("instance-abcd1234"),
								},
							},
						}, nil
					case "host-abcd1002":
						return &infra.HostServiceGetHostResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.HostResource{
								ResourceId: stringPtr(hostId),
								Name:       "edge-host-002",
								Hostname:   stringPtr("edge-host-002.example.com"),
							},
						}, nil
					default:
						return &infra.HostServiceGetHostResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.HostResource{
								ResourceId:         stringPtr(hostId),
								Name:               "edge-host-001",
								Hostname:           stringPtr("edge-host-001.example.com"),
								Note:               stringPtr("Edge computing host"),
								CpuArchitecture:    stringPtr("x86_64"),
								CpuCores:           func() *int { i := 8; return &i }(),
								CpuModel:           stringPtr("Intel(R) Xeon(R) CPU E5-2670 v3"),
								CpuSockets:         func() *int { i := 2; return &i }(),
								CpuThreads:         func() *int { i := 32; return &i }(),
								MemoryBytes:        stringPtr("17179869184"), // 16GB in bytes
								SerialNumber:       stringPtr("1234567890"),  // Match ListHosts
								Uuid:               stringPtr("550e8400-e29b-41d4-a716-446655440000"),
								ProductName:        stringPtr("ThinkSystem SR650"),
								BiosVendor:         stringPtr("Lenovo"),
								BiosVersion:        stringPtr("TEE142L-2.61"),
								BiosReleaseDate:    stringPtr("03/25/2023"),
								BmcIp:              stringPtr("192.168.1.101"),
								BmcKind:            (*infra.BaremetalControllerKind)(stringPtr("BAREMETAL_CONTROLLER_KIND_IPMI")),
								CurrentState:       (*infra.HostState)(stringPtr("HOST_STATE_ONBOARDED")),
								CurrentPowerState:  (*infra.PowerState)(stringPtr("POWER_STATE_ON")),
								CurrentAmtState:    (*infra.AmtState)(stringPtr("AMT_STATE_PROVISIONED")),
								DesiredAmtState:    (*infra.AmtState)(stringPtr("AMT_STATE_PROVISIONED")),
								DesiredPowerState:  (*infra.PowerState)(stringPtr("POWER_STATE_ON")),
								PowerCommandPolicy: (*infra.PowerCommandPolicy)(stringPtr("POWER_COMMAND_POLICY_ALWAYS_ON")),
								PowerOnTime:        &stamp,
								HostNics: &[]infra.HostnicResource{
									{
										DeviceName: stringPtr("eth0"),
										Ipaddresses: &[]infra.IPAddressResource{
											{
												Address: stringPtr("192.168.1.102"),
											},
										},
										Mtu:           func() *int { i := 1500; return &i }(),
										MacAddr:       stringPtr("30:d0:42:d9:02:7c"),
										PciIdentifier: stringPtr("0000:19:00.0"),
										SriovEnabled:  func() *bool { i := true; return &i }(),
										SriovVfsNum:   func() *int { i := 4; return &i }(),
										SriovVfsTotal: func() *int { i := 8; return &i }(),
										BmcInterface:  func() *bool { i := true; return &i }(),
										LinkState: &infra.NetworkInterfaceLinkState{
											Type: func() *infra.LinkState { t := infra.NETWORKINTERFACELINKSTATEUNSPECIFIED; return &t }(),
										},
									},
								},
								HostGpus: &[]infra.HostgpuResource{
									{
										DeviceName: stringPtr("TestGPU"),
										Vendor:     stringPtr("TestVendor"),
										Capabilities: &[]string{
											"cap1",
											"cap2",
										},
										PciId: stringPtr("03:00.0"),
									},
								},
								HostStorages: &[]infra.HoststorageResource{
									{
										Wwid:          stringPtr("abcd"),
										CapacityBytes: stringPtr("200000"),
										Model:         stringPtr("Model1"),
										Serial:        stringPtr("123456"),
										Vendor:        stringPtr("Vendor1"),
									},
								},
								HostUsbs: &[]infra.HostusbResource{
									{
										Class:     stringPtr("Hub"),
										Serial:    stringPtr("123456"),
										IdVendor:  stringPtr("abcd"),
										IdProduct: stringPtr("1234"),
										Bus:       func() *int { i := 8; return &i }(),
										Addr:      func() *int { i := 1; return &i }(),
									},
								},
								HostStatus:                  stringPtr("Running"),
								HostStatusIndicator:         (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								OnboardingStatus:            stringPtr("Onboarded successfully"),
								OnboardingStatusIndicator:   (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								PowerStatus:                 stringPtr("Powered on"),
								PowerStatusIndicator:        (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								RegistrationStatus:          stringPtr("Registered"),
								RegistrationStatusIndicator: (*infra.StatusIndication)(stringPtr("STATUS_INDICATION_IDLE")),
								SiteId:                      stringPtr("site-abc123"),
								UserLvmSize:                 func() *int { i := 10; return &i }(), // 10GB in bytes
								Instance: &infra.InstanceResource{
									ResourceId: stringPtr("instance-abcd1234"),
									InstanceID: stringPtr("instance-abcd1234"),
									UpdatePolicy: &infra.OSUpdatePolicy{
										ResourceId: stringPtr("updatepolicy-abc12345"),
									},
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
							DesiredAmtState:             (*infra.AmtState)(stringPtr("AMT_STATE_UNKNOWN")),
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
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				_ = resourceId // Acknowledge we're not using it
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
									Name:       stringPtr("site"),
									RegionId:   stringPtr("region-abcd1234"),
									SiteLat:    func() *int32 { lng := int32(50000000); return &lng }(),
									SiteLng:    func() *int32 { lng := int32(50000000); return &lng }(),
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
									Region: &infra.RegionResource{
										ResourceId: stringPtr("region-abcd1234"),
										RegionID:   stringPtr("region-abcd1234"), // Deprecated alias
										Name:       stringPtr("region"),
										ParentId:   stringPtr(""),
										TotalSites: func() *int32 { i := int32(1); return &i }(),
									},
								},
							},
							TotalElements: 1,
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock DeleteSite (used by delete command)
		mockInfraClient.EXPECT().SiteServiceDeleteSiteWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, regionId, siteId string, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceDeleteSiteResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				_ = regionId   // Acknowledge we're not using it
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
					switch siteId {
					case "nonexistent-site", "invalid-site-id":
						return &infra.SiteServiceDeleteSiteResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("Site not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.SiteServiceDeleteSiteResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock CreateSite (used by create command)
		mockInfraClient.EXPECT().SiteServiceCreateSiteWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, regionId string, body infra.SiteServiceCreateSiteJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceCreateSiteResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				_ = regionId   // Acknowledge we're not using it

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
							ResourceId:        stringPtr("site-abcd1111"),
							SiteID:            stringPtr("site-abcd1111"), // Deprecated alias
							Name:              body.Name,
							RegionId:          body.RegionId,
							SiteLat:           body.SiteLat,
							SiteLng:           body.SiteLng,
							Metadata:          body.Metadata,
							InheritedMetadata: body.InheritedMetadata,
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()
		// Mock GetSite (used by get command)
		mockInfraClient.EXPECT().SiteServiceGetSiteWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, region string, siteId string, reqEditors ...infra.RequestEditorFn) (*infra.SiteServiceGetSiteResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				_ = region     // Acknowledge we're not using it

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
							Name:       stringPtr("site"),
							RegionId:   stringPtr("region-abcd1234"),
							SiteLat:    func() *int32 { lng := int32(50000000); return &lng }(),
							SiteLng:    func() *int32 { lng := int32(50000000); return &lng }(),
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
							Region: &infra.RegionResource{
								ResourceId: stringPtr("region-abcd1234"),
								RegionID:   stringPtr("region-abcd1234"), // Deprecated alias
								Name:       stringPtr("region"),
								ParentId:   stringPtr(""),
								TotalSites: func() *int32 { i := int32(1); return &i }(),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock RegionServiceCreateRegionWithResponse (used by create region command)
		mockInfraClient.EXPECT().RegionServiceCreateRegionWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.RegionServiceCreateRegionJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.RegionServiceCreateRegionResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "invalid-project":
					return &infra.RegionServiceCreateRegionResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "invalid-parent-project":
					return &infra.RegionServiceCreateRegionResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Parent region not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.NotFound
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.RegionServiceCreateRegionResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.RegionResource{
							ResourceId: stringPtr("region-abcd1111"),
							RegionID:   stringPtr("region-abcd1111"), // Deprecated alias
							Name:       body.Name,
							ParentId:   body.ParentId,
							Metadata:   body.Metadata,
							InheritedMetadata: &[]infra.MetadataItem{
								{Key: "organization", Value: "acme-corp"},
							},
							TotalSites: func() *int32 { i := int32(0); return &i }(),
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
							ParentRegion: func() *infra.RegionResource {
								if body.ParentId != nil {
									return &infra.RegionResource{
										ResourceId: body.ParentId,
										RegionID:   body.ParentId,
										Name:       stringPtr("Parent Region"),
									}
								}
								return nil
							}(),
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock RegionServiceListRegionsWithResponse (used by list regions command)
		mockInfraClient.EXPECT().RegionServiceListRegionsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.RegionServiceListRegionsParams, reqEditors ...infra.RequestEditorFn) (*infra.RegionServiceListRegionsResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "nonexistent-project":
					return &infra.RegionServiceListRegionsResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: stringPtr("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				case "empty-regions-project":
					return &infra.RegionServiceListRegionsResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListRegionsResponse{
							Regions:       []infra.RegionResource{},
							TotalElements: 0,
						},
					}, nil
				default:
					switch projectName {
					case "parent-region":
						return &infra.RegionServiceListRegionsResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.ListRegionsResponse{
								Regions: []infra.RegionResource{
									{
										ResourceId: stringPtr("region-abcd1111"),
										RegionID:   stringPtr("region-abcd1111"), // Deprecated alias
										Name:       stringPtr("region"),
										ParentId:   stringPtr(""),
										Metadata: &[]infra.MetadataItem{
											{Key: "region", Value: "us-east"},
											{Key: "zone", Value: "east-coast"},
											{Key: "environment", Value: "production"},
										},
										InheritedMetadata: &[]infra.MetadataItem{
											{Key: "organization", Value: "acme-corp"},
											{Key: "datacenter-type", Value: "primary"},
										},
										TotalSites: func() *int32 { i := int32(1); return &i }(),
										Timestamps: &infra.Timestamps{
											CreatedAt: timestampPtr(timestamp),
											UpdatedAt: timestampPtr(timestamp),
										},
										ParentRegion: &infra.RegionResource{
											ResourceId: stringPtr(""),
											RegionID:   stringPtr(""),
											Name:       stringPtr(""),
										},
									},
									{
										ResourceId: stringPtr("region-abcd2222"),
										RegionID:   stringPtr("region-abcd2222"), // Deprecated alias
										Name:       stringPtr("region"),
										ParentId:   stringPtr("region-abcd1111"),
										Metadata: &[]infra.MetadataItem{
											{Key: "region", Value: "us-east"},
											{Key: "zone", Value: "east-coast"},
											{Key: "environment", Value: "production"},
										},
										InheritedMetadata: &[]infra.MetadataItem{
											{Key: "organization", Value: "acme-corp"},
											{Key: "datacenter-type", Value: "primary"},
										},
										TotalSites: func() *int32 { i := int32(1); return &i }(),
										Timestamps: &infra.Timestamps{
											CreatedAt: timestampPtr(timestamp),
											UpdatedAt: timestampPtr(timestamp),
										},
										ParentRegion: &infra.RegionResource{
											ResourceId: stringPtr("region-abcd1111"),
											RegionID:   stringPtr("region-abcd1111"),
											Name:       stringPtr("region"),
										},
									},
								},
								TotalElements: 2,
							},
						}, nil
					default:
						return &infra.RegionServiceListRegionsResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.ListRegionsResponse{
								Regions: []infra.RegionResource{
									{
										ResourceId: stringPtr("region-abcd1111"),
										RegionID:   stringPtr("region-abcd1111"), // Deprecated alias
										Name:       stringPtr("region"),
										ParentId:   stringPtr(""),
										Metadata: &[]infra.MetadataItem{
											{Key: "region", Value: "us-east"},
											{Key: "zone", Value: "east-coast"},
											{Key: "environment", Value: "production"},
										},
										InheritedMetadata: &[]infra.MetadataItem{
											{Key: "organization", Value: "acme-corp"},
											{Key: "datacenter-type", Value: "primary"},
										},
										TotalSites: func() *int32 { i := int32(1); return &i }(),
										Timestamps: &infra.Timestamps{
											CreatedAt: timestampPtr(timestamp),
											UpdatedAt: timestampPtr(timestamp),
										},
										ParentRegion: &infra.RegionResource{
											ResourceId: stringPtr(""),
											RegionID:   stringPtr(""),
											Name:       stringPtr(""),
										},
									},
								},
								TotalElements: 1,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock RegionServiceGetRegionWithResponse (used by get region command)
		mockInfraClient.EXPECT().RegionServiceGetRegionWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, regionId string, reqEditors ...infra.RequestEditorFn) (*infra.RegionServiceGetRegionResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "invalid-project":
					return &infra.RegionServiceGetRegionResponse{
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
					switch regionId {
					case "region-11111111", "invalid-region-id":
						return &infra.RegionServiceGetRegionResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("Region not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.RegionServiceGetRegionResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.RegionResource{
								ResourceId: stringPtr(regionId),
								RegionID:   stringPtr(regionId), // Deprecated alias
								Name:       stringPtr("region"),
								ParentId:   stringPtr("region-abcd1111"),
								Metadata: &[]infra.MetadataItem{
									{Key: "region", Value: "us-east"},
								},
								InheritedMetadata: &[]infra.MetadataItem{
									{Key: "organization", Value: "acme-corp"},
									{Key: "datacenter-type", Value: "primary"},
								},
								TotalSites: func() *int32 { i := int32(1); return &i }(),
								Timestamps: &infra.Timestamps{
									CreatedAt: timestampPtr(timestamp),
									UpdatedAt: timestampPtr(timestamp),
								},
								ParentRegion: &infra.RegionResource{
									ResourceId: stringPtr(""),
									RegionID:   stringPtr(""),
									Name:       stringPtr(""),
								},
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock RegionServiceDeleteRegionWithResponse (used by delete region command)
		mockInfraClient.EXPECT().RegionServiceDeleteRegionWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, regionId string, reqEditors ...infra.RequestEditorFn) (*infra.RegionServiceDeleteRegionResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "invalid-project":
					return &infra.RegionServiceDeleteRegionResponse{
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
					switch regionId {
					case "nonexistent-region", "invalid-region-id":
						return &infra.RegionServiceDeleteRegionResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("Region not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.RegionServiceDeleteRegionResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock CreateInstance (used by create command)
		mockInfraClient.EXPECT().InstanceServiceCreateInstanceWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.InstanceServiceCreateInstanceJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.InstanceServiceCreateInstanceResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				_ = instanceId // Acknowledge we're not using it

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
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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
							ResourceId:           stringPtr(instanceId),
							ProvisioningStatus:   stringPtr("PROVISIONING_STATUS_COMPLETED"),
							InstanceStatusDetail: stringPtr("INSTANCE_STATUS_RUNNING"),
							Name:                 stringPtr("edge-instance-001"),
							CurrentState:         (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
							DesiredState:         (*infra.InstanceState)(stringPtr("INSTANCE_STATE_RUNNING")),
							Kind:                 (*infra.InstanceKind)(stringPtr("INSTANCE_KIND_OPERATING_SYSTEM")),
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
								Name:      stringPtr("Edge Microvisor Toolkit 3.0.20250504"),
								FixedCves: stringPtr(""),
							},
							ExistingCves: stringPtr(`[{"cve_id":"CVE-2021-1234","priority":"HIGH","affected_packages":["fluent-bit-3.1.9-11.emt3.x86_64"]}]`),
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
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
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

		// Mock OSUpdateRunListOSUpdateRunWithResponse (used by list os update runs command)
		mockInfraClient.EXPECT().OSUpdateRunListOSUpdateRunWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.OSUpdateRunListOSUpdateRunParams, reqEditors ...infra.RequestEditorFn) (*infra.OSUpdateRunListOSUpdateRunResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "nonexistent-project":
					return &infra.OSUpdateRunListOSUpdateRunResponse{
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
					return &infra.OSUpdateRunListOSUpdateRunResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListOSUpdateRunResponse{
							OsUpdateRuns: []infra.OSUpdateRun{
								{
									Name:          stringPtr("security-update-jan-2025"),
									ResourceId:    stringPtr("osupdate-run-abc123"),
									Status:        stringPtr("completed"),
									StatusDetails: stringPtr("All updates applied successfully"),
									AppliedPolicy: &infra.OSUpdatePolicy{
										Name:        "security-policy-v1.2",
										Description: stringPtr("Security update policy"),
										// Add other fields as needed based on the actual struct
									},
									Description: stringPtr("Monthly security updates for edge devices"),
									StartTime:   func() *int { t := int(timestamp.Unix()); return &t }(),
									EndTime:     func() *int { t := int(timestamp.Unix()); return &t }(),
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

		// Mock OSUpdateRunGetOSUpdateRunWithResponse (used by get os update run command)
		mockInfraClient.EXPECT().OSUpdateRunGetOSUpdateRunWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, osUpdateRunId string, reqEditors ...infra.RequestEditorFn) (*infra.OSUpdateRunGetOSUpdateRunResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "nonexistent-project":
					return &infra.OSUpdateRunGetOSUpdateRunResponse{
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
					switch osUpdateRunId {
					case "nonexistent-run", "invalid-run-id":
						return &infra.OSUpdateRunGetOSUpdateRunResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("OS Update Run not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.OSUpdateRunGetOSUpdateRunResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.OSUpdateRun{
								Name:          stringPtr("security-update-jan-2025"),
								ResourceId:    stringPtr(osUpdateRunId),
								Status:        stringPtr("completed"),
								StatusDetails: stringPtr("All updates applied successfully"),
								AppliedPolicy: &infra.OSUpdatePolicy{
									Name:        "security-policy-v1.2",
									Description: stringPtr("Monthly security update policy"),
									// Remove Version field if it doesn't exist in OSUpdatePolicy
								},
								Description:     stringPtr("Monthly security updates for edge devices"),
								StartTime:       func() *int { t := int(timestamp.Unix()); return &t }(),
								EndTime:         func() *int { t := int(timestamp.Unix()); return &t }(),
								StatusTimestamp: func() *int { t := int(timestamp.Unix()); return &t }(),
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

		// Mock OSUpdateRunDeleteOSUpdateRunWithResponse (used by delete os update run command)
		mockInfraClient.EXPECT().OSUpdateRunDeleteOSUpdateRunWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, osUpdateRunId string, reqEditors ...infra.RequestEditorFn) (*infra.OSUpdateRunDeleteOSUpdateRunResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "invalid-project":
					return &infra.OSUpdateRunDeleteOSUpdateRunResponse{
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
					switch osUpdateRunId {
					case "nonexistent-run", "invalid-run-id":
						return &infra.OSUpdateRunDeleteOSUpdateRunResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("OS Update Run not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.OSUpdateRunDeleteOSUpdateRunResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock OSUpdatePolicyListOSUpdatePolicyWithResponse (used by list os update policies command)
		mockInfraClient.EXPECT().OSUpdatePolicyListOSUpdatePolicyWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *infra.OSUpdatePolicyListOSUpdatePolicyParams, reqEditors ...infra.RequestEditorFn) (*infra.OSUpdatePolicyListOSUpdatePolicyResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = params     // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "nonexistent-project":
					return &infra.OSUpdatePolicyListOSUpdatePolicyResponse{
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
					return &infra.OSUpdatePolicyListOSUpdatePolicyResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListOSUpdatePolicyResponse{
							OsUpdatePolicies: []infra.OSUpdatePolicy{
								{
									Name:            "security-policy-v1.2", // string, not *string
									ResourceId:      stringPtr("osupdatepolicy-abc12345"),
									Description:     stringPtr("Monthly security update policy"),
									TargetOsId:      stringPtr("os-1234abcd"),
									InstallPackages: stringPtr("curl wget vim"),
									UpdatePolicy:    (*infra.UpdatePolicy)(stringPtr("UPDATE_POLICY_LATEST")),
									UpdateSources:   &[]string{"https://updates.example.com"},
									KernelCommand:   stringPtr("console=ttyS0"),
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

		// Mock OSUpdatePolicyGetOSUpdatePolicyWithResponse (used by get os update policy command)
		mockInfraClient.EXPECT().OSUpdatePolicyGetOSUpdatePolicyWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, policyId string, reqEditors ...infra.RequestEditorFn) (*infra.OSUpdatePolicyGetOSUpdatePolicyResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "nonexistent-project":
					return &infra.OSUpdatePolicyGetOSUpdatePolicyResponse{
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
					switch policyId {
					case "nonexistent-policy", "invalid-policy-id":
						return &infra.OSUpdatePolicyGetOSUpdatePolicyResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("OS Update Policy not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					case "osupdatepolicy-ccccaaaa":
						return &infra.OSUpdatePolicyGetOSUpdatePolicyResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("OS Update Policy not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.OSUpdatePolicyGetOSUpdatePolicyResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.OSUpdatePolicy{
								Name:                "security-policy-v1.2",
								ResourceId:          stringPtr(policyId),
								Description:         stringPtr("Monthly security update policy"),
								TargetOsId:          stringPtr("os-1234abcd"),
								UpdatePackages:      stringPtr("curl wget vim"),
								UpdatePolicy:        (*infra.UpdatePolicy)(stringPtr("UPDATE_POLICY_LATEST")),
								UpdateSources:       &[]string{"https://updates.example.com"},
								UpdateKernelCommand: stringPtr("console=ttyS0"),
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

		// Mock OSUpdatePolicyCreateOSUpdatePolicyWithResponse (used by create os update policy command)
		mockInfraClient.EXPECT().OSUpdatePolicyCreateOSUpdatePolicyWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body infra.OSUpdatePolicyCreateOSUpdatePolicyJSONRequestBody, reqEditors ...infra.RequestEditorFn) (*infra.OSUpdatePolicyCreateOSUpdatePolicyResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "invalid-project":
					return &infra.OSUpdatePolicyCreateOSUpdatePolicyResponse{
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
					return &infra.OSUpdatePolicyCreateOSUpdatePolicyResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.OSUpdatePolicy{
							Name:        body.Name,
							ResourceId:  stringPtr("updatepolicy-abc12345"),
							Description: body.Description,
							TargetOsId:  body.TargetOsId,
							Timestamps: &infra.Timestamps{
								CreatedAt: timestampPtr(timestamp),
								UpdatedAt: timestampPtr(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock OSUpdatePolicyDeleteOSUpdatePolicyWithResponse (used by delete os update policy command)
		mockInfraClient.EXPECT().OSUpdatePolicyDeleteOSUpdatePolicyWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, policyId string, reqEditors ...infra.RequestEditorFn) (*infra.OSUpdatePolicyDeleteOSUpdatePolicyResponse, error) {
				_ = ctx        // Acknowledge we're not using it
				_ = reqEditors // Acknowledge we're not using it
				switch projectName {
				case "invalid-project":
					return &infra.OSUpdatePolicyDeleteOSUpdatePolicyResponse{
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
					switch policyId {
					case "nonexistent-policy", "invalid-policy-id":
						return &infra.OSUpdatePolicyDeleteOSUpdatePolicyResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: stringPtr("OS Update Policy not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.OSUpdatePolicyDeleteOSUpdatePolicyResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ProviderServiceCreateProviderWithResponse (used by create provider command)
		mockInfraClient.EXPECT().ProviderServiceCreateProviderWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, body infra.ProviderServiceCreateProviderJSONRequestBody, _ ...infra.RequestEditorFn) (*infra.ProviderServiceCreateProviderResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ProviderServiceCreateProviderResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.ProviderServiceCreateProviderResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.ProviderResource{
							Name:           body.Name,
							ApiEndpoint:    body.ApiEndpoint,
							ProviderKind:   body.ProviderKind,
							ProviderVendor: body.ProviderVendor,
							ResourceId:     func(s string) *string { return &s }("provider-abc12345"),
							Timestamps: &infra.Timestamps{
								CreatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
								UpdatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock ProviderServiceDeleteProviderWithResponse (used by delete provider command)
		mockInfraClient.EXPECT().ProviderServiceDeleteProviderWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, providerId string, _ ...infra.RequestEditorFn) (*infra.ProviderServiceDeleteProviderResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ProviderServiceDeleteProviderResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					switch providerId {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ProviderServiceDeleteProviderResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Provider not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ProviderServiceDeleteProviderResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ProviderServiceGetProviderWithResponse (used by get provider command)
		mockInfraClient.EXPECT().ProviderServiceGetProviderWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, providerId string, _ ...infra.RequestEditorFn) (*infra.ProviderServiceGetProviderResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ProviderServiceGetProviderResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					switch providerId {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ProviderServiceGetProviderResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Provider not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ProviderServiceGetProviderResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.ProviderResource{
								Name:           "provider",
								ApiEndpoint:    "hello.com",
								ProviderKind:   "PROVIDER_KIND_BAREMETAL",
								Config:         func(s string) *string { return &s }("{\"defaultOs\": \"\", \"autoProvision\": false, \"defaultLocalAccount\": \"\", \"osSecurityFeatureEnable\": false}"),
								ProviderVendor: (*infra.ProviderVendor)(stringPtr("PROVIDER_VENDOR_UNSPECIFIED")),
								ResourceId:     &providerId,
								Timestamps: &infra.Timestamps{
									CreatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
									UpdatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
								},
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ProviderServiceListProvidersWithResponse (used by list provider command)
		mockInfraClient.EXPECT().ProviderServiceListProvidersWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, _ *infra.ProviderServiceListProvidersParams, _ ...infra.RequestEditorFn) (*infra.ProviderServiceListProvidersResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ProviderServiceListProvidersResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.ProviderServiceListProvidersResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListProvidersResponse{
							Providers: []infra.ProviderResource{
								{
									Name:           "provider",
									ApiEndpoint:    "hello.com",
									ProviderKind:   "PROVIDER_KIND_BAREMETAL",
									ProviderVendor: (*infra.ProviderVendor)(stringPtr("PROVIDER_VENDOR_UNSPECIFIED")),
									ResourceId:     func(s string) *string { return &s }("provider-7ceae560"),
									Timestamps: &infra.Timestamps{
										CreatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
										UpdatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
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

		// Mock ScheduleServiceListWithResponse (used by list provider command)
		mockInfraClient.EXPECT().ScheduleServiceListSchedulesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, _ *infra.ScheduleServiceListSchedulesParams, _ ...infra.RequestEditorFn) (*infra.ScheduleServiceListSchedulesResponse, error) {

				name := "schedule"
				rid := "repeatedsche-abcd1234"
				sid := "singlesche-abcd1234"
				site := "site-abcd1234"

				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServiceListSchedulesResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.ScheduleServiceListSchedulesResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &infra.ListSchedulesResponse{
							RepeatedSchedules: []infra.RepeatedScheduleResource{
								{
									CronDayMonth:    "1",
									CronDayWeek:     "1",
									CronHours:       "1",
									CronMinutes:     "1",
									CronMonth:       "1",
									DurationSeconds: 1,
									Name:            &name,
									ResourceId:      &rid,
									ScheduleStatus:  infra.SCHEDULESTATUSMAINTENANCE,
									TargetHostId:    nil,
									TargetRegionId:  nil,
									TargetSiteId:    &site,
									Timestamps: &infra.Timestamps{
										CreatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
										UpdatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
									},
								},
							},
							SingleSchedules: []infra.SingleScheduleResource{
								{
									Name:           &name,
									ResourceId:     &sid,
									ScheduleStatus: infra.SCHEDULESTATUSMAINTENANCE,
									TargetHostId:   nil,
									TargetRegionId: nil,
									TargetSiteId:   &site,
									StartSeconds:   10000,
									Timestamps: &infra.Timestamps{
										CreatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
										UpdatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
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

		// Mock ScheduleServiceCreateRepeatedScheduleWithResponse
		mockInfraClient.EXPECT().ScheduleServiceCreateRepeatedScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, body infra.ScheduleServiceCreateRepeatedScheduleJSONRequestBody, _ ...infra.RequestEditorFn) (*infra.ScheduleServiceCreateRepeatedScheduleResponse, error) {

				rid := "repeatedsche-abcd1234"
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServiceCreateRepeatedScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.ScheduleServiceCreateRepeatedScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.RepeatedScheduleResource{
							CronDayMonth:    "1",
							CronDayWeek:     "1",
							CronHours:       "1",
							CronMinutes:     "1",
							CronMonth:       "1",
							DurationSeconds: 1,
							Name:            body.Name,
							ResourceId:      &rid,
							ScheduleStatus:  infra.SCHEDULESTATUSMAINTENANCE,
							TargetHostId:    nil,
							TargetRegionId:  nil,
							TargetSiteId:    body.TargetSiteId,
							Timestamps: &infra.Timestamps{
								CreatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
								UpdatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock ScheduleServiceCreateSingleScheduleWithResponse
		mockInfraClient.EXPECT().ScheduleServiceCreateSingleScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, body infra.ScheduleServiceCreateSingleScheduleJSONRequestBody, _ ...infra.RequestEditorFn) (*infra.ScheduleServiceCreateSingleScheduleResponse, error) {

				rid := "repeatedsche-abcd1234"
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServiceCreateSingleScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					return &infra.ScheduleServiceCreateSingleScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON200: &infra.SingleScheduleResource{
							StartSeconds:   body.StartSeconds,
							Name:           body.Name,
							ResourceId:     &rid,
							ScheduleStatus: infra.SCHEDULESTATUSMAINTENANCE,
							TargetHostId:   nil,
							TargetRegionId: nil,
							TargetSiteId:   body.TargetSiteId,
							Timestamps: &infra.Timestamps{
								CreatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
								UpdatedAt: func(t time.Time) *infra.GoogleProtobufTimestamp { return &t }(timestamp),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock ScheduleServiceDeleteRepeatedScheduleWithResponse (used by delete provider command)
		mockInfraClient.EXPECT().ScheduleServiceDeleteRepeatedScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, scheduleId string, _ ...infra.RequestEditorFn) (*infra.ScheduleServiceDeleteRepeatedScheduleResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServiceDeleteRepeatedScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					switch scheduleId {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ScheduleServiceDeleteRepeatedScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Schedule not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ScheduleServiceDeleteRepeatedScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ScheduleServiceDeleteSingleScheduleWithResponse (used by delete provider command)
		mockInfraClient.EXPECT().ScheduleServiceDeleteSingleScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, scheduleId string, _ ...infra.RequestEditorFn) (*infra.ScheduleServiceDeleteSingleScheduleResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServiceDeleteSingleScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					switch scheduleId {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ScheduleServiceDeleteSingleScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Schedule not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ScheduleServiceDeleteSingleScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ScheduleServiceGetSingleScheduleWithResponse
		mockInfraClient.EXPECT().ScheduleServiceGetSingleScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, id string, _ ...infra.RequestEditorFn) (*infra.ScheduleServiceGetSingleScheduleResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServiceGetSingleScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					name := "schedule"
					site := "site-abcd1234"
					switch id {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ScheduleServiceGetSingleScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Provider not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ScheduleServiceGetSingleScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.SingleScheduleResource{
								StartSeconds:   1,
								Name:           &name,
								ResourceId:     &id,
								ScheduleStatus: infra.SCHEDULESTATUSMAINTENANCE,
								TargetHostId:   nil,
								TargetRegionId: nil,
								TargetSiteId:   &site,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ScheduleServiceGetRepeatedScheduleWithResponse
		mockInfraClient.EXPECT().ScheduleServiceGetRepeatedScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, id string, _ ...infra.RequestEditorFn) (*infra.ScheduleServiceGetRepeatedScheduleResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServiceGetRepeatedScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					name := "schedule"
					site := "site-abcd1234"
					switch id {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ScheduleServiceGetRepeatedScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Provider not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ScheduleServiceGetRepeatedScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.RepeatedScheduleResource{
								CronDayMonth:    "1",
								CronDayWeek:     "1",
								CronHours:       "1",
								CronMinutes:     "1",
								CronMonth:       "1",
								DurationSeconds: 1,
								Name:            &name,
								ResourceId:      &id,
								ScheduleStatus:  infra.SCHEDULESTATUSMAINTENANCE,
								TargetHostId:    nil,
								TargetRegionId:  nil,
								TargetSiteId:    &site,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ScheduleServicePatchSingleScheduleWithResponse
		mockInfraClient.EXPECT().ScheduleServicePatchSingleScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, id string, body infra.ScheduleServicePatchSingleScheduleJSONRequestBody, _ ...infra.RequestEditorFn) (*infra.ScheduleServicePatchSingleScheduleResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServicePatchSingleScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					switch id {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ScheduleServicePatchSingleScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Provider not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ScheduleServicePatchSingleScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.SingleScheduleResource{
								StartSeconds:   body.StartSeconds,
								Name:           body.Name,
								ScheduleStatus: body.ScheduleStatus,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock ScheduleServicePatchRepeatedScheduleWithResponse
		mockInfraClient.EXPECT().ScheduleServicePatchRepeatedScheduleWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, id string, body infra.ScheduleServicePatchRepeatedScheduleJSONRequestBody, _ ...infra.RequestEditorFn) (*infra.ScheduleServicePatchRepeatedScheduleResponse, error) {
				switch projectName {
				case "invalid-project":
					return &infra.ScheduleServicePatchRepeatedScheduleResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSONDefault: &infra.ConnectError{
							Message: func(s string) *string { return &s }("Project not found"),
							Code: func() *infra.ConnectErrorCode {
								code := infra.Unknown
								return &code
							}(),
						},
					}, nil
				default:
					switch id {
					case "nonexistent-provider", "invalid-provider-id":
						return &infra.ScheduleServicePatchRepeatedScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSONDefault: &infra.ConnectError{
								Message: func(s string) *string { return &s }("Provider not found"),
								Code: func() *infra.ConnectErrorCode {
									code := infra.NotFound
									return &code
								}(),
							},
						}, nil
					default:
						return &infra.ScheduleServicePatchRepeatedScheduleResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &infra.RepeatedScheduleResource{
								CronDayMonth:    body.CronDayMonth,
								CronDayWeek:     body.CronDayWeek,
								CronHours:       body.CronHours,
								CronMinutes:     body.CronMinutes,
								CronMonth:       body.CronMonth,
								DurationSeconds: body.DurationSeconds,
								Name:            body.Name,
								ScheduleStatus:  body.ScheduleStatus,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockInfraClient, projectName, nil
	}
}

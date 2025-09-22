// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package rps

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	rpsapi "github.com/open-edge-platform/cli/pkg/rest/rps"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

// CreateRpsMock creates a mock RPS factory function
func CreateRpsMock(mctrl *gomock.Controller) interfaces.RpsFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, rpsapi.ClientWithResponsesInterface, string, error) {
		mockRpsClient := rpsapi.NewMockClientWithResponsesInterface(mctrl)

		// Helper function for string pointers
		stringPtr := func(s string) *string { return &s }

		// Get the project name from the command flags
		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project" // Default fallback
		}

		// Sample timestamp for testing
		expirationDate := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

		// Mock GetAllDomainsWithResponse (used by list domains command)
		mockRpsClient.EXPECT().GetAllDomainsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, params *rpsapi.GetAllDomainsParams, reqEditors ...rpsapi.RequestEditorFn) (*rpsapi.GetAllDomainsResponse, error) {
				_ = params     // Ignore params for this mock, you can add logic if needed
				_ = reqEditors // Ignore request editors for this mock
				switch projectName {
				case "nonexistent-project":
					return &rpsapi.GetAllDomainsResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON404: &rpsapi.APIResponse{
							Message: stringPtr("Project not found"),
						},
					}, nil
				default:
					// Create the domains slice
					domains := []rpsapi.DomainResponse{
						{
							ProfileName:                   "corporate-domain",
							DomainSuffix:                  "corp.example.com",
							ProvisioningCertStorageFormat: "pfx",
							TenantId:                      "tenant-abc12345",
							Version:                       "1.0.0",
							ExpirationDate:                expirationDate,
						},
					}

					// Create CountDomainResponse wrapper - this is what your client expects
					countResponse := rpsapi.CountDomainResponse{
						Data:       &domains,
						TotalCount: func() *int { count := len(domains); return &count }(),
					}

					// Convert the wrapper to JSON
					responseJSON, err := json.Marshal(countResponse)
					if err != nil {
						return nil, err
					}

					return &rpsapi.GetAllDomainsResponse{
						HTTPResponse: &http.Response{
							StatusCode: 200,
							Status:     "OK",
							Header:     make(http.Header),
						},
						Body: responseJSON, // Return the wrapped response, not the plain array
					}, nil
				}
			},
		).AnyTimes()

		// Mock GetDomainWithResponse (used by get domain command)
		mockRpsClient.EXPECT().GetDomainWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName, profileName string, reqEditors ...rpsapi.RequestEditorFn) (*rpsapi.GetDomainResponse, error) {
				_ = reqEditors // Ignore request editors for this mock
				switch projectName {
				case "nonexistent-project":
					return &rpsapi.GetDomainResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON404: &rpsapi.APIResponse{
							Message: stringPtr("Project not found"),
						},
					}, nil
				default:
					switch profileName {
					case "nonexistent-domain":
						return &rpsapi.GetDomainResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSON404: &rpsapi.APIResponse{
								Message: stringPtr("Domain profile not found"),
							},
						}, nil
					default:
						return &rpsapi.GetDomainResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &rpsapi.DomainResponse{
								ProfileName:                   "corporate-domain",
								DomainSuffix:                  "corp.example.com",
								ProvisioningCertStorageFormat: "pfx",
								TenantId:                      "tenant-abc12345",
								Version:                       "1.0.0",
								ExpirationDate:                expirationDate,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock CreateDomainWithResponse (used by create domain command)
		mockRpsClient.EXPECT().CreateDomainWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, body rpsapi.CreateDomainJSONRequestBody, reqEditors ...rpsapi.RequestEditorFn) (*rpsapi.CreateDomainResponse, error) {
				_ = reqEditors // Ignore request editors for this mock
				switch projectName {
				case "invalid-project":
					return &rpsapi.CreateDomainResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSON500: &rpsapi.APIResponse{
							Message: stringPtr("Project validation failed"),
						},
					}, nil
				case "validation-error-project":
					return &rpsapi.CreateDomainResponse{
						HTTPResponse: &http.Response{StatusCode: 400, Status: "Bad Request"},
						JSON400: &rpsapi.APIResponse{
							Message: stringPtr("Invalid domain configuration"),
						},
					}, nil
				default:
					return &rpsapi.CreateDomainResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON201: &rpsapi.DomainResponse{
							ProfileName:                   body.ProfileName,
							DomainSuffix:                  body.DomainSuffix,
							ProvisioningCertStorageFormat: "pfx",
							TenantId:                      "tenant-new-123",
							Version:                       "1.0.0",
							ExpirationDate:                expirationDate,
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock UpdateDomainSuffixWithResponse (used by update domain command)
		mockRpsClient.EXPECT().UpdateDomainSuffixWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName string, body rpsapi.UpdateDomainSuffixJSONRequestBody, reqEditors ...rpsapi.RequestEditorFn) (*rpsapi.UpdateDomainSuffixResponse, error) {
				_ = reqEditors // Ignore request editors for this mock
				switch projectName {
				case "invalid-project":
					return &rpsapi.UpdateDomainSuffixResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSON500: &rpsapi.APIResponse{
							Message: stringPtr("Project validation failed"),
						},
					}, nil
				case "validation-error-project":
					return &rpsapi.UpdateDomainSuffixResponse{
						HTTPResponse: &http.Response{StatusCode: 400, Status: "Bad Request"},
						JSON400: &rpsapi.APIResponse{
							Message: stringPtr("Invalid domain suffix format"),
						},
					}, nil
				default:
					return &rpsapi.UpdateDomainSuffixResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &rpsapi.DomainResponse{
							ProfileName:                   "corporate-domain",
							DomainSuffix:                  body.DomainSuffix,
							ProvisioningCertStorageFormat: "pfx",
							TenantId:                      "tenant-12345",
							Version:                       "1.0.1", // Increment version for update
							ExpirationDate:                expirationDate,
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock RemoveDomainWithResponse (used by delete domain command)
		mockRpsClient.EXPECT().RemoveDomainWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, projectName, profileName string, reqEditors ...rpsapi.RequestEditorFn) (*rpsapi.RemoveDomainResponse, error) {
				_ = reqEditors // Ignore request editors for this mock
				switch projectName {
				case "invalid-project":
					return &rpsapi.RemoveDomainResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Internal Server Error"},
						JSON500: &rpsapi.APIResponse{
							Message: stringPtr("Project validation failed"),
						},
					}, nil
				default:
					switch profileName {
					case "nonexistent-domain":
						return &rpsapi.RemoveDomainResponse{
							HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
							JSON404: &rpsapi.APIResponse{
								Message: stringPtr("Domain profile not found"),
							},
						}, nil
					default:
						return &rpsapi.RemoveDomainResponse{
							HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
						}, nil
					}
				}
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockRpsClient, projectName, nil
	}
}

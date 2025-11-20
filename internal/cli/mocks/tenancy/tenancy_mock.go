// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package tenancy

import (
	"context"
	"net/http"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	tenancyapi "github.com/open-edge-platform/cli/pkg/rest/tenancy"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

// CreateTenancyMock creates a mock Tenancy factory function
func CreateTenancyMock(mctrl *gomock.Controller) interfaces.TenancyFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, tenancyapi.ClientWithResponsesInterface, error) {
		mockTenancyClient := tenancyapi.NewMockClientWithResponsesInterface(mctrl)

		// Helper function for string pointers
		stringPtr := func(s string) *string { return &s }

		// // Sample timestamp for testing
		// expirationDate := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

		// Mock PUTV1OrgsOrgOrgWithResponse
		mockTenancyClient.EXPECT().PUTV1OrgsOrgOrgWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, org string, _ *tenancyapi.PUTV1OrgsOrgOrgParams, _ tenancyapi.PUTV1OrgsOrgOrgJSONRequestBody, _ ...tenancyapi.RequestEditorFn) (*tenancyapi.PUTV1OrgsOrgOrgResponse, error) {

				switch org {
				case "nonexistent-project", "nonexistent-init":
					return &tenancyapi.PUTV1OrgsOrgOrgResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
					}, nil
				default:
					// Return the correct response type based on the OpenAPI spec
					return &tenancyapi.PUTV1OrgsOrgOrgResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						// No JSON200 field here, as the OpenAPI-generated response expects DefaultResponse, not OrgOrgGet
					}, nil
				}
			},
		).AnyTimes()

		// Mock GETV1OrgsOrgOrgWithResponse
		mockTenancyClient.EXPECT().GETV1OrgsOrgOrgWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, org string, _ ...tenancyapi.RequestEditorFn) (*tenancyapi.GETV1OrgsOrgOrgResponse, error) {
				switch org {
				case "nonexistent-project", "nonexistent-init":
					return &tenancyapi.GETV1OrgsOrgOrgResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
					}, nil
				default:
					return &tenancyapi.GETV1OrgsOrgOrgResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &tenancyapi.OrgOrgGet{
							Spec: &struct {
								Description *string `json:"description,omitempty"`
							}{
								Description: stringPtr("itep"),
							},
							Status: &struct {
								OrgStatus *struct {
									Message         *string `json:"message,omitempty"`
									StatusIndicator *string `json:"statusIndicator,omitempty"`
									TimeStamp       *int64  `json:"timeStamp,omitempty"`
									UID             *string `json:"uID,omitempty"`
								} `json:"orgStatus,omitempty"`
							}{
								OrgStatus: &struct {
									Message         *string `json:"message,omitempty"`
									StatusIndicator *string `json:"statusIndicator,omitempty"`
									TimeStamp       *int64  `json:"timeStamp,omitempty"`
									UID             *string `json:"uID,omitempty"`
								}{
									Message:         stringPtr("Org itep CREATE is complete"),
									StatusIndicator: stringPtr("STATUS_INDICATION_IDLE"),
									TimeStamp:       func() *int64 { t := int64(1640995200); return &t }(),
									UID:             stringPtr("db8d42ad-849d-4626-8dc7-d7955b83e995"),
								},
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock LISTV1OrgsWithResponse
		mockTenancyClient.EXPECT().LISTV1OrgsWithResponse(
			gomock.Any(), gomock.Any(),
		).Return(&tenancyapi.LISTV1OrgsResponse{
			HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
			JSON200: &tenancyapi.OrgOrgList{
				{
					Name: stringPtr("itep"),
					Spec: &struct {
						Description *string `json:"description,omitempty"`
					}{
						Description: stringPtr("itep"),
					},
					Status: &struct {
						OrgStatus *struct {
							Message         *string `json:"message,omitempty"`
							StatusIndicator *string `json:"statusIndicator,omitempty"`
							TimeStamp       *int64  `json:"timeStamp,omitempty"`
							UID             *string `json:"uID,omitempty"`
						} `json:"orgStatus,omitempty"`
					}{
						OrgStatus: &struct {
							Message         *string `json:"message,omitempty"`
							StatusIndicator *string `json:"statusIndicator,omitempty"`
							TimeStamp       *int64  `json:"timeStamp,omitempty"`
							UID             *string `json:"uID,omitempty"`
						}{
							Message:         stringPtr("Org itep CREATE is complete"),
							StatusIndicator: stringPtr("STATUS_INDICATION_IDLE"),
							TimeStamp:       func() *int64 { t := int64(1640995200); return &t }(),
							UID:             stringPtr("db8d42ad-849d-4626-8dc7-d7955b83e995"),
						},
					},
				},
			},
		}, nil).AnyTimes()

		// Mock DELETEV1OrgsOrgOrgWithResponse - ADD THIS MISSING MOCK
		mockTenancyClient.EXPECT().DELETEV1OrgsOrgOrgWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, org string, _ ...tenancyapi.RequestEditorFn) (*tenancyapi.DELETEV1OrgsOrgOrgResponse, error) {
				switch org {
				case "nonexistent-org", "nonexistent-init":
					return &tenancyapi.DELETEV1OrgsOrgOrgResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
					}, nil
				default:
					return &tenancyapi.DELETEV1OrgsOrgOrgResponse{
						HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
					}, nil
				}
			},
		).AnyTimes()

		// Mock PUTV1ProjectsProjectProjectWithResponse
		mockTenancyClient.EXPECT().PUTV1ProjectsProjectProjectWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, project string, _ *tenancyapi.PUTV1ProjectsProjectProjectParams, _ tenancyapi.PUTV1ProjectsProjectProjectJSONRequestBody, _ ...tenancyapi.RequestEditorFn) (*tenancyapi.PUTV1ProjectsProjectProjectResponse, error) {
				switch project {
				case "nonexistent-project", "nonexistent-init":
					return &tenancyapi.PUTV1ProjectsProjectProjectResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
					}, nil
				default:
					return &tenancyapi.PUTV1ProjectsProjectProjectResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					}, nil
				}
			},
		).AnyTimes()

		// Mock LISTV1ProjectsWithResponse
		mockTenancyClient.EXPECT().LISTV1ProjectsWithResponse(
			gomock.Any(), gomock.Any(),
		).Return(&tenancyapi.LISTV1ProjectsResponse{
			HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
			JSON200: &tenancyapi.ProjectProjectList{
				{
					Name: stringPtr("itep"),
					Spec: &struct {
						Description *string `json:"description,omitempty"`
					}{
						Description: stringPtr("itep"),
					},
					Status: &struct {
						ProjectStatus *struct {
							Message         *string `json:"message,omitempty"`
							StatusIndicator *string `json:"statusIndicator,omitempty"`
							TimeStamp       *int64  `json:"timeStamp,omitempty"`
							UID             *string `json:"uID,omitempty"`
						} `json:"projectStatus,omitempty"`
					}{
						ProjectStatus: &struct {
							Message         *string `json:"message,omitempty"`
							StatusIndicator *string `json:"statusIndicator,omitempty"`
							TimeStamp       *int64  `json:"timeStamp,omitempty"`
							UID             *string `json:"uID,omitempty"`
						}{
							Message:         stringPtr("Project itep CREATE is complete"),
							StatusIndicator: stringPtr("STATUS_INDICATION_IDLE"),
							TimeStamp:       func() *int64 { t := int64(1640995200); return &t }(),
							UID:             stringPtr("70883f2f-4bbe-4a67-9eea-1a5824dee549"),
						},
					},
				},
			},
		}, nil).AnyTimes()

		// Mock GETV1ProjectsProjectProjectWithResponse - ADD THIS MISSING MOCK
		mockTenancyClient.EXPECT().GETV1ProjectsProjectProjectWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, project string, _ ...tenancyapi.RequestEditorFn) (*tenancyapi.GETV1ProjectsProjectProjectResponse, error) {
				switch project {
				case "nonexistent-project", "nonexistent-init":
					return &tenancyapi.GETV1ProjectsProjectProjectResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
					}, nil
				default:
					return &tenancyapi.GETV1ProjectsProjectProjectResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &tenancyapi.ProjectProjectGet{
							Spec: &struct {
								Description *string `json:"description,omitempty"`
							}{
								Description: stringPtr("itep"),
							},
							Status: &struct {
								ProjectStatus *struct {
									Message         *string `json:"message,omitempty"`
									StatusIndicator *string `json:"statusIndicator,omitempty"`
									TimeStamp       *int64  `json:"timeStamp,omitempty"`
									UID             *string `json:"uID,omitempty"`
								} `json:"projectStatus,omitempty"`
							}{
								ProjectStatus: &struct {
									Message         *string `json:"message,omitempty"`
									StatusIndicator *string `json:"statusIndicator,omitempty"`
									TimeStamp       *int64  `json:"timeStamp,omitempty"`
									UID             *string `json:"uID,omitempty"`
								}{
									Message:         stringPtr("Project itep CREATE is complete"),
									StatusIndicator: stringPtr("STATUS_INDICATION_IDLE"),
									TimeStamp:       func() *int64 { t := int64(1640995200); return &t }(),
									UID:             stringPtr("70883f2f-4bbe-4a67-9eea-1a5824dee549"),
								},
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock DELETEV1ProjectsProjectProjectWithResponse - ADD THIS MISSING MOCK
		mockTenancyClient.EXPECT().DELETEV1ProjectsProjectProjectWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, project string, _ ...tenancyapi.RequestEditorFn) (*tenancyapi.DELETEV1ProjectsProjectProjectResponse, error) {
				switch project {
				case "nonexistent-project", "nonexistent-init":
					return &tenancyapi.DELETEV1ProjectsProjectProjectResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
					}, nil
				default:
					return &tenancyapi.DELETEV1ProjectsProjectProjectResponse{
						HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
					}, nil
				}
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockTenancyClient, nil
	}
}

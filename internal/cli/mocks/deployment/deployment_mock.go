// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"net/http"
	"time"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	"github.com/open-edge-platform/cli/pkg/rest/deployment"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

func CreateDeploymentMock(mctrl *gomock.Controller) interfaces.DeploymentFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, deployment.ClientWithResponsesInterface, string, error) {
		mockClient := deployment.NewMockClientWithResponsesInterface(mctrl)

		// Helper function for string pointers
		stringPtr := func(s string) *string { return &s }
		timePtr := func(t time.Time) *time.Time { return &t }

		// Sample timestamp for testing
		testTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project"
		}

		// Mock GetDeployment
		mockClient.EXPECT().DeploymentV1DeploymentServiceGetDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ string, _ ...deployment.RequestEditorFn) (*deployment.DeploymentV1DeploymentServiceGetDeploymentResponse, error) {
				summary := &deployment.DeploymentV1Summary{
					Down:    int32Ptr(1),
					Running: int32Ptr(2),
					Total:   int32Ptr(3),
					Type:    stringPtr("apps"),
					Unknown: int32Ptr(0),
				}
				return &deployment.DeploymentV1DeploymentServiceGetDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &deployment.DeploymentV1GetDeploymentResponse{
						Deployment: deployment.DeploymentV1Deployment{

							DeployId:    stringPtr("9d652bb4-2412-4566-89b5-614f22e2a837"),
							Name:        stringPtr("projectName"),
							DisplayName: stringPtr("displayName"),
							ProfileName: stringPtr("profileName"),
							Status: &deployment.DeploymentV1DeploymentStatus{
								State:   statusStatePtr("state"),
								Message: stringPtr("message"),
								Summary: summary,
							},
							CreateTime: timePtr(testTime),
						},
					},
				}, nil
			},
		).AnyTimes()

		// Mock CreateDeployment
		mockClient.EXPECT().DeploymentV1DeploymentServiceCreateDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ deployment.DeploymentV1DeploymentServiceCreateDeploymentJSONRequestBody, _ ...deployment.RequestEditorFn) (*deployment.DeploymentV1DeploymentServiceCreateDeploymentResponse, error) {
				return &deployment.DeploymentV1DeploymentServiceCreateDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200: &deployment.DeploymentV1CreateDeploymentResponse{
						DeploymentId: "id",
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().DeploymentV1DeploymentServiceUpdateDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ string, _ deployment.DeploymentV1DeploymentServiceUpdateDeploymentJSONRequestBody, _ ...deployment.RequestEditorFn) (*deployment.DeploymentV1DeploymentServiceUpdateDeploymentResponse, error) {
				return &deployment.DeploymentV1DeploymentServiceUpdateDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200:      &deployment.DeploymentV1UpdateDeploymentResponse{
						// Add fields as needed for your tests
					},
				}, nil
			},
		).AnyTimes()

		// Mock DeleteDeployment
		mockClient.EXPECT().DeploymentV1DeploymentServiceDeleteDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ string, _ *deployment.DeploymentV1DeploymentServiceDeleteDeploymentParams, _ ...deployment.RequestEditorFn) (*deployment.DeploymentV1DeploymentServiceDeleteDeploymentResponse, error) {
				return &deployment.DeploymentV1DeploymentServiceDeleteDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().DeploymentV1DeploymentServiceListDeploymentsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, params *deployment.DeploymentV1DeploymentServiceListDeploymentsParams, _ ...deployment.RequestEditorFn) (*deployment.DeploymentV1DeploymentServiceListDeploymentsResponse, error) {
				// On paginated follow-up calls (offset>0) return empty page to exercise the pagination loop exit
				if params != nil && params.Offset != nil && *params.Offset > 0 {
					return &deployment.DeploymentV1DeploymentServiceListDeploymentsResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &deployment.DeploymentV1ListDeploymentsResponse{
							Deployments:   []deployment.DeploymentV1Deployment{},
							TotalElements: 2,
						},
					}, nil
				}
				summary := &deployment.DeploymentV1Summary{
					Down:    int32Ptr(1),
					Running: int32Ptr(2),
					Total:   int32Ptr(3),
					Type:    stringPtr("apps"),
					Unknown: int32Ptr(0),
				}
				return &deployment.DeploymentV1DeploymentServiceListDeploymentsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &deployment.DeploymentV1ListDeploymentsResponse{
						Deployments: []deployment.DeploymentV1Deployment{
							{
								DeployId:    stringPtr("deployment-id"),
								Name:        stringPtr("projectName"),
								DisplayName: stringPtr("displayName"),
								ProfileName: stringPtr("profileName"),
								Status: &deployment.DeploymentV1DeploymentStatus{
									State:   statusStatePtr("state"),
									Message: stringPtr("message"),
									Summary: summary,
								},
								CreateTime: timePtr(testTime),
							},
						},
						TotalElements: 2,
					},
				}, nil
			},
		).AnyTimes()

		return context.Background(), mockClient, projectName, nil
	}
}

// Helper functions
func int32Ptr(i int32) *int32 { return &i }
func statusStatePtr(s string) *deployment.DeploymentV1State {
	v := deployment.DeploymentV1State(s)
	return &v
}

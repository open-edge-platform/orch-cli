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
		mockClient.EXPECT().DeploymentServiceGetDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentName string, reqEditors ...deployment.RequestEditorFn) (*deployment.DeploymentServiceGetDeploymentResponse, error) {
				summary := &deployment.Summary{
					Down:    int32Ptr(1),
					Running: int32Ptr(2),
					Total:   int32Ptr(3),
					Type:    stringPtr("apps"),
					Unknown: int32Ptr(0),
				}
				return &deployment.DeploymentServiceGetDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &deployment.GetDeploymentResponse{
						Deployment: deployment.Deployment{

							DeployId:    stringPtr("deployment-id"),
							Name:        stringPtr("projectName"),
							DisplayName: stringPtr("displayName"),
							ProfileName: stringPtr("profileName"),
							Status: &deployment.DeploymentStatus{
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
		mockClient.EXPECT().DeploymentServiceCreateDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body deployment.DeploymentServiceCreateDeploymentJSONRequestBody, reqEditors ...deployment.RequestEditorFn) (*deployment.DeploymentServiceCreateDeploymentResponse, error) {
				return &deployment.DeploymentServiceCreateDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200: &deployment.CreateDeploymentResponse{
						DeploymentId: "id",
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().DeploymentServiceUpdateDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentName string, body deployment.DeploymentServiceUpdateDeploymentJSONRequestBody, reqEditors ...deployment.RequestEditorFn) (*deployment.DeploymentServiceUpdateDeploymentResponse, error) {
				return &deployment.DeploymentServiceUpdateDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200:      &deployment.UpdateDeploymentResponse{
						// Add fields as needed for your tests
					},
				}, nil
			},
		).AnyTimes()

		// Mock DeleteDeployment
		mockClient.EXPECT().DeploymentServiceDeleteDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentName string, params *deployment.DeploymentServiceDeleteDeploymentParams, reqEditors ...deployment.RequestEditorFn) (*deployment.DeploymentServiceDeleteDeploymentResponse, error) {
				return &deployment.DeploymentServiceDeleteDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().DeploymentServiceListDeploymentsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *deployment.DeploymentServiceListDeploymentsParams, reqEditors ...deployment.RequestEditorFn) (*deployment.DeploymentServiceListDeploymentsResponse, error) {
				summary := &deployment.Summary{
					Down:    int32Ptr(1),
					Running: int32Ptr(2),
					Total:   int32Ptr(3),
					Type:    stringPtr("apps"),
					Unknown: int32Ptr(0),
				}
				return &deployment.DeploymentServiceListDeploymentsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &deployment.ListDeploymentsResponse{
						Deployments: []deployment.Deployment{
							{
								DeployId:    stringPtr("deployment-id"),
								Name:        stringPtr("projectName"),
								DisplayName: stringPtr("displayName"),
								ProfileName: stringPtr("profileName"),
								Status: &deployment.DeploymentStatus{
									State:   statusStatePtr("state"),
									Message: stringPtr("message"),
									Summary: summary,
								},
								CreateTime: timePtr(testTime),
							},
						},
					},
				}, nil
			},
		).AnyTimes()

		return context.Background(), mockClient, projectName, nil
	}
}

// Helper functions
func int32Ptr(i int32) *int32 { return &i }
func statusStatePtr(s string) *deployment.DeploymentStatusState {
	v := deployment.DeploymentStatusState(s)
	return &v
}

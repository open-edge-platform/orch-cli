// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package deployment

import (
	"context"
	"net/http"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	"github.com/open-edge-platform/cli/pkg/rest/deployment"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

func CreateDeploymentMock(mctrl *gomock.Controller) interfaces.DeploymentFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, deployment.ClientWithResponsesInterface, string, error) {
		mockClient := deployment.NewMockClientWithResponsesInterface(mctrl)

		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project"
		}

		// Mock GetDeployment
		mockClient.EXPECT().DeploymentServiceGetDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentName string, reqEditors ...deployment.RequestEditorFn) (*deployment.DeploymentServiceGetDeploymentResponse, error) {
				return &deployment.DeploymentServiceGetDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &deployment.GetDeploymentResponse{
						Deployment: deployment.Deployment{
							Name: &deploymentName,
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

		// Mock DeleteDeployment
		mockClient.EXPECT().DeploymentServiceDeleteDeploymentWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentName string, reqEditors ...deployment.RequestEditorFn) (*deployment.DeploymentServiceDeleteDeploymentResponse, error) {
				return &deployment.DeploymentServiceDeleteDeploymentResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		return context.Background(), mockClient, projectName, nil
	}
}

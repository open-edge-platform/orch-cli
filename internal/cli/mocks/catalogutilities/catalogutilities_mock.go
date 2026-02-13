// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package catalogutilities

import (
	"context"
	"fmt"
	"net/http"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	catutilapi "github.com/open-edge-platform/cli/pkg/rest/catalogutilities"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

func CreateCatalogUtilitiesMock(mctrl *gomock.Controller) interfaces.CatalogUtilitiesFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, catutilapi.ClientWithResponsesInterface, string, error) {
		mockClient := catutilapi.NewMockClientWithResponsesInterface(mctrl)

		// Get the project name from the command flags
		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project"
		}

		// Mock CatalogServiceDownloadDeploymentPackageWithResponse
		mockClient.EXPECT().CatalogServiceDownloadDeploymentPackageWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, deploymentPackageName string, version string, _ ...catutilapi.RequestEditorFn) (*catutilapi.CatalogServiceDownloadDeploymentPackageResponse, error) {
				// Create a mock tar.gz content
				mockTarGzContent := []byte("mock-deployment-package-content")

				// Create HTTP response with Content-Disposition header
				httpResp := &http.Response{
					StatusCode: 200,
					Status:     "OK",
					Header:     http.Header{},
				}
				httpResp.Header.Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-%s.tar.gz"`, deploymentPackageName, version))
				httpResp.Header.Set("Content-Type", "application/gzip")

				return &catutilapi.CatalogServiceDownloadDeploymentPackageResponse{
					Body:         mockTarGzContent,
					HTTPResponse: httpResp,
				}, nil
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockClient, projectName, nil
	}
}

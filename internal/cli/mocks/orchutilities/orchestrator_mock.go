// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package orchutilities

import (
	"context"
	"net/http"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	orchapi "github.com/open-edge-platform/cli/pkg/rest/orchutilities"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/mock/gomock"
)

// CreateOrchestratorMock creates a mock Orchestrator factory function
func CreateOrchestratorMock(mctrl *gomock.Controller) interfaces.OrchestratorFactoryFunc {
	return func(_ *cobra.Command) (context.Context, orchapi.ClientWithResponsesInterface, error) {
		mockOrchClient := orchapi.NewMockClientWithResponsesInterface(mctrl)

		// Helper function for string pointers
		stringPtr := func(s string) *string { return &s }
		boolPtr := func(b bool) *bool { return &b }

		// Mock GetOrchestratorInfoWithResponse
		mockOrchClient.EXPECT().GetOrchestratorInfoWithResponse(
			gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ ...orchapi.RequestEditorFn) (*orchapi.InfoResponse, error) {
				var info *orchapi.Info

				// Check viper setting to determine which response to return
				// Tests can set viper.Set("test_orchestrator_404", true) to simulate 404
				if viper.GetBool("test_orchestrator_404") {
					return &orchapi.InfoResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
					}, nil
				}

				// Tests can set viper.Set("test_orchestrator_features_disabled", true)
				featuresDisabled := viper.GetBool("test_orchestrator_features_disabled")

				if featuresDisabled {
					// All features disabled (apart from onboarding)
					info = &orchapi.Info{
						SchemaVersion: stringPtr("1.0"),
						Orchestrator: &orchapi.Data{
							Version: stringPtr("v2026.0.0-test"),
							Features: map[string]orchapi.FeatureInfo{
								"application-orchestration": {
									Installed: boolPtr(false),
								},
								"cluster-orchestration": {
									Installed: boolPtr(false),
								},
								"edge-infrastructure-manager": {
									Installed: boolPtr(true),
									Features: map[string]orchapi.FeatureInfo{
										"onboarding": {
											Installed: boolPtr(true),
										},
										"provisioning": {
											Installed: boolPtr(false),
										},
										"oob": {
											Installed: boolPtr(false),
										},
										"day2": {
											Installed: boolPtr(false),
										},
										"oxm-profile": {
											Installed: boolPtr(false),
										},
									},
								},
								"multitenancy": {
									Installed: boolPtr(false),
								},
								"orchestrator-observability": {
									Installed: boolPtr(false),
								},
							},
						},
					}
				} else {
					// All features enabled (default behavior)
					info = &orchapi.Info{
						SchemaVersion: stringPtr("1.0"),
						Orchestrator: &orchapi.Data{
							Version: stringPtr("v2026.0.0-test"),
							Features: map[string]orchapi.FeatureInfo{
								"application-orchestration": {
									Installed: boolPtr(true),
								},
								"cluster-orchestration": {
									Installed: boolPtr(true),
								},
								"edge-infrastructure-manager": {
									Installed: boolPtr(true),
									Features: map[string]orchapi.FeatureInfo{
										"onboarding": {
											Installed: boolPtr(true),
										},
										"provisioning": {
											Installed: boolPtr(true),
										},
										"oob": {
											Installed: boolPtr(true),
										},
										"day2": {
											Installed: boolPtr(true),
										},
										"oxm-profile": {
											Installed: boolPtr(true),
										},
									},
								},
								"multitenancy": {
									Installed: boolPtr(true),
								},
								"orchestrator-observability": {
									Installed: boolPtr(true),
								},
							},
						},
					}
				}

				return &orchapi.InfoResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200:      info,
				}, nil
			},
		).AnyTimes()

		return context.Background(), mockOrchClient, nil
	}
}

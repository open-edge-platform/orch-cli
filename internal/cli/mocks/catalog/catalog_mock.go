package catalog

import (
	"context"
	"net/http"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

func CreateCatalogMock(mctrl *gomock.Controller) interfaces.CatalogFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, catapi.ClientWithResponsesInterface, string, error) {
		mockClient := catapi.NewMockClientWithResponsesInterface(mctrl)

		// Get the project name from the command flags
		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project"
		}

		// Mock ListRegistries
		mockClient.EXPECT().CatalogServiceListRegistriesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *catapi.CatalogServiceListRegistriesParams, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceListRegistriesResponse, error) {
				_ = ctx // Acknowledge we're not using it
				_ = projectName
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
				_ = ctx // Acknowledge we're not using it
				_ = projectName
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
				_ = ctx // Acknowledge we're not using it
				_ = projectName
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
				_ = ctx // Acknowledge we're not using it
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
		return ctx, mockClient, projectName, nil
	}
}

package catalog

import (
	"context"
	"encoding/json"
	"net/http"
	"time" // Make sure this import is present

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

func CreateCatalogMock(mctrl *gomock.Controller) interfaces.CatalogFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, catapi.ClientWithResponsesInterface, string, error) {
		mockClient := catapi.NewMockClientWithResponsesInterface(mctrl)

		// Helper functions
		stringPtr := func(s string) *string { return &s }
		timePtr := func(t time.Time) *time.Time { return &t }

		// Sample timestamp for testing
		testTime := time.Date(2025, 12, 31, 23, 59, 59, 0, time.UTC)

		// Get the project name from the command flags
		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project"
		}

		// Helper function to select registry type and name
		getRegistryInfo := func(registryName string) (name, displayName, regType string) {
			switch registryName {
			case "registry-image":
				return "registry-image", "registry-display-name", "IMAGE"
			case "registry-helm":
				return "registry-helm", "registry-display-name", "HELM"
			default:
				return "registry", "registry-display-name", "HELM"
			}
		}

		// Mock GetRegistry
		mockClient.EXPECT().CatalogServiceGetRegistryWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, registryName string, params *catapi.CatalogServiceGetRegistryParams, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetRegistryResponse, error) {
				name, displayName, regType := getRegistryInfo(registryName)
				resp := &catapi.CatalogServiceGetRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.GetRegistryResponse{
						Registry: catapi.Registry{
							Name:        name,
							DisplayName: stringPtr(displayName),
							Description: stringPtr("new-description"),
							Type:        regType,
							RootUrl:     "http://x.y.z",
							Username:    stringPtr("user"),
							AuthToken:   stringPtr("token"),
							CreateTime:  timePtr(testTime),
							UpdateTime:  timePtr(testTime),
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
				// You may want to simulate both registries in the list
				registries := []catapi.Registry{}
				for _, registryName := range []string{"registry-image", "registry-helm"} {
					name, displayName, regType := getRegistryInfo(registryName)
					var authToken, username *string
					if params.ShowSensitiveInfo != nil && *params.ShowSensitiveInfo {
						authToken = stringPtr("token")
						username = stringPtr("user")
					} else {
						authToken = stringPtr("********")
						username = stringPtr("<none>")
					}
					registries = append(registries, catapi.Registry{
						Name:        name,
						DisplayName: stringPtr(displayName),
						Description: stringPtr("Registry-Description"),
						Type:        regType,
						RootUrl:     "http://x.y.z",
						CreateTime:  timePtr(testTime),
						UpdateTime:  timePtr(testTime),
						AuthToken:   authToken,
						Username:    username,
					})
				}
				resp := &catapi.CatalogServiceListRegistriesResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.ListRegistriesResponse{
						Registries: registries,
					},
				}
				return resp, nil
			},
		).AnyTimes()

		// Mock CreateRegistry
		mockClient.EXPECT().CatalogServiceCreateRegistryWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body interface{}, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateRegistryResponse, error) {
				// Extract registry name from body if possible
				var registryName string
				if b, ok := body.(map[string]interface{}); ok {
					if n, ok := b["name"].(string); ok {
						registryName = n
					}
				}
				name, displayName, regType := getRegistryInfo(registryName)
				resp := &catapi.CatalogServiceCreateRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200: &catapi.CreateRegistryResponse{
						Registry: catapi.Registry{
							Name:        name,
							DisplayName: stringPtr(displayName),
							Description: stringPtr("Registry-Description"),
							Type:        regType,
							RootUrl:     "http://x.y.z",
							CreateTime:  timePtr(testTime),
							UpdateTime:  timePtr(testTime),
						},
					},
				}
				return resp, nil
			},
		).AnyTimes()

		// Mock UpdateRegistry
		mockClient.EXPECT().CatalogServiceUpdateRegistryWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, registryName string, body catapi.CatalogServiceUpdateRegistryJSONRequestBody, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateRegistryResponse, error) {
				respBody, err := json.Marshal(struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}{
					Success: true,
					Message: "Registry updated successfully",
				})
				if err != nil {
					return nil, err
				}
				return &catapi.CatalogServiceUpdateRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					Body:         respBody,
				}, nil
			},
		).AnyTimes()

		// Mock DeleteRegistry
		mockClient.EXPECT().CatalogServiceDeleteRegistryWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, registryName string, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceDeleteRegistryResponse, error) {
				return &catapi.CatalogServiceDeleteRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockClient, projectName, nil
	}
}

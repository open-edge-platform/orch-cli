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

		mockClient.EXPECT().CatalogServiceGetApplicationWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, publisher string, appName string, appVersion string, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetApplicationResponse, error) {
				displayName := "profile.display.name"
				description := "Profile.Description"
				chartValues := "dmFsdWVzOiAxCnZhbDoy" // You can set a base64 string if needed

				profiles := []catapi.Profile{
					{
						Name:        "new-profile",
						DisplayName: &displayName,
						Description: &description,
						ChartValues: &chartValues,
						DeploymentRequirement: &[]catapi.DeploymentRequirement{
							{
								Name:                  "requirement",
								Version:               "1.2.3",
								DeploymentProfileName: stringPtr("Web server"),
							},
						},
						CreateTime: timePtr(testTime),
						UpdateTime: timePtr(testTime),
						ParameterTemplates: &[]catapi.ParameterTemplate{
							{
								Name:            "param1",
								DisplayName:     stringPtr("Parameter 1"),
								Type:            "string",
								Default:         stringPtr("default-value"),
								SuggestedValues: &[]string{"value1", "value2"},
							},
						},
					},
				}

				return &catapi.CatalogServiceGetApplicationResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.GetApplicationResponse{
						Application: catapi.Application{
							Name:               appName,
							Version:            appVersion,
							DisplayName:        &displayName,
							Description:        &description,
							ChartName:          "chart-name",
							ChartVersion:       "22.33.44",
							HelmRegistryName:   "myreg",
							ImageRegistryName:  nil,
							Profiles:           &profiles,
							DefaultProfileName: stringPtr("new-profile"),
							// Add other fields as needed
						},
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceUpdateApplicationWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, applicationName string, version string, body catapi.CatalogServiceUpdateApplicationJSONRequestBody, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateApplicationResponse, error) {
				respBody, err := json.Marshal(struct {
					Success bool   `json:"success"`
					Message string `json:"message"`
				}{
					Success: true,
					Message: "Application updated successfully",
				})
				if err != nil {
					return nil, err
				}
				return &catapi.CatalogServiceUpdateApplicationResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					Body:         respBody,
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceCreateApplicationWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, publisher string, body interface{}, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateApplicationResponse, error) {
				return &catapi.CatalogServiceCreateApplicationResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200:      &catapi.CreateApplicationResponse{
						// Fill with mock application data as needed
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceCreateDeploymentPackageWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body catapi.CatalogServiceCreateDeploymentPackageJSONRequestBody, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateDeploymentPackageResponse, error) {
				return &catapi.CatalogServiceCreateDeploymentPackageResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200:      &catapi.CreateDeploymentPackageResponse{
						// Fill with mock deployment package data as needed
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceGetDeploymentPackageWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentPackageName string, version string, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetDeploymentPackageResponse, error) {

				return &catapi.CatalogServiceGetDeploymentPackageResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.GetDeploymentPackageResponse{
						DeploymentPackage: catapi.DeploymentPackage{
							Name:        deploymentPackageName,
							Version:     version,
							DisplayName: stringPtr("displayName"),
							Description: stringPtr("description"),
							Profiles: &[]catapi.DeploymentProfile{
								{
									Name:                "deployment-package-profile",
									DisplayName:         stringPtr("deployment.profile.display.name"),
									Description:         stringPtr("Profile.for.testing"),
									CreateTime:          timePtr(testTime),
									UpdateTime:          timePtr(testTime),
									ApplicationProfiles: map[string]string{},
								},
							},
							DefaultProfileName:      stringPtr("default-profile"),
							ApplicationDependencies: &[]catapi.ApplicationDependency{},
							ApplicationReferences: []catapi.ApplicationReference{
								{Name: "app1", Version: "1.0"},
								{Name: "app2", Version: "1.0"},
							},
							// Add other fields as needed for your tests
						},
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceUpdateDeploymentPackageWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentPackageName string, version string, body catapi.CatalogServiceUpdateDeploymentPackageJSONRequestBody, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateDeploymentPackageResponse, error) {
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
				return &catapi.CatalogServiceUpdateDeploymentPackageResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					Body:         respBody,
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceListDeploymentPackagesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *catapi.CatalogServiceListDeploymentPackagesParams, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceListDeploymentPackagesResponse, error) {
				return &catapi.CatalogServiceListDeploymentPackagesResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.ListDeploymentPackagesResponse{
						DeploymentPackages: []catapi.DeploymentPackage{
							{
								Name:        "deployment-pkg",
								Version:     "1.0",
								DisplayName: stringPtr("deployment.package.display.name"),
								Description: stringPtr("Publisher.for.testing"),
								Profiles: &[]catapi.DeploymentProfile{
									{
										Name:                "deployment-package-profile",
										DisplayName:         stringPtr("deployment.profile.display.name"),
										Description:         stringPtr("Profile.for.testing"),
										CreateTime:          timePtr(testTime),
										UpdateTime:          timePtr(testTime),
										ApplicationProfiles: map[string]string{},
									},
								},
								ApplicationDependencies: &[]catapi.ApplicationDependency{},
								ApplicationReferences: []catapi.ApplicationReference{
									{Name: "app1", Version: "1.0"},
									{Name: "app2", Version: "1.0"},
								},
								Artifacts:          []catapi.ArtifactReference{},
								Extensions:         []catapi.APIExtension{},
								IsDeployed:         boolPtr(false),
								IsVisible:          boolPtr(true),
								DefaultProfileName: stringPtr("default-profile"),
								CreateTime:         timePtr(testTime),
								UpdateTime:         timePtr(testTime),
								// Add other fields as needed for your tests
							},
						},
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceDeleteDeploymentPackageWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentPackageName string, version string, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceDeleteDeploymentPackageResponse, error) {
				return &catapi.CatalogServiceDeleteDeploymentPackageResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceGetDeploymentPackageVersionsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, deploymentPackageName string, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetDeploymentPackageVersionsResponse, error) {
				return &catapi.CatalogServiceGetDeploymentPackageVersionsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.GetDeploymentPackageVersionsResponse{
						DeploymentPackages: []catapi.DeploymentPackage{
							{
								Name:        deploymentPackageName,
								Version:     "1.0",
								DisplayName: stringPtr("deployment.package.display.name"),
								Description: stringPtr("Publisher.for.testing"),
								Profiles: &[]catapi.DeploymentProfile{
									{
										Name:                "deployment-package-profile",
										DisplayName:         stringPtr("deployment.profile.display.name"),
										Description:         stringPtr("Profile.for.testing"),
										CreateTime:          timePtr(testTime),
										UpdateTime:          timePtr(testTime),
										ApplicationProfiles: map[string]string{},
									},
								},
							},
						},
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceCreateArtifactWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body catapi.CatalogServiceCreateArtifactJSONRequestBody, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateArtifactResponse, error) {
				return &catapi.CatalogServiceCreateArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200:      &catapi.CreateArtifactResponse{
						// Fill with mock artifact data as needed
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceListArtifactsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *catapi.CatalogServiceListArtifactsParams, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceListArtifactsResponse, error) {
				return &catapi.CatalogServiceListArtifactsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.ListArtifactsResponse{
						Artifacts: []catapi.Artifact{

							{
								Name:        "artifact",
								DisplayName: stringPtr("artifact-display-name"),
								Description: stringPtr("Artifact-Description"),
								MimeType:    "text/plain",
								CreateTime:  timePtr(time.Now()),
								UpdateTime:  timePtr(time.Now()),
								// Add other fields as needed
							},
						},
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceGetArtifactWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, artifactName string, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetArtifactResponse, error) {
				return &catapi.CatalogServiceGetArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.GetArtifactResponse{
						Artifact: catapi.Artifact{
							Name:        artifactName,
							DisplayName: stringPtr("artifact-display-name"),
							Description: stringPtr("Artifact-Description"),
							MimeType:    "text/plain",
							CreateTime:  timePtr(time.Now()),
							UpdateTime:  timePtr(time.Now()),
							// Add other fields as needed
						},
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceUpdateArtifactWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, artifactName string, body catapi.CatalogServiceUpdateArtifactJSONRequestBody, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateArtifactResponse, error) {
				return &catapi.CatalogServiceUpdateArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					Body:         []byte(`{"success":true,"message":"Artifact updated successfully"}`),
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceDeleteArtifactWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, artifactName string, reqEditors ...catapi.RequestEditorFn) (*catapi.CatalogServiceDeleteArtifactResponse, error) {
				return &catapi.CatalogServiceDeleteArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockClient, projectName, nil
	}

}

func boolPtr(b bool) *bool { return &b }

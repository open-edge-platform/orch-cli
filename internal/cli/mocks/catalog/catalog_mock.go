// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package catalog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time" // Make sure this import is present

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

// State tracking for mock behavior
var (
	createdProfiles = make(map[string][]catapi.CatalogV3DeploymentProfile)
	mockStateMutex  sync.RWMutex
)

func applicationKindPtr(k catapi.CatalogV3Kind) *catapi.CatalogV3Kind { return &k }
func boolPtr(b bool) *bool                                            { return &b }

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
			func(_ context.Context, _, registryName string, _ *catapi.CatalogServiceGetRegistryParams, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetRegistryResponse, error) {
				name, displayName, regType := getRegistryInfo(registryName)
				resp := &catapi.CatalogServiceGetRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3GetRegistryResponse{
						Registry: catapi.CatalogV3Registry{
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
			func(_ context.Context, _ string, params *catapi.CatalogServiceListRegistriesParams, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceListRegistriesResponse, error) {
				// You may want to simulate both registries in the list
				registries := []catapi.CatalogV3Registry{}
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
					registries = append(registries, catapi.CatalogV3Registry{
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
					JSON200: &catapi.CatalogV3ListRegistriesResponse{
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
			func(_ context.Context, _ string, body interface{}, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateRegistryResponse, error) {
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
					JSON200: &catapi.CatalogV3CreateRegistryResponse{
						Registry: catapi.CatalogV3Registry{
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
			func(_ context.Context, _ string, _ string, _ catapi.CatalogServiceUpdateRegistryJSONRequestBody, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateRegistryResponse, error) {
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
			func(_ context.Context, _ string, _ string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceDeleteRegistryResponse, error) {
				return &catapi.CatalogServiceDeleteRegistryResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceGetApplicationWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, appName string, appVersion string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetApplicationResponse, error) {
				displayName := "profile.display.name"
				description := "Profile.Description"
				chartValues := "dmFsdWVzOiAxCnZhbDoy" // You can set a base64 string if needed

				profiles := []catapi.CatalogV3Profile{
					{
						Name:        "new-profile",
						DisplayName: &displayName,
						Description: &description,
						ChartValues: &chartValues,
						DeploymentRequirement: &[]catapi.CatalogV3DeploymentRequirement{
							{
								Name:                  "requirement",
								Version:               "1.2.3",
								DeploymentProfileName: stringPtr("Web server"),
							},
						},
						CreateTime: timePtr(testTime),
						UpdateTime: timePtr(testTime),
						ParameterTemplates: &[]catapi.CatalogV3ParameterTemplate{
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
					JSON200: &catapi.CatalogV3GetApplicationResponse{
						Application: catapi.CatalogV3Application{
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
			func(_ context.Context, _ string, _ string, _ string, _ catapi.CatalogServiceUpdateApplicationJSONRequestBody, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateApplicationResponse, error) {
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
			func(_ context.Context, _ string, _ interface{}, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateApplicationResponse, error) {
				return &catapi.CatalogServiceCreateApplicationResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200:      &catapi.CatalogV3CreateApplicationResponse{
						// Fill with mock application data as needed
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceListApplicationsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ *catapi.CatalogServiceListApplicationsParams, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceListApplicationsResponse, error) {
				return &catapi.CatalogServiceListApplicationsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3ListApplicationsResponse{
						Applications: []catapi.CatalogV3Application{
							{
								Name:               "new-application",
								Version:            "1.2.3",
								Kind:               applicationKindPtr(catapi.KINDNORMAL),
								DisplayName:        stringPtr("application.display.name"),
								Description:        stringPtr("Application.Description"),
								ChartName:          "chart-name",
								ChartVersion:       "22.33.44",
								HelmRegistryName:   "test-registry",
								ImageRegistryName:  nil,
								Profiles:           &[]catapi.CatalogV3Profile{},
								DefaultProfileName: stringPtr(""),
								CreateTime:         timePtr(testTime),
								UpdateTime:         timePtr(testTime),
							},
							{
								Name:               "addon-app",
								Version:            "1.0.0",
								Kind:               applicationKindPtr(catapi.KINDADDON),
								DisplayName:        stringPtr("addon.display.name"),
								Description:        stringPtr("Addon Description"),
								ChartName:          "addon-chart",
								ChartVersion:       "1.0.0",
								HelmRegistryName:   "addon-registry",
								ImageRegistryName:  nil,
								Profiles:           &[]catapi.CatalogV3Profile{},
								DefaultProfileName: stringPtr(""),
								CreateTime:         timePtr(testTime),
								UpdateTime:         timePtr(testTime),
							},
							{
								Name:               "extension-app",
								Version:            "2.0.0",
								Kind:               applicationKindPtr(catapi.KINDEXTENSION),
								DisplayName:        stringPtr("extension.display.name"),
								Description:        stringPtr("Extension Description"),
								ChartName:          "extension-chart",
								ChartVersion:       "2.0.0",
								HelmRegistryName:   "extension-registry",
								ImageRegistryName:  nil,
								Profiles:           &[]catapi.CatalogV3Profile{},
								DefaultProfileName: stringPtr(""),
								CreateTime:         timePtr(testTime),
								UpdateTime:         timePtr(testTime),
							},
						},
						TotalElements: 3,
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceGetApplicationVersionsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetApplicationVersionsResponse, error) {
				return &catapi.CatalogServiceGetApplicationVersionsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3GetApplicationVersionsResponse{
						Application: []catapi.CatalogV3Application{
							{
								Name:               "new-application",
								Version:            "1.2.3",
								Kind:               applicationKindPtr(catapi.KINDNORMAL),
								DisplayName:        stringPtr("application.display.name"),
								Description:        stringPtr("Application.Description"),
								ChartName:          "chart-name",
								ChartVersion:       "22.33.44",
								HelmRegistryName:   "test-registry",
								ImageRegistryName:  nil,
								Profiles:           &[]catapi.CatalogV3Profile{},
								DefaultProfileName: stringPtr(""),
								CreateTime:         timePtr(testTime),
								UpdateTime:         timePtr(testTime),
							},
						},
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceDeleteApplicationWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, applicationName string, version string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceDeleteApplicationResponse, error) {
				if applicationName == "missing-app" {
					return nil, fmt.Errorf("application %s:%s not found", applicationName, version)
				}
				return &catapi.CatalogServiceDeleteApplicationResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceCreateDeploymentPackageWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ catapi.CatalogServiceCreateDeploymentPackageJSONRequestBody, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateDeploymentPackageResponse, error) {
				return &catapi.CatalogServiceCreateDeploymentPackageResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200:      &catapi.CatalogV3CreateDeploymentPackageResponse{
						// Fill with mock deployment package data as needed
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceGetDeploymentPackageWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, deploymentPackageName string, version string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetDeploymentPackageResponse, error) {
				// Get the tracked profiles for this package
				mockStateMutex.RLock()
				key := deploymentPackageName + ":" + version
				profiles, exists := createdProfiles[key]
				mockStateMutex.RUnlock()

				// If no tracked profiles, use default profiles
				if !exists {
					profiles = []catapi.CatalogV3DeploymentProfile{
						{
							Name:                "deployment-package-profile",
							DisplayName:         stringPtr("deployment.profile.display.name"),
							Description:         stringPtr("Profile.for.testing"),
							CreateTime:          timePtr(testTime),
							UpdateTime:          timePtr(testTime),
							ApplicationProfiles: map[string]string{},
						},
						{
							Name:                "test-deployment-profile",
							DisplayName:         stringPtr("test.deployment.profile.display.name"),
							Description:         stringPtr("Test.Profile.for.testing"),
							CreateTime:          timePtr(testTime),
							UpdateTime:          timePtr(testTime),
							ApplicationProfiles: map[string]string{},
						},
					}
				}

				return &catapi.CatalogServiceGetDeploymentPackageResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3GetDeploymentPackageResponse{
						DeploymentPackage: catapi.CatalogV3DeploymentPackage{
							Name:                    deploymentPackageName,
							Version:                 version,
							DisplayName:             stringPtr("displayName"),
							Description:             stringPtr("description"),
							Profiles:                &profiles,
							DefaultProfileName:      stringPtr("default-profile"),
							ApplicationDependencies: &[]catapi.CatalogV3ApplicationDependency{},
							ApplicationReferences: []catapi.CatalogV3ApplicationReference{
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
			func(_ context.Context, _ string, pkgName string, version string, body catapi.CatalogServiceUpdateDeploymentPackageJSONRequestBody, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateDeploymentPackageResponse, error) {
				// Track the profiles from the update request
				mockStateMutex.Lock()
				key := pkgName + ":" + version
				if body.Profiles != nil {
					// Ensure all profiles have required fields set
					profiles := make([]catapi.CatalogV3DeploymentProfile, len(*body.Profiles))
					for i, profile := range *body.Profiles {
						profiles[i] = profile
						// Ensure time fields are set if they're nil
						if profiles[i].CreateTime == nil {
							profiles[i].CreateTime = timePtr(testTime)
						}
						if profiles[i].UpdateTime == nil {
							profiles[i].UpdateTime = timePtr(testTime)
						}
						// Ensure ApplicationProfiles is not nil
						if profiles[i].ApplicationProfiles == nil {
							profiles[i].ApplicationProfiles = map[string]string{}
						}
					}
					createdProfiles[key] = profiles
				}
				mockStateMutex.Unlock()

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
			func(_ context.Context, _ string, _ *catapi.CatalogServiceListDeploymentPackagesParams, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceListDeploymentPackagesResponse, error) {
				// Get the tracked profiles for deployment-pkg:1.0.0
				mockStateMutex.RLock()
				key := "deployment-pkg:1.0.0"
				profiles, exists := createdProfiles[key]
				mockStateMutex.RUnlock()

				// If no tracked profiles, use default profiles
				if !exists {
					profiles = []catapi.CatalogV3DeploymentProfile{
						{
							Name:                "deployment-package-profile",
							DisplayName:         stringPtr("deployment.profile.display.name"),
							Description:         stringPtr("Profile.for.testing"),
							CreateTime:          timePtr(testTime),
							UpdateTime:          timePtr(testTime),
							ApplicationProfiles: map[string]string{},
						},
						{
							Name:                "test-deployment-profile",
							DisplayName:         stringPtr("test.deployment.profile.display.name"),
							Description:         stringPtr("Test.Profile.for.testing"),
							CreateTime:          timePtr(testTime),
							UpdateTime:          timePtr(testTime),
							ApplicationProfiles: map[string]string{},
						},
					}
				}

				return &catapi.CatalogServiceListDeploymentPackagesResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3ListDeploymentPackagesResponse{
						DeploymentPackages: []catapi.CatalogV3DeploymentPackage{
							{
								Name:                    "deployment-pkg",
								Version:                 "1.0.0",
								DisplayName:             stringPtr("deployment.package.display.name"),
								Description:             stringPtr("Publisher.for.testing"),
								Profiles:                &profiles,
								ApplicationDependencies: &[]catapi.CatalogV3ApplicationDependency{},
								ApplicationReferences: []catapi.CatalogV3ApplicationReference{
									{Name: "app1", Version: "1.0.0"},
									{Name: "app2", Version: "1.0.0"},
								},
								Artifacts:          []catapi.CatalogV3ArtifactReference{},
								Extensions:         []catapi.CatalogV3APIExtension{},
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
			func(_ context.Context, _ string, _ string, _ string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceDeleteDeploymentPackageResponse, error) {
				return &catapi.CatalogServiceDeleteDeploymentPackageResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceGetDeploymentPackageVersionsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, deploymentPackageName string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetDeploymentPackageVersionsResponse, error) {
				return &catapi.CatalogServiceGetDeploymentPackageVersionsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3GetDeploymentPackageVersionsResponse{
						DeploymentPackages: []catapi.CatalogV3DeploymentPackage{
							{
								Name:        deploymentPackageName,
								Version:     "1.0",
								DisplayName: stringPtr("deployment.package.display.name"),
								Description: stringPtr("Publisher.for.testing"),
								Profiles: &[]catapi.CatalogV3DeploymentProfile{
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
			func(_ context.Context, _ string, _ catapi.CatalogServiceCreateArtifactJSONRequestBody, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceCreateArtifactResponse, error) {
				return &catapi.CatalogServiceCreateArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
					JSON200:      &catapi.CatalogV3CreateArtifactResponse{
						// Fill with mock artifact data as needed
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceListArtifactsWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ *catapi.CatalogServiceListArtifactsParams, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceListArtifactsResponse, error) {
				return &catapi.CatalogServiceListArtifactsResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3ListArtifactsResponse{
						Artifacts: []catapi.CatalogV3Artifact{

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
			func(_ context.Context, _ string, artifactName string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceGetArtifactResponse, error) {
				return &catapi.CatalogServiceGetArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					JSON200: &catapi.CatalogV3GetArtifactResponse{
						Artifact: catapi.CatalogV3Artifact{
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
			func(_ context.Context, _ string, _ string, _ catapi.CatalogServiceUpdateArtifactJSONRequestBody, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceUpdateArtifactResponse, error) {
				return &catapi.CatalogServiceUpdateArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
					Body:         []byte(`{"success":true,"message":"Artifact updated successfully"}`),
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().CatalogServiceDeleteArtifactWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, _ string, _ ...catapi.RequestEditorFn) (*catapi.CatalogServiceDeleteArtifactResponse, error) {
				return &catapi.CatalogServiceDeleteArtifactResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockClient, projectName, nil
	}

}

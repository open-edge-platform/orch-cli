package cluster

import (
	"context"
	"fmt"
	"net/http"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	cluster "github.com/open-edge-platform/cli/pkg/rest/cluster"

	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

func CreateClusterMock(mctrl *gomock.Controller) interfaces.ClusterFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, cluster.ClientWithResponsesInterface, string, error) {
		mockClusterClient := cluster.NewMockClientWithResponsesInterface(mctrl)

		// Helper function for string pointers
		stringPtr := func(s string) *string { return &s }

		// Get the project name from the command flags
		projectName, err := cmd.Flags().GetString("project")
		if err != nil || projectName == "" {
			projectName = "test-project" // Default fallback
		}

		// Mock GetV2ProjectsProjectNameTemplatesNameVersionsVersionWithResponse (used by get template command)
		mockClusterClient.EXPECT().GetV2ProjectsProjectNameTemplatesNameVersionsVersionWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, templateName, version string, reqEditors ...cluster.RequestEditorFn) (*cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse, error) {
				fmt.Printf("The name of the template is %s", templateName)
				switch projectName {
				case "nonexistent-project":
					return &cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Not Found"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Project not found"),
						},
					}, nil
				default:
					switch templateName {
					case "nonexistent-template":
						return &cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse{
							HTTPResponse: &http.Response{StatusCode: 500, Status: "Not Found"},
							JSON500: &cluster.ProblemDetails{
								Message: stringPtr("Template not found"),
							},
						}, nil
					default:
						return &cluster.GetV2ProjectsProjectNameTemplatesNameVersionsVersionResponse{
							HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
							JSON200: &cluster.TemplateInfo{
								Name:    templateName,
								Version: version,
							},
						}, nil
					}
				}
			},
		).AnyTimes()

		// Mock GetV2ProjectsProjectNameTemplatesWithResponse (used by list templates command)
		mockClusterClient.EXPECT().GetV2ProjectsProjectNameTemplatesWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *cluster.GetV2ProjectsProjectNameTemplatesParams, reqEditors ...cluster.RequestEditorFn) (*cluster.GetV2ProjectsProjectNameTemplatesResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &cluster.GetV2ProjectsProjectNameTemplatesResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Project not found"),
						},
					}, nil
				default:
					return &cluster.GetV2ProjectsProjectNameTemplatesResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &cluster.TemplateInfoList{
							TemplateInfoList: &[]cluster.TemplateInfo{
								{
									Name:              "default-template",
									Version:           "v1.0.0",
									KubernetesVersion: "v1.28.0",
									Description:       stringPtr("Default Kubernetes cluster template"),
								},
								{
									Name:              "ha-template",
									Version:           "v1.1.0",
									KubernetesVersion: "v1.28.0",
									Description:       stringPtr("High availability cluster template"),
								},
							},
							TotalElements: func() *int32 { count := int32(2); return &count }(),
						},
					}, nil
				}
			},
		).AnyTimes()

		// Mock for getting a single cluster's details
		mockClusterClient.EXPECT().GetV2ProjectsProjectNameClustersNameWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, clusterName string, reqEditors ...cluster.RequestEditorFn) (*cluster.GetV2ProjectsProjectNameClustersNameResponse, error) {
				switch {
				case projectName == "nonexistent-project":
					return &cluster.GetV2ProjectsProjectNameClustersNameResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON404: &cluster.N404NotFound{
							Message: stringPtr("Project not found"),
						},
					}, nil
				case clusterName == "nonexistent-cluster":
					return &cluster.GetV2ProjectsProjectNameClustersNameResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON404: &cluster.N404NotFound{
							Message: stringPtr("Cluster not found"),
						},
					}, nil

				default:
					// Default case for any other cluster name
					return &cluster.GetV2ProjectsProjectNameClustersNameResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &cluster.ClusterDetailInfo{
							Name:              stringPtr(clusterName),
							KubernetesVersion: stringPtr("v1.28.0"),
							Template:          stringPtr("default-template-v1.0.0"),
							Nodes: &[]cluster.NodeInfo{ // Changed from NodeSpec to NodeInfo
								{
									Id:   stringPtr("default-node-id"),
									Role: stringPtr("control-plane"),
								},
							},
							Labels: &map[string]interface{}{ // Changed to interface{} type
								"created-by": "test",
							},
							ControlPlaneReady: &cluster.GenericStatus{
								Indicator: func() *cluster.StatusIndicator {
									status := cluster.STATUSINDICATIONIDLE
									return &status
								}(),
								Message: stringPtr("Ready"),
							},
							InfrastructureReady: &cluster.GenericStatus{
								Indicator: func() *cluster.StatusIndicator {
									status := cluster.STATUSINDICATIONIDLE
									return &status
								}(),
								Message: stringPtr("Ready"),
							},
							LifecyclePhase: &cluster.GenericStatus{
								Indicator: func() *cluster.StatusIndicator {
									status := cluster.STATUSINDICATIONIDLE
									return &status
								}(),
								Message: stringPtr("Provisioned"),
							},
							NodeHealth: &cluster.GenericStatus{
								Indicator: func() *cluster.StatusIndicator {
									status := cluster.STATUSINDICATIONIDLE
									return &status
								}(),
								Message: stringPtr("Healthy"),
							},
							ProviderStatus: &cluster.GenericStatus{
								Indicator: func() *cluster.StatusIndicator {
									status := cluster.STATUSINDICATIONIDLE
									return &status
								}(),
								Message: stringPtr("Ready"),
							},
						},
					}, nil
				}
			},
		).AnyTimes()

		// Update the list clusters mock to use the correct response structure
		mockClusterClient.EXPECT().GetV2ProjectsProjectNameClustersWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, params *cluster.GetV2ProjectsProjectNameClustersParams, reqEditors ...cluster.RequestEditorFn) (*cluster.GetV2ProjectsProjectNameClustersResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &cluster.GetV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Project not found"),
						},
					}, nil
				case "empty-project":
					// Return empty cluster list
					return &cluster.GetV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &struct {
							Clusters      *[]cluster.ClusterInfo `json:"clusters,omitempty"`
							TotalElements int32                  `json:"totalElements"`
						}{
							Clusters:      &[]cluster.ClusterInfo{},
							TotalElements: 0,
						},
					}, nil
				default:
					// Return sample clusters using the correct structure
					return &cluster.GetV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 200, Status: "OK"},
						JSON200: &struct {
							Clusters      *[]cluster.ClusterInfo `json:"clusters,omitempty"`
							TotalElements int32                  `json:"totalElements"`
						}{
							Clusters: &[]cluster.ClusterInfo{
								{
									Name:              stringPtr("test-cluster-1"),
									KubernetesVersion: stringPtr("v1.28.0"),
									NodeQuantity:      func() *int { count := 2; return &count }(),
									ControlPlaneReady: &cluster.GenericStatus{
										Indicator: func() *cluster.StatusIndicator {
											status := cluster.STATUSINDICATIONIDLE
											return &status
										}(),
										Message: stringPtr("Ready"),
									},
									InfrastructureReady: &cluster.GenericStatus{
										Indicator: func() *cluster.StatusIndicator {
											status := cluster.STATUSINDICATIONIDLE
											return &status
										}(),
										Message: stringPtr("Ready"),
									},
									LifecyclePhase: &cluster.GenericStatus{
										Indicator: func() *cluster.StatusIndicator {
											status := cluster.STATUSINDICATIONIDLE
											return &status
										}(),
										Message: stringPtr("Provisioned"),
									},
									NodeHealth: &cluster.GenericStatus{
										Indicator: func() *cluster.StatusIndicator {
											status := cluster.STATUSINDICATIONIDLE
											return &status
										}(),
										Message: stringPtr("Healthy"),
									},
									ProviderStatus: &cluster.GenericStatus{
										Indicator: func() *cluster.StatusIndicator {
											status := cluster.STATUSINDICATIONIDLE
											return &status
										}(),
										Message: stringPtr("Ready"),
									},
								},
							},
							TotalElements: 1,
						},
					}, nil
				}
			},
		).AnyTimes()

		// Add mock for DeleteV2ProjectsProjectNameClustersNameWithResponse (delete cluster)
		mockClusterClient.EXPECT().DeleteV2ProjectsProjectNameClustersNameWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, clusterName string, reqEditors ...cluster.RequestEditorFn) (*cluster.DeleteV2ProjectsProjectNameClustersNameResponse, error) {
				switch {
				case projectName == "nonexistent-project":
					return &cluster.DeleteV2ProjectsProjectNameClustersNameResponse{
						HTTPResponse: &http.Response{StatusCode: 404, Status: "Not Found"},
						JSON404: &cluster.N404NotFound{
							Message: stringPtr("Project not found"),
						},
					}, nil

				default:
					// Successful deletion
					return &cluster.DeleteV2ProjectsProjectNameClustersNameResponse{
						HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
					}, nil
				}
			},
		).AnyTimes()
		// Add mock for DeleteV2ProjectsProjectNameClustersNameNodesNodeIdWithResponse (delete cluster node)
		mockClusterClient.EXPECT().DeleteV2ProjectsProjectNameClustersNameNodesNodeIdWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName, clusterName, nodeId string, params *cluster.DeleteV2ProjectsProjectNameClustersNameNodesNodeIdParams, reqEditors ...cluster.RequestEditorFn) (*cluster.DeleteV2ProjectsProjectNameClustersNameNodesNodeIdResponse, error) {
				return &cluster.DeleteV2ProjectsProjectNameClustersNameNodesNodeIdResponse{
					HTTPResponse: &http.Response{StatusCode: 204, Status: "No Content"},
				}, nil
			},
		).AnyTimes()

		// Mock PostV2ProjectsProjectNameClustersWithResponse (used by create cluster command)
		mockClusterClient.EXPECT().PostV2ProjectsProjectNameClustersWithResponse(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(ctx context.Context, projectName string, body cluster.PostV2ProjectsProjectNameClustersJSONRequestBody, reqEditors ...cluster.RequestEditorFn) (*cluster.PostV2ProjectsProjectNameClustersResponse, error) {
				switch projectName {
				case "nonexistent-project":
					return &cluster.PostV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Not Found"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Project not found"),
						},
					}, nil
				case "duplicate-cluster-project":
					return &cluster.PostV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 500, Status: "Conflict"},
						JSON500: &cluster.ProblemDetails{
							Message: stringPtr("Cluster with same name already exists"),
						},
					}, nil
				default:
					return &cluster.PostV2ProjectsProjectNameClustersResponse{
						HTTPResponse: &http.Response{StatusCode: 201, Status: "Created"},
						JSON201:      stringPtr("cluster-12345"), // Return cluster ID as string
					}, nil
				}
			},
		).AnyTimes()

		ctx := context.Background()
		return ctx, mockClusterClient, projectName, nil
	}
}

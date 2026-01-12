// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/open-edge-platform/cli/pkg/rest/cluster"
	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
	depapi "github.com/open-edge-platform/cli/pkg/rest/deployment"
	infraapi "github.com/open-edge-platform/cli/pkg/rest/infra"
	rpsapi "github.com/open-edge-platform/cli/pkg/rest/rps"
	tenantapi "github.com/open-edge-platform/cli/pkg/rest/tenancy"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const timeLayout = "2006-01-02T15:04:05"
const maxValuesYAMLSize = 1 << 20 // 1 MiB

const (
	OobFeature           = "orchestrator.features.edge-infrastructure-manager.oob"
	OnboardingFeature    = "orchestrator.features.edge-infrastructure-manager.onboarding"
	ProvisioningFeature  = "orchestrator.features.edge-infrastructure-manager.provisioning"
	Day2Feature          = "orchestrator.features.edge-infrastructure-manager.day2"
	AppOrchFeature       = "orchestrator.features.application-orchestration"
	ClusterOrchFeature   = "orchestrator.features.cluster-orchestration"
	ObservabilityFeature = "orchestrator.features.observability"
	MultitenancyFeature  = "orchestrator.features.multitenancy"
	OrchVersion          = "orchestrator.version"
)

const (
	REGION = 0
	SITE   = 1
)

var disabledCommands = []string{}
var enabledCommands = []string{}

// Use the interface type instead of the concrete function type
var InfraFactory interfaces.InfraFactoryFunc = func(cmd *cobra.Command) (context.Context, infraapi.ClientWithResponsesInterface, string, error) {
	return getInfraServiceContext(cmd)
}

var ClusterFactory interfaces.ClusterFactoryFunc = func(cmd *cobra.Command) (context.Context, cluster.ClientWithResponsesInterface, string, error) {
	return getClusterServiceContext(cmd)
}

var CatalogFactory interfaces.CatalogFactoryFunc = func(cmd *cobra.Command) (context.Context, catapi.ClientWithResponsesInterface, string, error) {
	return getCatalogServiceContext(cmd)
}

var RpsFactory interfaces.RpsFactoryFunc = func(cmd *cobra.Command) (context.Context, rpsapi.ClientWithResponsesInterface, string, error) {
	return getRpsServiceContext(cmd)
}

var DeploymentFactory interfaces.DeploymentFactoryFunc = func(cmd *cobra.Command) (context.Context, depapi.ClientWithResponsesInterface, string, error) {
	return getDeploymentServiceContext(cmd)
}

var TenancyFactory interfaces.TenancyFactoryFunc = func(cmd *cobra.Command) (context.Context, tenantapi.ClientWithResponsesInterface, error) {
	return getTenancyServiceContext(cmd)
}

func getOutputContext(cmd *cobra.Command) (*tabwriter.Writer, bool) {
	verbose, _ := cmd.Flags().GetBool("verbose")
	debugHeadersValue, _ := cmd.Flags().GetBool(debugHeaders)
	writer := new(tabwriter.Writer)
	tabindent := tabwriter.TabIndent
	if debugHeadersValue {
		tabindent = tabwriter.Debug
	}
	writer.Init(cmd.OutOrStdout(), 0, 0, 3, ' ', tabindent)
	return writer, verbose
}

// Get the new background context, REST client, and project name given the specified command.
func getCatalogServiceContext(cmd *cobra.Command) (context.Context, *catapi.ClientWithResponses, string, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, nil, "", err
	}
	projectName, err := getProjectName(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	catalogClient, err := catapi.NewClientWithResponses(serverAddress, TLS13CatalogClientOption())
	if err != nil {
		return nil, nil, "", err
	}
	return context.Background(), catalogClient, projectName, nil
}

// Get the new background context, REST client, and project name given the specified command.
func getDeploymentServiceContext(cmd *cobra.Command) (context.Context, *depapi.ClientWithResponses, string, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, nil, "", err
	}
	projectName, err := getProjectName(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	deploymentClient, err := depapi.NewClientWithResponses(serverAddress, TLS13DeploymentClientOption())
	if err != nil {
		return nil, nil, "", err
	}
	return context.Background(), deploymentClient, projectName, nil
}

// Get the new background context, REST client, and project name given the specified command.
func getClusterServiceContext(cmd *cobra.Command) (context.Context, *coapi.ClientWithResponses, string, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, nil, "", err
	}
	projectName, err := getProjectName(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	coClient, err := coapi.NewClientWithResponses(serverAddress, TLS13ClusterClientOption())
	if err != nil {
		return nil, nil, "", err
	}
	return context.Background(), coClient, projectName, nil
}

// Get the new background context, REST client, and project name given the specified command.
func getInfraServiceContext(cmd *cobra.Command) (context.Context, *infraapi.ClientWithResponses, string, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, nil, "", err
	}
	projectName, err := getProjectName(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	infraClient, err := infraapi.NewClientWithResponses(serverAddress, TLS13InfraClientOption())
	if err != nil {
		return nil, nil, "", err
	}
	return context.Background(), infraClient, projectName, nil
}

// Get the new background context, REST client, and project name given the specified command.
func getRpsServiceContext(cmd *cobra.Command) (context.Context, *rpsapi.ClientWithResponses, string, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, nil, "", err
	}
	projectName, err := getProjectName(cmd)
	if err != nil {
		return nil, nil, "", err
	}
	rpsClient, err := rpsapi.NewClientWithResponses(serverAddress, TLS13RPSClientOption())
	if err != nil {
		return nil, nil, "", err
	}
	return context.Background(), rpsClient, projectName, nil
}

// Get the new background context, REST client, and project name given the specified command.
func getTenancyServiceContext(cmd *cobra.Command) (context.Context, *tenantapi.ClientWithResponses, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, nil, err
	}
	tenancyClient, err := tenantapi.NewClientWithResponses(serverAddress, TLS13TenancyClientOption())
	if err != nil {
		return nil, nil, err
	}
	return context.Background(), tenancyClient, nil
}

// Adds the mandatory project UUID, and the standard display-name, and description
func addEntityFlags(cmd *cobra.Command, entity string) {
	cmd.Flags().String("display-name", "", fmt.Sprintf("%s display name", entity))
	cmd.Flags().String("description", "", fmt.Sprintf("description of the %s", entity))
}

// Adds the standard orderBy and filter flags for List operations
func addListOrderingFilteringPaginationFlags(cmd *cobra.Command, entity string) {
	cmd.Flags().String("order-by", "", fmt.Sprintf("%s list order by", entity))
	cmd.Flags().String("filter", "", fmt.Sprintf("%s list filter", entity))
	cmd.Flags().Int32("page-size", 0, fmt.Sprintf("%s list maximum number of items", entity))
	cmd.Flags().Int32("offset", 0, fmt.Sprintf("%s list starting offset", entity))
}

// Gets the standard display-name, and description
func getEntityFlags(cmd *cobra.Command) (string, string, error) {
	displayName, err := cmd.Flags().GetString("display-name")
	if err != nil {
		return "", "", err
	}
	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return "", "", err
	}
	return displayName, description, err
}

func checkProjectExists(cmd *cobra.Command, projectName string) error {
	ctx, projectClient, err := TenancyFactory(cmd)
	if err != nil {
		return err
	}

	resp, err := projectClient.GETV1ProjectsProjectProjectWithResponse(ctx, projectName, auth.AddAuthHeader)

	// If the project does not exist, then resp.JSON200 and err will both be nil.
	// If the project does exist, but the user does not have access to see it, then statusUnauthorized will
	// be returned.

	if err == nil && (resp == nil || resp.JSON200 == nil || statusUnauthorized(resp.HTTPResponse)) {
		return fmt.Errorf("project %s does not exist or you do not have access to it", projectName)
	}

	if err != nil {
		return processError(err)
	}

	return nil
}

// Get the project name from the flag.
func getProjectName(cmd *cobra.Command) (string, error) {
	projectName, err := cmd.Flags().GetString("project")
	if err != nil {
		return "", err
	}

	if projectName == "" {
		return "", fmt.Errorf("required flag \"project\" not set")
	}

	// We're assuming that if getProjectName is required, then the project must exist.
	// CLI commands that do not require projects should never call getProjectName.
	err = checkProjectExists(cmd, projectName)
	if err != nil {
		return "", err
	}

	return projectName, nil
}

// Get the named flag as an optional string reference.
func getFlag(cmd *cobra.Command, flag string) *string {
	value, err := cmd.Flags().GetString(flag)
	if err != nil {
		return nil
	}
	return &value
}

// Get the named flag or a default value as a string reference
func getFlagOrDefault(cmd *cobra.Command, flag string, defaultValue *string) *string {
	value, err := cmd.Flags().GetString(flag)
	if err != nil || value == "" {
		return defaultValue
	}
	return &value
}

// Get the named flag or a default value as a boolean reference
func getBoolFlagOrDefault(cmd *cobra.Command, flag string, defaultValue *bool) *bool {
	value, err := cmd.Flags().GetBool(flag)
	if err != nil || !cmd.Flags().Changed(flag) {
		return defaultValue
	}
	return &value
}

// Get page size and offset from a command as int32 values
func getPageSizeOffset(cmd *cobra.Command) (int32, int32, error) {
	pageSize, err := cmd.Flags().GetInt32("page-size")
	if err != nil {
		return 0, 0, err
	}
	offset, err := cmd.Flags().GetInt32("offset")
	if err != nil {
		return 0, 0, err
	}
	return pageSize, offset, nil
}

// Reads input from the specified file path; from stdin if the path is "-"
func readInput(path string) ([]byte, error) {
	if err := isSafePath(path); err != nil {
		return nil, err
	}
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

func readInputWithLimit(path string) ([]byte, error) {
	var reader io.Reader
	if err := isSafePath(path); err != nil {
		return nil, err
	}
	if path == "-" {
		reader = os.Stdin
	} else {
		file, err := os.Open(path)
		if err != nil {
			return nil, err
		}
		defer file.Close()
		reader = file
	}
	limited := io.LimitReader(reader, maxValuesYAMLSize+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxValuesYAMLSize {
		return nil, fmt.Errorf("input exceeds maximum allowed size of %d bytes", maxValuesYAMLSize)
	}
	return data, nil
}

// Checks the specified REST status and if it signals an anomaly, return an error formatted using the specified message
// and status details.
func checkResponse(response *http.Response, body []byte, message string) error {
	if response != nil {
		return checkResponseCode(response.StatusCode, message, response.Status, body)
	}
	return nil
}

// Checks the specified REST status and if it signals an anomaly, return an error formatted using the specified message
// and status details.
func checkResponseCode(responseCode int, message string, responseMessage string, body []byte) error {
	if responseCode == 401 {
		return fmt.Errorf("%s. Unauthorized. Please Login. %s", message, responseMessage)
	} else if responseCode != 200 && responseCode != 201 && responseCode != 204 {
		// Try to parse the JSON body to extract just the message
		var errorResponse struct {
			Message string `json:"message"`
		}

		var bodyMessage string
		if len(body) > 0 {
			if err := json.Unmarshal(body, &errorResponse); err == nil && errorResponse.Message != "" {
				bodyMessage = fmt.Sprintf("\"%s\"", errorResponse.Message)
			} else {
				// Fallback to raw body if JSON parsing fails
				bodyMessage = string(body)
			}
		}

		if len(message) > 0 {
			if bodyMessage != "" {
				return fmt.Errorf("%s: %s\n%s", message, responseMessage, bodyMessage)
			}
			return fmt.Errorf("%s: %s", message, responseMessage)
		}

		if bodyMessage != "" {
			return fmt.Errorf("%s\n%s", responseMessage, bodyMessage)
		}
		return fmt.Errorf("%s", responseMessage)
	}
	return nil
}

// grpcStatus is a structure that represents the gRPC status message returned in the response body.
// Defining this here because the one in the official grpc package does not handle Details well.

type grpcStatus struct {
	Message string              `json:"message"`
	Code    int                 `json:"code"`
	Details []map[string]string `json:"details"`
}

// checkResponseGRPC is For apis that are using grpc-gateway and return a grpc Status in the response body for an error.
func checkResponseGRPC(response *http.Response, message string) error {
	if response == nil {
		return nil
	}
	if response.StatusCode >= http.StatusBadRequest { // handle 4xx and 5xx errors
		var status grpcStatus
		if err := json.NewDecoder(response.Body).Decode(&status); err == nil {
			// if there are details associated with the error, then print them
			for _, detail := range status.Details {
				if detailMessage, ok := detail["value"]; ok {
					fmt.Fprintln(os.Stderr, detailMessage)
				}
			}
			// if the grpc Status included a message then use it and return.
			// Otherwise, fall back to the standard response message.
			if status.Message != "" {
				return checkResponseCode(response.StatusCode, message, status.Message, []byte{})
			}
		}
	}
	return checkResponseCode(response.StatusCode, message, response.Status, []byte{})
}

// Checks the status code and returns the appropriate error
func checkStatus(statusCode int, message string, statusMessage string) (proceed bool, err error) {
	if statusCode == http.StatusOK {
		return true, nil
	} else if statusCode == 403 {
		return false, fmt.Errorf("%s: %s. Unauthenticated. Please login", message, statusMessage)
	}
	return false, fmt.Errorf("no response from backend - check api-endpoint and deployment-endpoint")
}

// Returns an error if the status is abnormal, i.e. status code is not OK and not merely NOT_FOUND
func statusIsAbnormal(response *http.Response, message string, args ...string) error {
	if response == nil || (response.StatusCode != 200 && response.StatusCode != 404 && response.StatusCode != 401) {
		return fmt.Errorf("%s:%s", message, args)
	}
	return nil
}

// Returns true of the code is NOT_FOUND
func statusIsNotFound(response *http.Response) bool {
	return response.StatusCode == 404
}

// Returns true of the code is UNAUTHORIZED
func statusUnauthorized(response *http.Response) bool {
	return response.StatusCode == 401
}

// Returns true of the code is UNAUTHORIZED
func statusForbidden(response *http.Response) bool {
	return response.StatusCode == 403
}

func processResponse(resp *http.Response, body []byte, writer *tabwriter.Writer, verbose bool, header string, message string) (proceed bool, err error) {
	if err = statusIsAbnormal(resp, message, resp.Status); err != nil {
		return false, err
	} else if statusIsNotFound(resp) {
		return false, getError(body, message)
	} else if statusUnauthorized(resp) {
		return false, getError(body, "Unauthorized. Please login")
	} else if statusForbidden(resp) {
		return false, getError(body, "Unauthorized (forbidden). Please login")
	}

	if !verbose {
		_, _ = fmt.Fprintf(writer, "%s\n", header)
	}
	return true, nil
}

// Constructs error with message from the specified prefix and a body of the error response
func getError(body []byte, prefixMessage string) error {
	m := map[string]interface{}{}
	if err := json.Unmarshal(body, &m); err == nil {
		if message, ok := m["message"]; ok {
			return fmt.Errorf("%s: %s", prefixMessage, message.(string))
		}
	}
	return fmt.Errorf("%s", prefixMessage)
}

func processError(err error) error {
	if strings.Contains(err.Error(), "504 DNS look up failed") {
		return fmt.Errorf("Unauthorized. Please login: token expired")
	}
	return err
}

func valueOrNone(s *string) string {
	if s != nil && len(*s) > 0 {
		return *s
	}
	return "<none>"
}

func safeBool(b *bool) bool {
	if b != nil {
		return *b
	}
	return false
}

func obscureValue(s *string) string {
	if s != nil && len(*s) > 0 {
		return "********"
	}
	return "<none>"
}

// isSafePath checks for path traversal and null byte injection.
func isSafePath(path string) error {
	clean := filepath.Clean(path)
	if strings.Contains(clean, ".."+string(os.PathSeparator)) || strings.HasPrefix(clean, "..") {
		return errors.New("path traversal detected: '..' not allowed in file paths")
	}
	if strings.ContainsRune(path, '\x00') {
		return errors.New("null byte detected in file path")
	}
	return nil
}

func TLS13CatalogClientOption() func(*catapi.Client) error {
	return func(c *catapi.Client) error {
		c.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					MaxVersion: tls.VersionTLS13,
				},
			},
		}
		return nil
	}
}
func TLS13DeploymentClientOption() func(*depapi.Client) error {
	return func(c *depapi.Client) error {
		c.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					MaxVersion: tls.VersionTLS13,
				},
			},
		}
		return nil
	}
}
func TLS13InfraClientOption() func(*infraapi.Client) error {
	return func(c *infraapi.Client) error {
		c.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					MaxVersion: tls.VersionTLS13,
				},
			},
		}
		return nil
	}
}
func TLS13ClusterClientOption() func(*coapi.Client) error {
	return func(c *coapi.Client) error {
		c.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					MaxVersion: tls.VersionTLS13,
				},
			},
		}
		return nil
	}
}

func TLS13RPSClientOption() func(*rpsapi.Client) error {
	return func(c *rpsapi.Client) error {
		c.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					MaxVersion: tls.VersionTLS13,
				},
			},
		}
		return nil
	}
}

func TLS13TenancyClientOption() func(*tenantapi.Client) error {
	return func(c *tenantapi.Client) error {
		c.Client = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{
					MinVersion: tls.VersionTLS13,
					MaxVersion: tls.VersionTLS13,
				},
			},
		}
		return nil
	}
}

// Helper function for fuzz tests
func isExpectedError(err error) bool {
	if err == nil {
		return false
	}
	expectedSubstrings := []string{
		"not", "unknown", "match", "invalid", "required", "requires",
		"no such", "missing", "no", "must", "in form", "incorrect",
		"unexpected", "expected", "failed", "is a", "bad", "exists", "open",
		"cannot", "nonexistent", "deleting", "getting", "listing", "wrong",
		"creating", "Internal Server Error", "null", "accepts", "error", "failed", "inappropriate",
	}
	errStr := strings.ToLower(err.Error())
	for _, substr := range expectedSubstrings {
		if strings.Contains(errStr, strings.ToLower(substr)) {
			return true
		}
	}
	return false
}

// addCommandIfFeatureEnabled conditionally adds a command to a parent command if the feature is enabled
func addCommandIfFeatureEnabled(parent *cobra.Command, child *cobra.Command, feature string) {
	commandPath := parent.Name() + " " + child.Name()
	if isFeatureEnabled(feature) {
		enabledCommands = append(enabledCommands, commandPath)
		parent.AddCommand(child)
	} else {
		disabledCommands = append(disabledCommands, commandPath)
	}
}

func isFeatureEnabled(feature string) bool {
	switch feature {
	case OobFeature:
		return viper.GetBool(OobFeature)
	case OnboardingFeature:
		return viper.GetBool(OnboardingFeature)
	case ProvisioningFeature:
		return viper.GetBool(ProvisioningFeature)
	case Day2Feature:
		return viper.GetBool(Day2Feature)
	case ObservabilityFeature:
		return viper.GetBool(ObservabilityFeature)
	case AppOrchFeature:
		return viper.GetBool(AppOrchFeature)
	case ClusterOrchFeature:
		return viper.GetBool(ClusterOrchFeature)
	case MultitenancyFeature:
		return viper.GetBool(MultitenancyFeature)
	default:
		return true // Default to enabled for unknown features
	}
}

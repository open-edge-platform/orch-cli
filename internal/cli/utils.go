// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"text/tabwriter"

	"github.com/gorilla/websocket"
	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	coapi "github.com/open-edge-platform/cli/pkg/rest/cluster"
	depapi "github.com/open-edge-platform/cli/pkg/rest/deployment"
	infraapi "github.com/open-edge-platform/cli/pkg/rest/infra"
	"github.com/spf13/cobra"
	"google.golang.org/grpc/metadata"
)

const timeLayout = "2006-01-02T15:04:05"

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
	catalogClient, err := catapi.NewClientWithResponses(serverAddress)
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
	deploymentClient, err := depapi.NewClientWithResponses(serverAddress)
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
	coClient, err := coapi.NewClientWithResponses(serverAddress)
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
	infraClient, err := infraapi.NewClientWithResponses(serverAddress)
	if err != nil {
		return nil, nil, "", err
	}
	return context.Background(), infraClient, projectName, nil
}

// Get the web socket for receiving event notifications.
func getCatalogWebSocket(cmd *cobra.Command) (*websocket.Conn, error) {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, err
	}
	serverAddress = strings.Replace(serverAddress, "https", "wss", 1)
	serverAddress = strings.Replace(serverAddress, "http", "ws", 1)

	u, err := url.JoinPath(serverAddress, "/catalog.orchestrator.apis/events")
	if err != nil {
		return nil, err
	}

	projectUUID, err := getProjectName(cmd)
	if err != nil {
		return nil, err
	}

	// Create an auxiliary request so that we can inject required auth headers into it
	req, err := http.NewRequest("GET", u, nil)
	if err != nil {
		return nil, err
	}

	// Inject the headers
	ctx := metadata.NewOutgoingContext(context.Background(), map[string][]string{auth.ActiveProjectID: {projectUUID}})
	if err := auth.AddAuthHeader(ctx, req); err != nil {
		return nil, err
	}

	// Dial to the web-socket using the annotated headers
	ws, _, err := websocket.DefaultDialer.Dial(u, req.Header)
	return ws, err
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

// Get the project name from the flag.
func getProjectName(cmd *cobra.Command) (string, error) {
	projectName, err := cmd.Flags().GetString("project")
	if err != nil {
		return "", err
	}
	if projectName == "" {
		return "", fmt.Errorf("required flag \"project\" not set")
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
	if path == "-" {
		return io.ReadAll(os.Stdin)
	}
	return os.ReadFile(path)
}

// Checks the specified REST status and if it signals an anomaly, return an error formatted using the specified message
// and status details.
func checkResponse(response *http.Response, message string) error {
	if response != nil {
		return checkResponseCode(response.StatusCode, message, response.Status)
	}
	return nil
}

// Checks the specified REST status and if it signals an anomaly, return an error formatted using the specified message
// and status details.
func checkResponseCode(responseCode int, message string, responseMessage string) error {
	if responseCode == 401 {
		return fmt.Errorf("%s. Unauthorized. Please Login. %s", message, responseMessage)
	} else if responseCode != 200 && responseCode != 201 && responseCode != 204 {
		if len(message) > 0 {
			return fmt.Errorf("%s: %s", message, responseMessage)
		}
		return fmt.Errorf("%s", responseMessage)
	}
	return nil
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

// Processes any error and any anomalous GET HTTP responses and determines whether to proceed or not
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

// Message represents subscription control messages
type Message struct {
	Op      string `json:"op"`
	Kind    string `json:"kind"`
	Project string `json:"project"`
	Payload []byte `json:"payload"`
}

// Subscribe for updates of a particular kind of entity
func subscribe(ws *websocket.Conn, kind string, projectUUID string) error {
	return ws.WriteJSON(Message{Op: "subscribe", Kind: kind, Project: projectUUID})
}

// Unsubscribe from updates of aparticular kind of entity
func unsubscribe(ws *websocket.Conn, kind string) {
	_ = ws.WriteJSON(Message{Op: "unsubscribe", Kind: kind})
}

// Unsubscribe from updates of a particular kind of entity when keyboard interrupt is detected.
func unsubscribeOnInterrupt(ws *websocket.Conn, kinds ...string) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for range c {
			for _, kind := range kinds {
				unsubscribe(ws, kind)
			}
			os.Exit(0)
		}
	}()
}

func isEvent(op string) bool {
	return op == "created" || op == "updated" || op == "deleted"
}

// Runs the main body of the watch command
func runWatchCommand(cmd *cobra.Command, printer func(io.Writer, string, []byte, bool) error, kinds ...string) error {
	ws, err := getCatalogWebSocket(cmd)
	if err != nil {
		return err
	}

	projectUUID, err := getProjectName(cmd)
	if err != nil {
		return err
	}

	// subscribe and on interrupt unsubscribe and exit
	for _, kind := range kinds {
		if err := subscribe(ws, kind, projectUUID); err != nil {
			return err
		}
		defer unsubscribe(ws, kind)
	}
	unsubscribeOnInterrupt(ws, kinds...)

	// consume acknowledgement and any events and print them
	msg := &Message{}
	writer, verbose := getOutputContext(cmd)
	for {
		if err = ws.ReadJSON(msg); err != nil {
			return err
		}
		if isEvent(msg.Op) {
			_, _ = fmt.Fprintf(writer, "%s: %s %s\t", shortenUUID(msg.Project), msg.Kind, msg.Op)
			if err := printer(writer, msg.Kind, msg.Payload, verbose); err != nil {
				return err
			}
			_ = writer.Flush()
		}
	}
}

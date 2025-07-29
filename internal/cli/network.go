// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/cobra"
)

var (
	networkAliases = []string{"net"}
)

var httpClient = &http.Client{Transport: &http.Transport{}}

func getCreateNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network <name> [flags]",
		Aliases: networkAliases,
		Short:   "Create a Network",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli create network my-network --project some-project",
		RunE:    runCreateNetworkCommand,
	}
	addEntityFlags(cmd, "network")
	cmd.Flags().String("type", "application-mesh", "Network type")
	return cmd
}

func getListNetworksCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "networks [flags]",
		Aliases: []string{"nets", "networks"},
		Short:   "List all networks",
		Example: "orch-cli list networks --project some-project",
		RunE:    runListNetworksCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "network")
	return cmd
}

func getGetNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network <name> [flags]",
		Aliases: networkAliases,
		Short:   "Get a network",
		Example: "orch-cli get network my-network --project some-project",
		Args:    cobra.ExactArgs(1),
		RunE:    runGetNetworkCommand,
	}
	return cmd
}

func getSetNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network <name> [flags]",
		Aliases: applicationAliases,
		Short:   "Update a network",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli set network my-updated-network --project some-project --type application-mesh",
		RunE:    runSetNetworkCommand,
	}
	addEntityFlags(cmd, "network")
	return cmd
}

func getDeleteNetworkCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "network <name> [flags]",
		Aliases: networkAliases,
		Short:   "Delete a network",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli delete network my-network --project some-project",
		RunE:    runDeleteNetworkCommand,
	}
	return cmd
}

func doREST(
	ctx context.Context,
	catEP string,
	method string,
	endpoint string,
	project string,
	body io.Reader,
	expectedStatus int,
) (*http.Response, error) {
	c := httpClient

	u, err := url.Parse(catEP)
	if err != nil {
		return nil, err
	}
	netURL := fmt.Sprintf("https://%s/v1/projects/%s/%s", u.Host, project, endpoint)

	req, err := http.NewRequestWithContext(ctx, method,
		netURL,
		body)
	req.Header.Add("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	err = auth.AddAuthHeader(ctx, req)
	if err != nil {
		return nil, err
	}
	resp, err := c.Do(req)

	if resp.StatusCode != expectedStatus {
		return resp, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return resp, err
}

type NetworkSpec struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type Network struct {
	Name string      `json:"name"`
	Spec NetworkSpec `json:"spec"`
}

type networks []Network

func listNetworks(ctx context.Context, catEP string, project string) (networks, error) {
	resp, err := doREST(ctx, catEP, http.MethodGet,
		"networks",
		project,
		nil,
		http.StatusOK)
	if err != nil {
		return networks{}, err
	}
	defer resp.Body.Close()

	/*
			bodyBytes, err := io.ReadAll(resp.Body)
		    if err != nil {
		        log.Fatal(err)
		    }
		    bodyString := string(bodyBytes)
		    fmt.Printf("%s\n", bodyString)
	*/

	var networksResp networks
	err = json.NewDecoder(resp.Body).Decode(&networksResp)
	if err != nil {
		return networks{}, err
	}
	return networksResp, nil
}

func getNetwork(ctx context.Context, catEP string, name string, project string) (NetworkSpec, error) {
	resp, err := doREST(ctx, catEP, http.MethodGet,
		"networks/"+name,
		project,
		nil,
		http.StatusOK)
	if err != nil {
		return NetworkSpec{}, err
	}
	defer resp.Body.Close()

	// TODO: smbaker: this returns {} when the network exists. Why?

	var networksResp NetworkSpec
	err = json.NewDecoder(resp.Body).Decode(&networksResp)
	if err != nil {
		return NetworkSpec{}, err
	}
	return networksResp, nil
}

func deleteNetwork(
	ctx context.Context,
	catEP string,
	name string,
	project string,
) error {
	resp, err := doREST(
		ctx,
		catEP,
		http.MethodDelete,
		"networks/"+name,
		project,
		nil,
		http.StatusOK,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return err
}

func createNetwork(
	ctx context.Context,
	catEP string,
	name string,
	netType string,
	description string,
	project string,
) error {
	b, err := json.Marshal(&NetworkSpec{Type: netType, Description: description})
	if err != nil {
		return err
	}

	resp, err := doREST(
		ctx,
		catEP,
		http.MethodPut,
		"networks/"+name,
		project,
		bytes.NewReader(b),
		http.StatusOK,
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	return err
}

var networkHeader = fmt.Sprintf("%s\t%s\t%s", "Name", "Type", "Description")

func printNetworks(writer io.Writer, netList networks, verbose bool) {
	_ = verbose

	for _, net := range netList {
		_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", net.Name, net.Spec.Type, net.Spec.Description)
	}
}

func getNetworkContext(cmd *cobra.Command) (context.Context, string, string, error) {
	ctx := context.Background()
	projectName, err := getProjectName(cmd)
	if err != nil {
		return nil, "", "", err
	}
	catEP, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return nil, "", "", err
	}
	return ctx, catEP, projectName, nil
}

func runCreateNetworkCommand(cmd *cobra.Command, args []string) error {
	ctx, catEP, projectName, err := getNetworkContext(cmd)
	if err != nil {
		return processError(err)
	}

	name := args[0]

	netType := *getFlag(cmd, "type")

	description, err := cmd.Flags().GetString("description")
	if err != nil {
		return processError(err)
	}

	err = createNetwork(ctx, catEP, name, netType, description, projectName)
	if err != nil {
		return processError(err)
	}
	return nil
}

func runListNetworksCommand(cmd *cobra.Command, _ []string) error {
	ctx, catEP, projectName, err := getNetworkContext(cmd)
	if err != nil {
		return processError(err)
	}

	writer, verbose := getOutputContext(cmd)

	networks, err := listNetworks(ctx, catEP, projectName)
	if err != nil {
		return processError(err)
	}

	fmt.Fprintf(writer, "%s\n", networkHeader)
	printNetworks(writer, networks, verbose)
	return writer.Flush()
}

func runGetNetworkCommand(cmd *cobra.Command, args []string) error {
	ctx, catEP, projectName, err := getNetworkContext(cmd)
	if err != nil {
		return processError(err)
	}

	name := args[0]

	writer, verbose := getOutputContext(cmd)

	network, err := getNetwork(ctx, catEP, name, projectName)
	if err != nil {
		return processError(err)
	}

	_ = verbose

	fmt.Printf("[Note: Type and Despection might be shown as empty]\n")

	fmt.Fprintf(writer, "Name: %s\n", name)
	fmt.Fprintf(writer, "Type: %s\n", network.Type)
	fmt.Fprintf(writer, "Description: %s\n", network.Description)

	return writer.Flush()
}

func runSetNetworkCommand(cmd *cobra.Command, args []string) error {
	_ = cmd
	_ = args
	// there's nothing to set
	return nil
}

func runDeleteNetworkCommand(cmd *cobra.Command, args []string) error {
	ctx, catEP, projectName, err := getNetworkContext(cmd)
	if err != nil {
		return processError(err)
	}

	name := args[0]

	err = deleteNetwork(ctx, catEP, name, projectName)
	if err != nil {
		return processError(err)
	}
	return nil
}

// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/errors"
	"github.com/spf13/cobra"
)

func getListChartsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "charts <registry-name> [<chart-name>] [flags]",
		Args:              cobra.MinimumNArgs(1),
		Short:             "Get chart names or chart versions from a HELM registry",
		PersistentPreRunE: auth.CheckAuth,
		Example:           "orch-cli get charts my-registry --project my-project",
		Aliases:           chartAliases,
		RunE:              getCharts,
	}
	return cmd
}

func getCharts(cmd *cobra.Command, args []string) error {
	serverAddress, err := cmd.Flags().GetString(apiEndpoint)
	if err != nil {
		return err
	}

	projectName, err := getProjectName(cmd)
	if err != nil {
		return err
	}

	registryName := args[0]

	url := fmt.Sprintf("%s/v3/projects/%s/catalog/charts?registry=%s", serverAddress, projectName, registryName)
	if len(args) > 1 {
		url = fmt.Sprintf("%s&chart=%s", url, args[1])
	}

	data, err := getRegistryContent(url)
	if err != nil {
		return err
	}

	list := strings.Replace(string(data), "null", "[]", 1)
	fmt.Printf("%s\n", list)
	return nil
}

func getRegistryContent(url string) ([]byte, error) {
	ctx := context.Background()
	r, _ := http.NewRequest("GET", url, nil)
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("CORS", "true")
	if err := auth.AddAuthHeader(ctx, r); err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(r)
	if err != nil {
		return nil, err
	} else if resp.StatusCode == 204 {
		return []byte("[]"), nil
	} else if resp.StatusCode != 200 {
		return nil, errors.NewInvalid("chart retrieval failed: %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

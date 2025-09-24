// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/open-edge-platform/cli/pkg/auth"
	catapi "github.com/open-edge-platform/cli/pkg/rest/catalog"
	"github.com/spf13/cobra"
)

func getCreateRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Create a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli create registry my-registry --root-url https://my-registry.example.com --username my-user --auth-token my-token --project some-project",
		Aliases: registryAliases,
		RunE:    runCreateRegistryCommand,
	}
	addEntityFlags(cmd, "registry")
	cmd.Flags().String("root-url", "", "root URL of the registry (required)")
	_ = cmd.MarkFlagRequired("root-url")
	cmd.Flags().String("username", "", "username for accessing the registry")
	cmd.Flags().String("auth-token", "", "authentication token for accessing the registry")
	cmd.Flags().String("ca-certs", "", "CA certs for accessing the registry")
	cmd.Flags().String("registry-type", "helm", "registry type (helm or image)")
	cmd.Flags().String("inventory-url", "", "inventory URL of the registry")
	cmd.Flags().String("api-type", "helm", "registry API type")
	return cmd
}

func getListRegistriesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registries [flags]",
		Aliases: registryAliases,
		Short:   "List all registries",
		Example: "orch-cli list registries --project some-project --order-by name",
		RunE:    runListRegistriesCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "registry")
	cmd.Flags().Bool("show-sensitive-info", false, "show sensitive info, e.g. auth-token, CA certs")
	return cmd
}

func getGetRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Get a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli get registry my-registry --project some-project",
		Aliases: registryAliases,
		RunE:    runGetRegistryCommand,
	}
	cmd.Flags().Bool("show-sensitive-info", false, "show sensitive info, e.g. auth-token, CA certs")
	return cmd
}

func getSetRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Update a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli set registry my-registry --root-url https://my-registry.example.com --username my-user --auth-token my-token --project some-project",
		Aliases: registryAliases,
		RunE:    runSetRegistryCommand,
	}
	addEntityFlags(cmd, "registry")
	cmd.Flags().String("root-url", "", "root URL of the registry")
	cmd.Flags().String("username", "", "username for accessing the registry")
	cmd.Flags().String("auth-token", "", "authentication token for accessing the registry")
	cmd.Flags().String("ca-certs", "", "CA certs for accessing the registry")
	cmd.Flags().String("registry-type", "helm", "registry type (helm or image)")
	cmd.Flags().String("inventory-url", "", "inventory URL of the registry")
	cmd.Flags().String("api-type", "helm", "registry API type")
	return cmd
}

func getDeleteRegistryCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "registry <name> [flags]",
		Short:   "Delete a registry",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli delete registry my-registry --project some-project",
		Aliases: registryAliases,
		RunE:    runDeleteRegistryCommand,
	}
	return cmd
}

var registryHeader = fmt.Sprintf("%s\t%s\t%s\t%s\t%s",
	"Name", "Display Name", "Description", "Type", "Root URL")

func printRegistries(writer io.Writer, registryList *[]catapi.Registry, verbose bool, showSensitive bool) {
	for _, r := range *registryList {
		if !verbose {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\t%s\t%s\n", r.Name,
				valueOrNone(r.DisplayName), valueOrNone(r.Description), r.Type, r.RootUrl)
		} else {
			_, _ = fmt.Fprintf(writer, "Name: %s\n", r.Name)
			_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(r.DisplayName))
			_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(r.Description))
			_, _ = fmt.Fprintf(writer, "Root URL: %s\n", r.RootUrl)
			_, _ = fmt.Fprintf(writer, "Inventory URL: %s\n", valueOrNone(r.InventoryUrl))
			_, _ = fmt.Fprintf(writer, "Type: %s\n", r.Type)
			_, _ = fmt.Fprintf(writer, "API Type: %s\n", valueOrNone(r.ApiType))
			_, _ = fmt.Fprintf(writer, "Username: %s\n", valueOrNone(r.Username))
			if showSensitive {
				_, _ = fmt.Fprintf(writer, "AuthToken: %s\n", valueOrNone(r.AuthToken))
				_, _ = fmt.Fprintf(writer, "CA Certs: %s\n", valueOrNone(r.Cacerts))
			} else {
				_, _ = fmt.Fprintf(writer, "AuthToken: %s\n", obscureValue(r.AuthToken))
				_, _ = fmt.Fprintf(writer, "CA Certs: %s\n", obscureValue(r.Cacerts))
			}
			_, _ = fmt.Fprintf(writer, "Create Time: %s\n", r.CreateTime.Format(timeLayout))
			_, _ = fmt.Fprintf(writer, "Update Time: %s\n\n", r.UpdateTime.Format(timeLayout))
		}
	}
}

func verifyRegistryType(cmd *cobra.Command) error {
	regType := *getFlag(cmd, "registry-type")
	if regType == "helm" || regType == "image" {
		return nil
	}
	return fmt.Errorf("invalid registry type %s", regType)
}

func getRegistryType(cmd *cobra.Command) string {
	typeFromCommand := *getFlag(cmd, "registry-type")
	if typeFromCommand == "helm" {
		return "HELM"
	} else if typeFromCommand == "image" {
		return "IMAGE"

	}
	return ""
}

func runCreateRegistryCommand(cmd *cobra.Command, args []string) error {
	err := verifyRegistryType(cmd)
	if err != nil {
		return err
	}
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}
	name := args[0]
	registryType := getRegistryType(cmd)

	resp, err := catalogClient.CatalogServiceCreateRegistryWithResponse(ctx, projectName,
		catapi.CatalogServiceCreateRegistryJSONRequestBody{
			Name:         name,
			DisplayName:  &displayName,
			Description:  &description,
			RootUrl:      *getFlag(cmd, "root-url"),
			InventoryUrl: getFlag(cmd, "inventory-url"),
			Username:     getFlag(cmd, "username"),
			AuthToken:    getFlag(cmd, "auth-token"),
			Cacerts:      getFlag(cmd, "ca-certs"),
			Type:         registryType,
			ApiType:      getFlag(cmd, "api-type"),
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating registry %s", name))
}

func runListRegistriesCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	showSensitive, _ := cmd.Flags().GetBool("show-sensitive-info")
	resp, err := catalogClient.CatalogServiceListRegistriesWithResponse(ctx, projectName,
		&catapi.CatalogServiceListRegistriesParams{
			OrderBy:           getFlag(cmd, "order-by"),
			Filter:            getFlag(cmd, "filter"),
			PageSize:          &pageSize,
			Offset:            &offset,
			ShowSensitiveInfo: &showSensitive,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, registryHeader,
		"error listing registries"); !proceed {
		return err
	}
	printRegistries(writer, &resp.JSON200.Registries, verbose, showSensitive)
	return writer.Flush()
}

func runGetRegistryCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	showSensitive, _ := cmd.Flags().GetBool("show-sensitive-info")

	resp, err := catalogClient.CatalogServiceGetRegistryWithResponse(ctx, projectName, name,
		&catapi.CatalogServiceGetRegistryParams{ShowSensitiveInfo: &showSensitive}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose,
		registryHeader, fmt.Sprintf("error getting registry %s", name)); !proceed {
		return err
	}
	printRegistries(writer, &[]catapi.Registry{resp.JSON200.Registry}, verbose, showSensitive)
	return writer.Flush()
}

func runSetRegistryCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	showSensitive := true
	gresp, err := catalogClient.CatalogServiceGetRegistryWithResponse(ctx, projectName, name,
		&catapi.CatalogServiceGetRegistryParams{ShowSensitiveInfo: &showSensitive}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("registry %s not found", name)); err != nil {
		return err
	}

	registry := gresp.JSON200.Registry

	resp, _ := catalogClient.CatalogServiceUpdateRegistryWithResponse(ctx, projectName, name,
		catapi.CatalogServiceUpdateRegistryJSONRequestBody{
			Name:         name,
			DisplayName:  getFlagOrDefault(cmd, "display-name", registry.DisplayName),
			Description:  getFlagOrDefault(cmd, "description", registry.Description),
			RootUrl:      *getFlagOrDefault(cmd, "root-url", &registry.RootUrl),
			InventoryUrl: getFlagOrDefault(cmd, "inventory-url", registry.InventoryUrl),
			Username:     getFlagOrDefault(cmd, "username", registry.Username),
			AuthToken:    getFlagOrDefault(cmd, "auth-token", registry.AuthToken),
			Cacerts:      getFlagOrDefault(cmd, "ca-certs", registry.Cacerts),
			Type:         registry.Type,
			ApiType:      getFlagOrDefault(cmd, "api-type", registry.ApiType),
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating registry %s", name))
}

func runDeleteRegistryCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	gresp, err := catalogClient.CatalogServiceGetRegistryWithResponse(ctx, projectName, name,
		&catapi.CatalogServiceGetRegistryParams{}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("registry %s not found", name)); err != nil {
		return err
	}

	resp, err := catalogClient.CatalogServiceDeleteRegistryWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	return checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting registry %s", name))
}

func printRegistryEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.Registry
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	printRegistries(writer, &[]catapi.Registry{item}, verbose, false)
	return nil
}

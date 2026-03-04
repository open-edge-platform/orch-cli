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

func getCreateArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Create an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli create artifact my-artifact --mime-type application/octet-stream --artifact /path/to/artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runCreateArtifactCommand,
	}
	addEntityFlags(cmd, "artifact")
	cmd.Flags().String("mime-type", "", "artifact MIME type (required)")
	_ = cmd.MarkFlagRequired("mime-type")
	cmd.Flags().String("artifact", "-", "path to the artifact file; - for stdin")
	return cmd
}

func getListArtifactsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifacts [flags]",
		Short:   "List all artifacts",
		Example: "orch-cli list artifacts --project some-project --order-by name",
		Aliases: artifactAliases,
		RunE:    runListArtifactsCommand,
	}
	addListOrderingFilteringPaginationFlags(cmd, "artifact")
	return cmd
}

func getGetArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Get an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli get artifact my-artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runGetArtifactCommand,
	}
	return cmd
}

func getSetArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Update an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli set artifact my-artifact --mime-type application/octet-stream --artifact /path/to/artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runSetArtifactCommand,
	}
	addEntityFlags(cmd, "artifact")
	cmd.Flags().String("mime-type", "", "artifact MIME type")
	cmd.Flags().String("artifact", "", "path to the artifact file; - for stdin")
	return cmd
}

func getDeleteArtifactCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "artifact <name> [flags]",
		Short:   "Delete an artifact",
		Args:    cobra.ExactArgs(1),
		Example: "orch-cli delete artifact my-artifact --project some-project",
		Aliases: artifactAliases,
		RunE:    runDeleteArtifactCommand,
	}
	return cmd
}

var artifactHeader = fmt.Sprintf("%s\t%s\t%s", "Name", "Display Name", "Description")

func printArtifacts(writer io.Writer, artifactList *[]catapi.CatalogV3Artifact, verbose bool) {
	for _, a := range *artifactList {
		if !verbose {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", a.Name, valueOrNone(a.DisplayName), valueOrNone(a.Description))
		} else {
			_, _ = fmt.Fprintf(writer, "Name: %s\n", a.Name)
			_, _ = fmt.Fprintf(writer, "Display Name: %s\n", valueOrNone(a.DisplayName))
			_, _ = fmt.Fprintf(writer, "Description: %s\n", valueOrNone(a.Description))
			_, _ = fmt.Fprintf(writer, "Mime Type: %s\n\n", a.MimeType)
		}
	}
}

func runCreateArtifactCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}
	displayName, description, err := getEntityFlags(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	binary, err := readInput(*getFlag(cmd, "artifact"))
	if err != nil {
		return fmt.Errorf("error reading artifact content: %w", err)
	}

	resp, err := catalogClient.CatalogServiceCreateArtifactWithResponse(ctx, projectName,
		catapi.CatalogServiceCreateArtifactJSONRequestBody{
			Name:        name,
			DisplayName: &displayName,
			Description: &description,
			MimeType:    *getFlag(cmd, "mime-type"),
			Artifact:    binary,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while creating artifact %s", name)); err != nil {
		return err
	}
	fmt.Printf("Artifact '%s' created successfully\n", name)
	return nil
}

func runListArtifactsCommand(cmd *cobra.Command, _ []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	pageSize, offset, err := getPageSizeOffset(cmd)
	if err != nil {
		return err
	}

	resp, err := catalogClient.CatalogServiceListArtifactsWithResponse(ctx, projectName,
		&catapi.CatalogServiceListArtifactsParams{
			OrderBy:  getFlag(cmd, "order-by"),
			Filter:   getFlag(cmd, "filter"),
			PageSize: &pageSize,
			Offset:   &offset,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, artifactHeader,
		"error listing artifacts"); !proceed {
		return err
	}
	printArtifacts(writer, &resp.JSON200.Artifacts, verbose)
	return writer.Flush()
}

func runGetArtifactCommand(cmd *cobra.Command, args []string) error {
	writer, verbose := getOutputContext(cmd)
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	resp, err := catalogClient.CatalogServiceGetArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if proceed, err := processResponse(resp.HTTPResponse, resp.Body, writer, verbose, artifactHeader,
		fmt.Sprintf("error getting artifact %s", name)); !proceed {
		return err
	}
	printArtifacts(writer, &[]catapi.CatalogV3Artifact{resp.JSON200.Artifact}, verbose)
	return writer.Flush()
}

func runSetArtifactCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	gresp, err := catalogClient.CatalogServiceGetArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("artifact %s not found", name)); err != nil {
		return err
	}

	artifact := gresp.JSON200.Artifact
	binary := gresp.Body

	// If the artifact flag was given, fetch the new content to replace the existing one
	newArtifactPath := *getFlag(cmd, "artifact")
	if len(newArtifactPath) > 0 {
		binary, err = readInput(*getFlag(cmd, "artifact"))
		if err != nil {
			return fmt.Errorf("error reading artifact content: %w", err)
		}
	}

	resp, err := catalogClient.CatalogServiceUpdateArtifactWithResponse(ctx, projectName, name,
		catapi.CatalogServiceUpdateArtifactJSONRequestBody{
			Name:        name,
			DisplayName: getFlagOrDefault(cmd, "display-name", artifact.DisplayName),
			Description: getFlagOrDefault(cmd, "description", artifact.Description),
			MimeType:    *getFlagOrDefault(cmd, "mime-type", &artifact.MimeType),
			Artifact:    binary,
		}, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error while updating artifact %s", name)); err != nil {
		return err
	}
	fmt.Printf("Artifact '%s' updated successfully\n", name)
	return nil
}

func runDeleteArtifactCommand(cmd *cobra.Command, args []string) error {
	ctx, catalogClient, projectName, err := CatalogFactory(cmd)
	if err != nil {
		return err
	}

	name := args[0]
	gresp, err := catalogClient.CatalogServiceGetArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err = checkResponse(gresp.HTTPResponse, gresp.Body, fmt.Sprintf("artifact %s not found", name)); err != nil {
		return err
	}

	resp, err := catalogClient.CatalogServiceDeleteArtifactWithResponse(ctx, projectName, name, auth.AddAuthHeader)
	if err != nil {
		return processError(err)
	}
	if err := checkResponse(resp.HTTPResponse, resp.Body, fmt.Sprintf("error deleting artifact %s", name)); err != nil {
		return err
	}
	fmt.Printf("Artifact '%s' deleted successfully\n", name)
	return nil
}

func printArtifactEvent(writer io.Writer, _ string, payload []byte, verbose bool) error {
	var item catapi.CatalogV3Artifact
	if err := json.Unmarshal(payload, &item); err != nil {
		return err
	}
	printArtifacts(writer, &[]catapi.CatalogV3Artifact{item}, verbose)
	return nil
}

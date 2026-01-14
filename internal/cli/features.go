// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func printFeatures(cmd *cobra.Command) {
	features := viper.GetStringMap("orchestrator.features")
	if len(features) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No features configured")
		return
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Edge Orchestrator Features:")
	printFeaturesRecursive(cmd, features, "", 0)

}

func printCommands(cmd *cobra.Command, ctype string) {
	if ctype == "disabled" {
		fmt.Fprintln(cmd.OutOrStdout(), "\nDisabled Commands:")
		for _, c := range disabledCommands {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", c)
		}
	} else if ctype == "enabled" {
		fmt.Fprintln(cmd.OutOrStdout(), "\nEnabled Commands:")
		for _, c := range enabledCommands {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", c)
		}
	}
}

func printFeaturesRecursive(cmd *cobra.Command, features map[string]interface{}, prefix string, depth int) {
	// Sort keys for consistent output
	keys := make([]string, 0, len(features))
	for k := range features {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Create indentation for nested features
	indent := strings.Repeat("  ", depth)

	// Print "installed" key first if it exists
	if installedValue, hasInstalled := features["installed"]; hasInstalled {
		fullKey := "installed"
		if prefix != "" {
			fullKey = prefix + ".installed"
		}

		if v, ok := installedValue.(bool); ok {
			status := "disabled"
			if v {
				status = "enabled"
			}
			displayKey := strings.TrimSuffix(fullKey, ".installed")
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s | %s\n", indent, displayKey, status)
		}
	}

	// Then print all other keys
	for _, key := range keys {
		if key == "installed" {
			continue // Skip, already printed above
		}

		value := features[key]
		fullKey := key
		if prefix != "" {
			fullKey = prefix + "." + key
		}

		switch v := value.(type) {
		case bool:
			status := "disabled"
			if v {
				status = "enabled"
			}
			// Remove .installed suffix from the display name
			displayKey := strings.TrimSuffix(fullKey, ".installed")
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s | %s\n", indent, displayKey, status)
		case map[string]interface{}:
			// Print nested features with increased depth
			printFeaturesRecursive(cmd, v, fullKey, depth+1)
		default:
			// Handle other types if needed
			displayKey := strings.TrimSuffix(fullKey, ".installed")
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s | %v\n", indent, displayKey, v)
		}
	}
}

func getListFeaturesCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "features [flags]",
		Aliases: featuresAliases,
		Short:   "Lists all features supported by Edge Orchestrator",
		Example: "orch-cli list features --project some-project",
		RunE:    runListFeaturesCommand,
	}
	cmd.Flags().BoolP("show-disabled", "d", false, "Show the commands which are disabled for the target Edge Orchestrator deployment configuration")
	cmd.Flags().BoolP("show-enabled", "e", false, "Show the commands which are exclusively enabled for the target Edge Orchestrator deployment configuration")
	return cmd
}

func runListFeaturesCommand(cmd *cobra.Command, _ []string) error {
	showDisabled, _ := cmd.Flags().GetBool("show-disabled")
	showEnabled, _ := cmd.Flags().GetBool("show-enabled")

	printFeatures(cmd)
	if showDisabled {
		printCommands(cmd, "disabled")
	}
	if showEnabled {
		printCommands(cmd, "enabled")
	}
	return nil
}

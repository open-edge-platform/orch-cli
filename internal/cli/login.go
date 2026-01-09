// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"net/url"
	"os"
	"strings"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/openidconnect"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"
)

func getLoginCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "login <username> [<password>] [flags]",
		Args:    cobra.MinimumNArgs(1),
		Short:   "Login to Orchestrator",
		Example: "orch-cli login admin",
		Long: "Login to Keycloak server to retrieve an refresh-token and save locally. " +
			"Refresh Token is good until Max Session Timout or until logout. " +
			"If password is not supplied it will be prompted for.",
		RunE: login,
	}
	cmd.Flags().String("client-id", auth.DefaultClientID, "client-id (application name) in keycloak")
	cmd.Flags().String("keycloak", "", "keycloak OIDC endpoint - will be retrieved from api-endpoint/openidc-issuer by default")
	cmd.Flags().String("claims", "openid profile email", "keycloak OIDC endpoint")
	cmd.Flags().Bool("quiet", false, "use to silence login message")
	cmd.Flags().Bool("show-token", false, "display the access token, e.g. for use in 'curl'")

	return cmd
}

func getLogoutCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "logout",
		Short:   "Logout of Orchestrator",
		Long:    "Discard local api-token",
		Example: "orch-cli logout",
		RunE:    logout,
	}
	return cmd
}

func login(cmd *cobra.Command, args []string) error {
	existingRefreshToken := viper.GetString(auth.RefreshTokenField)
	if existingRefreshToken != "" {
		log.Warnf("Already logged in - please logout first")
		return fmt.Errorf("already logged in - please logout first")
	}

	username := args[0]
	if username == "" {
		log.Warnf("username is blank")
		return fmt.Errorf("username cannot be blank")
	}

	clientID, err := cmd.Flags().GetString("client-id")
	if err != nil {
		return err
	}

	var keycloakEp string
	// If user has not given a keycloak endpoint, ask the api-endpoint what it should be
	keycloakEpUser, err := cmd.Flags().GetString("keycloak")
	if err != nil {
		return err
	}
	if keycloakEpUser != "" {
		// If user has specified a value then use it
		keycloakEp = keycloakEpUser
	} else {
		catEp := viper.GetString(apiEndpoint)
		u, err := url.Parse(catEp)
		if err != nil {
			return err
		}
		parts := strings.SplitN(u.Host, ".", 2)
		if len(parts) != 2 {
			return fmt.Errorf("Failed to determine keycloak enpoint from api endpoint. Consider using --keycloak flag")
		}
		keycloakEp = fmt.Sprintf("https://keycloak.%s/realms/master", parts[1])
		fmt.Printf("Determined keycloak endpoint from api endpoint: %s\n", keycloakEp)
	}

	claims, err := cmd.Flags().GetString("claims")
	if err != nil {
		return err
	}

	urlString := strings.Builder{}
	urlString.WriteString(keycloakEp)

	gt := openidconnect.TokenGrantType("password")

	kcClient, err := auth.KeycloakFactory(cmd.Context(), urlString.String())
	if err != nil {
		return err
	}
	// Check first that this is a keycloak instance before we start sending our password over
	responseWellKnown, errWellKnown := kcClient.GetWellKnownOpenidConfigurationWithResponse(cmd.Context())
	if errWellKnown != nil {
		return errWellKnown
	}
	if responseWellKnown.JSON200 == nil {
		return fmt.Errorf("invalid response from Identity Povider. Cannot login. Check Keycloak")
	}
	if *responseWellKnown.JSON200.TokenEndpoint != fmt.Sprintf("%s/protocol/openid-connect/token", urlString.String()) {
		return fmt.Errorf("unexpected token endpoint %s. Cannot login. Check Keycloak", *responseWellKnown.JSON200.TokenEndpoint)
	}

	var password string
	if len(args) > 1 {
		password = args[1]
	} else {
		fmt.Print("Enter Password: ")
		bytePassword, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return err
		}
		password = string(bytePassword)
	}

	response, err := kcClient.PostProtocolOpenidConnectTokenWithFormdataBodyWithResponse(cmd.Context(), openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody{
		ClientId:  &clientID,
		GrantType: &gt,
		Username:  &username,
		Password:  &password,
		Claims:    &claims,
	})
	if err != nil {
		return err
	}
	if response.StatusCode() == 401 {
		log.Warnf("Unauthorized")
		return fmt.Errorf("unauthorized %d", response.StatusCode())
	} else if response.StatusCode() != 200 {
		log.Warnf("unexpected response %d", response.StatusCode())
		return fmt.Errorf("response %s", string(response.Body))
	}
	refreshToken := response.JSON200.RefreshToken
	viper.Set(auth.RefreshTokenField, *refreshToken)
	viper.Set(auth.UserName, username)
	viper.Set(auth.ClientIDField, clientID)
	viper.Set(auth.KeycloakEndpointField, keycloakEp)

	if err = viper.WriteConfig(); err != nil {
		return err
	}

	showToken, err := cmd.Flags().GetBool("show-token")
	if err != nil {
		return err
	}
	if showToken {
		fmt.Printf("%s\n", *response.JSON200.AccessToken)
	} else {
		quiet, err := cmd.Flags().GetBool("quiet")
		if err != nil {
			return err
		}
		if !quiet {
			expiryTimeSec := response.JSON200.ExpiresIn
			fmt.Println("WARNING! Token has been issued and is stored locally. Do not share it with anyone.")
			fmt.Printf("Use 'logout' to delete it. Expires in %d sec.\n", *expiryTimeSec)
		}
	}

	// TODO get the config from EO endpoint, parse it out and pass on to loadConfig()
	if err := loadFeatureConfig(); err != nil {
		return err
	}

	return nil
}

func logout(_ *cobra.Command, _ []string) error {
	apiTokenIf := viper.Get(auth.RefreshTokenField)
	username := viper.Get(auth.UserName)
	if apiToken, ok := apiTokenIf.(string); ok && apiToken != "" {
		log.Warnf("Discarding local API token for %s", username)
		viper.Set(auth.RefreshTokenField, "")
		viper.Set(auth.UserName, "")
		viper.Set(auth.ClientIDField, "")
		viper.Set(auth.KeycloakEndpointField, "")

		return viper.WriteConfig()
	}
	log.Info("Was not logged in - no-op")
	return nil
}

func loadFeatureConfig() error {

	// TODO: For now we hardcode the feature config here. Then extract from EO endpoint later.
	viper.Set("orchestrator.version", "2026.0")
	viper.Set("orchestrator.features.application-orchestration", false)
	viper.Set("orchestrator.features.cluster-orchestration", false)
	viper.Set("orchestrator.features.edge-infrastructure-manager.onboarding", true)
	viper.Set("orchestrator.features.edge-infrastructure-manager.provisioning", false)
	viper.Set("orchestrator.features.edge-infrastructure-manager.day2", false)
	viper.Set("orchestrator.features.edge-infrastructure-manager.oob", false)
	viper.Set("orchestrator.features.observability", false)
	viper.Set("orchestrator.features.multitenancy", false)

	if err := viper.WriteConfig(); err != nil {
		return err
	}

	return nil
}

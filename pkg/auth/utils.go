// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/atomix/dazl"
	"github.com/golang-jwt/jwt/v5"
	"github.com/open-edge-platform/orch-library/go/pkg/openidconnect"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	AccessTokenEnv = "MT_GW_TOKEN"

	RefreshTokenField     = "refresh-token"
	ClientIDField         = "client-id"
	KeycloakEndpointField = "keycloak-endpoint"

	ActiveProjectID = "ActiveProjectID"
	DefaultClientID = "system-client"
	TrustCertField  = "trust-cert"
	UserName        = "username"
)

var log = dazl.GetPackageLogger()

// KeycloakFactory a global object that rerefs to a Keycloak API
// can be replaced during test to point at a mock implementation
var KeycloakFactory = newKeycloakClient

func TLS13ClientOption() openidconnect.ClientOption {
	return func(c *openidconnect.Client) error {
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

func newKeycloakClient(_ context.Context, endpoint string) (openidconnect.ClientWithResponsesInterface, error) {
	client, err := openidconnect.NewClientWithResponses(endpoint, TLS13ClientOption())
	if err != nil {
		return nil, err
	}
	return openidconnect.ClientWithResponsesInterface(client), err
}

func AddAuthHeader(ctx context.Context, req *http.Request) error {
	// Short-cut to use an actual access token from an environment variable, rather than refresh token from configuration.
	authToken := os.Getenv(AccessTokenEnv)
	if authToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", authToken))
		return nil
	}

	refreshTokenStr := viper.GetString(RefreshTokenField)
	if refreshTokenStr == "" {
		return nil
	}
	clientID := viper.GetString(ClientIDField)
	keycloakEp := viper.GetString(KeycloakEndpointField)

	urlString := strings.Builder{}
	urlString.WriteString(keycloakEp)

	gt := openidconnect.TokenGrantType("refresh_token")

	// Use refresh_token to get an access_token
	kcClient, err := KeycloakFactory(ctx, urlString.String())
	if err != nil {
		return err
	}
	response, err := kcClient.PostProtocolOpenidConnectTokenWithFormdataBodyWithResponse(ctx, openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody{
		ClientId:     &clientID,
		GrantType:    &gt,
		RefreshToken: &refreshTokenStr,
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
	accessToken := response.JSON200.AccessToken

	if accessToken != nil && *accessToken != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", *accessToken))
	}
	return nil
}

func CheckAuth(cmd *cobra.Command, _ []string) error {
	noAuth, err := cmd.Flags().GetBool("noauth")
	if err != nil {
		return err
	}
	if noAuth {
		return nil
	}
	if user := viper.Get(UserName); user == nil {
		return fmt.Errorf("not logged in - user unknown")
	}
	refreshToken := viper.Get(RefreshTokenField)
	if refreshToken == nil {
		return fmt.Errorf("not logged in - no token present")
	}
	refreshTokenStr, ok := refreshToken.(string)
	if !ok {
		return fmt.Errorf("refresh token invalid. Please logout")
	} else if refreshTokenStr == "" {
		return fmt.Errorf("token is empty. Please login")
	}
	jwtParser := jwt.NewParser()
	token, _, err := jwtParser.ParseUnverified(refreshTokenStr, jwt.MapClaims{})
	if err != nil {
		return fmt.Errorf("%v. Please logout", err)
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return fmt.Errorf("cannot extract claims from token. Please logout")
	}
	typ, audOk := claims["typ"]
	if !audOk {
		return fmt.Errorf("cannot extract 'typ' claim from token. Please logout")
	}
	if typ.(string) != "Refresh" {
		return fmt.Errorf("token type is not Refresh. Please logout")
	}

	azp, azpOk := claims["azp"]
	if !azpOk {
		return fmt.Errorf("cannot extract 'azp' claim from token. Please logout")
	}
	if azp.(string) != viper.GetString(ClientIDField) {
		return fmt.Errorf("token client-id is not correct. Please logout")
	}

	iss, issOk := claims["iss"]
	if !issOk {
		return fmt.Errorf("cannot extract 'iss' claim from token. Please logout")
	}
	if iss.(string) != viper.GetString(KeycloakEndpointField) {
		return fmt.Errorf("token issuer is not correct. Please logout")
	}

	return nil
}

// GrantTypeMatcher A simple type to see if a keycloak request body has a grantType matching the one given
type GrantTypeMatcher struct {
	GrantType string
}

func (g GrantTypeMatcher) Matches(x interface{}) bool {
	body, bodyOk := x.(openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody)
	if !bodyOk {
		return false
	}
	if string(*body.GrantType) == g.GrantType {
		return true
	}
	return false
}

func (g GrantTypeMatcher) String() string {
	return g.GrantType
}

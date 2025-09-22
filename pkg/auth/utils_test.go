// SPDX-FileCopyrightText: 2023-present Intel Corporation
//
// SPDX-License-Identifier: Apache-2.0

package auth

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/open-edge-platform/orch-library/go/pkg/openidconnect"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const kcTest = "http://unit-test-keycloak/realms/master"

func TestAddAuthHeader(t *testing.T) {

	mctrl := gomock.NewController(t)
	KeycloakFactory = func(_ context.Context, _ string) (openidconnect.ClientWithResponsesInterface, error) {
		mockClient := openidconnect.NewMockClientWithResponsesInterface(mctrl)

		mockClient.EXPECT().PostProtocolOpenidConnectTokenWithFormdataBodyWithResponse(gomock.Any(), GrantTypeMatcher{GrantType: "refresh_token"}, gomock.Any()).DoAndReturn(
			func(_ context.Context, body openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody, _ ...openidconnect.RequestEditorFn) (*openidconnect.PostProtocolOpenidConnectTokenResponse, error) {
				assert.Nil(t, body.Username)
				assert.Nil(t, body.Password)
				assert.NotNil(t, body.ClientId)
				assert.NotNil(t, body.RefreshToken)

				resp := new(openidconnect.PostProtocolOpenidConnectTokenResponse)
				resp.HTTPResponse = &http.Response{
					StatusCode: 200,
					Status:     "OK",
				}
				rt := "test refresh token after refresh"
				at := "test access token after refresh"
				expireSec := 60
				tokenResponse := openidconnect.TokenResponse{
					AccessToken:      &at,
					DeviceSecret:     nil,
					ExpiresIn:        &expireSec,
					IdToken:          nil,
					RefreshExpiresIn: &expireSec,
					RefreshToken:     &rt,
					Scope:            nil,
					TokenType:        nil,
				}
				resp.JSON200 = &tokenResponse

				return resp, nil
			}).AnyTimes()

		return mockClient, nil
	}
	req := new(http.Request)
	ctx := context.Background()
	req = req.WithContext(ctx)
	req.Header = make(map[string][]string)

	viper.Set(RefreshTokenField, "test_token")
	err := AddAuthHeader(context.Background(), req)
	assert.NoError(t, err)
	assert.Equal(t, "Bearer test access token after refresh", req.Header.Get("Authorization"))

	req2 := new(http.Request)
	req2 = req2.WithContext(ctx)
	req2.Header = make(map[string][]string)

	// Unset the refresh_token - same as logging out
	viper.Set(UserName, "")
	viper.Set(RefreshTokenField, "")
	viper.Set(ClientIDField, "")
	viper.Set(KeycloakEndpointField, "")
	err = AddAuthHeader(context.Background(), req2)
	assert.NoError(t, err)
	assert.Equal(t, "", req2.Header.Get("Authorization"))
}

func TestCheckAuth(t *testing.T) {
	viper.Set(UserName, nil)
	viper.Set(RefreshTokenField, nil)
	viper.Set(ClientIDField, nil)
	viper.Set(KeycloakEndpointField, nil)

	testCmd := &cobra.Command{
		Use: "test",
	}
	testCmd.Flags().Bool("noauth", false, "where auth is not required")
	err := testCmd.Flags().Set("noauth", "true")
	assert.NoError(t, err)
	err = CheckAuth(testCmd, nil)
	assert.NoError(t, err)

	err = testCmd.Flags().Set("noauth", "false")
	assert.NoError(t, err)

	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "not logged in - user unknown")

	viper.Set(UserName, "test")
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "not logged in - no token present")

	viper.Set(RefreshTokenField, "")
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "token is empty. Please login")

	viper.Set(RefreshTokenField, "test")
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "token is malformed: token contains an invalid number of segments. Please logout")

	viper.Set(ClientIDField, "system-client")
	viper.Set(KeycloakEndpointField, fmt.Sprintf("%s/protocol/openid-connect/token", kcTest))
	tokenNoClaims := jwt.NewWithClaims(jwt.SigningMethodHS256, nil)
	rtNoClaims, err := tokenNoClaims.SignedString([]byte("test-key"))
	assert.NoError(t, err)
	viper.Set(RefreshTokenField, rtNoClaims)
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "cannot extract 'typ' claim from token. Please logout")

	tokenWrongTyp := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ": "wrong",
	})
	rtWrongTyp, err := tokenWrongTyp.SignedString([]byte("test-key"))
	assert.NoError(t, err)
	viper.Set(RefreshTokenField, rtWrongTyp)
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "token type is not Refresh. Please logout")

	tokenNoAzp := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ": "Refresh",
	})
	rtNoAzp, err := tokenNoAzp.SignedString([]byte("test-key"))
	assert.NoError(t, err)
	viper.Set(RefreshTokenField, rtNoAzp)
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "cannot extract 'azp' claim from token. Please logout")

	tokenWrongAzp := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ": "Refresh",
		"azp": "invalid-azp",
	})
	rtWrongAzp, err := tokenWrongAzp.SignedString([]byte("test-key"))
	assert.NoError(t, err)
	viper.Set(RefreshTokenField, rtWrongAzp)
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "token client-id is not correct. Please logout")

	tokenNoIss := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ": "Refresh",
		"azp": "system-client",
	})
	rtNoIss, err := tokenNoIss.SignedString([]byte("test-key"))
	assert.NoError(t, err)
	viper.Set(RefreshTokenField, rtNoIss)
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "cannot extract 'iss' claim from token. Please logout")

	tokenWrongIss := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"typ": "Refresh",
		"azp": "system-client",
		"iss": "invalid-iss",
	})
	rtWrongIss, err := tokenWrongIss.SignedString([]byte("test-key"))
	assert.NoError(t, err)
	viper.Set(RefreshTokenField, rtWrongIss)
	err = CheckAuth(testCmd, nil)
	assert.EqualError(t, err, "token issuer is not correct. Please logout")

	tokenCorrect := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"username": "user",
		"typ":      "Refresh",
		"azp":      "system-client",
		"iss":      fmt.Sprintf("%s/protocol/openid-connect/token", kcTest),
		"nbf":      time.Now(),
	})
	rtCorrect, err := tokenCorrect.SignedString([]byte("test-key"))
	assert.NoError(t, err)
	viper.Set(RefreshTokenField, rtCorrect)
	err = CheckAuth(testCmd, nil)
	assert.NoError(t, err)
}

func TestNewKeycloakClient(t *testing.T) {
	client, err := newKeycloakClient(context.Background(), "http://just.for.test")
	assert.NoError(t, err)
	assert.NotNil(t, client)
}

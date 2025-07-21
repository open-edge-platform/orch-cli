package authmock

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/openidconnect"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const kcTest = "http://unit-test-keycloak/realms/master"

// Return the factory function instead of assigning it
func CreateKeycloakMock(s *suite.Suite, mctrl *gomock.Controller) func(context.Context, string) (openidconnect.ClientWithResponsesInterface, error) {
	return func(ctx context.Context, _ string) (openidconnect.ClientWithResponsesInterface, error) {
		kcTokenEndpoint := fmt.Sprintf("%s/protocol/openid-connect/token", kcTest)

		token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"username": "u",
			"typ":      "Refresh",
			"azp":      "system-client",
			"iss":      kcTest,
			"nbf":      time.Now(),
		})

		rt, err := token.SignedString([]byte("test-key"))
		s.NoError(err)

		mockClient := openidconnect.NewMockClientWithResponsesInterface(mctrl)

		mockClient.EXPECT().GetWellKnownOpenidConfigurationWithResponse(ctx, gomock.Any()).DoAndReturn(
			func(_ context.Context, _ ...openidconnect.RequestEditorFn) (*openidconnect.GetWellKnownOpenidConfigurationResponse, error) {
				return &openidconnect.GetWellKnownOpenidConfigurationResponse{
					JSON200: &openidconnect.WellKnownResponse{
						TokenEndpoint: &kcTokenEndpoint,
					},
				}, nil
			},
		).AnyTimes()

		mockClient.EXPECT().PostProtocolOpenidConnectTokenWithFormdataBodyWithResponse(gomock.Any(), auth.GrantTypeMatcher{GrantType: "password"}, gomock.Any()).DoAndReturn(
			func(_ context.Context, body openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody, _ ...openidconnect.RequestEditorFn) (*openidconnect.PostProtocolOpenidConnectTokenResponse, error) {
				s.NotNil(body.Username)
				s.NotNil(body.Password)
				s.NotNil(body.ClientId)
				s.Nil(body.RefreshToken)

				resp := new(openidconnect.PostProtocolOpenidConnectTokenResponse)
				resp.HTTPResponse = &http.Response{
					StatusCode: 200,
					Status:     "OK",
				}
				at := "test access token after login"
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

		mockClient.EXPECT().PostProtocolOpenidConnectTokenWithFormdataBodyWithResponse(gomock.Any(), auth.GrantTypeMatcher{GrantType: "refresh_token"}, gomock.Any()).DoAndReturn(
			func(_ context.Context, body openidconnect.PostProtocolOpenidConnectTokenFormdataRequestBody, _ ...openidconnect.RequestEditorFn) (*openidconnect.PostProtocolOpenidConnectTokenResponse, error) {
				s.Nil(body.Username)
				s.Nil(body.Password)
				s.NotNil(body.ClientId)
				s.NotNil(body.RefreshToken)

				resp := new(openidconnect.PostProtocolOpenidConnectTokenResponse)
				resp.HTTPResponse = &http.Response{
					StatusCode: 200,
					Status:     "OK",
				}
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
}

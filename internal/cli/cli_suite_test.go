// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	_ "github.com/mattn/go-sqlite3"
	"github.com/open-edge-platform/app-orch-catalog/pkg/restClient"
	restproxy "github.com/open-edge-platform/app-orch-catalog/pkg/restProxy"
	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/open-edge-platform/orch-library/go/pkg/openidconnect"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

const (
	verboseOutput  = true
	simpleOutput   = false
	timestampRegex = `^[0-9-]*T[0-9:]*$`
	kcTest         = "http://unit-test-keycloak/realms/master"
)

type commandArgs map[string]string
type commandOutput map[string]map[string]string

type CLITestSuite struct {
	suite.Suite
	proxy restproxy.MockRestProxy
}

func (s *CLITestSuite) SetupSuite() {
	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")
	viper.Set(auth.TrustCertField, "")

	mctrl := gomock.NewController(s.T())

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

	auth.KeycloakFactory = func(ctx context.Context, _ string) (openidconnect.ClientWithResponsesInterface, error) {
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

	////DeploymentFactory = func(_ context.Context, _ string) (depapi.ClientWithResponsesInterface, error) {
	////	mockClient := NewMockClientWithResponsesInterface(mctrl)
	////	msg := "ok"
	////
	////	mockClient.EXPECT().DeploymentServiceCreateDeploymentWithResponse(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
	////		func(_ context.Context, _ depapi.DeploymentServiceCreateDeploymentJSONRequestBody, _ ...depapi.RequestEditorFn) (*depapi.DeploymentServiceCreateDeploymentResponse, error) {
	////			resp := &depapi.DeploymentServiceCreateDeploymentResponse{
	////				JSONDefault: &depapi.Status{
	////					Message: &msg,
	////				},
	////			}
	////			return resp, nil
	////		}).
	////		AnyTimes()
	////
	////	mockClient.EXPECT().DeploymentServiceListDeploymentsWithResponse(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
	////		func(_ context.Context, _ *depapi.DeploymentServiceListDeploymentsParams, _ ...depapi.RequestEditorFn) (*depapi.DeploymentServiceListDeploymentsResponse, error) {
	////			resp := &depapi.DeploymentServiceListDeploymentsResponse{
	////				JSONDefault: &depapi.Status{
	////					Message: &msg,
	////				},
	////			}
	////			return resp, nil
	////		}).
	////		AnyTimes()
	////
	////	mockClient.EXPECT().DeploymentServiceGetDeploymentWithResponse(gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
	////		func(_ context.Context, _ string, _ ...depapi.RequestEditorFn) (*depapi.DeploymentServiceGetDeploymentResponse, error) {
	////			resp := &depapi.DeploymentServiceGetDeploymentResponse{
	////				JSONDefault: &depapi.Status{
	////					Message: &msg,
	////				},
	////				JSON200: &depapi.GetDeploymentResponse{
	////					Deployment: depapi.Deployment{
	////						AppName:    "test-app",
	////						AppVersion: "test-version",
	////					},
	////				},
	////			}
	////			return resp, nil
	////		}).
	////		AnyTimes()
	////
	////	mockClient.EXPECT().DeploymentServiceUpdateDeploymentWithResponse(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
	////		func(_ context.Context, _ string, _ depapi.DeploymentServiceUpdateDeploymentJSONRequestBody, _ ...depapi.RequestEditorFn) (*depapi.DeploymentServiceUpdateDeploymentResponse, error) {
	////			resp := &depapi.DeploymentServiceUpdateDeploymentResponse{
	////				JSONDefault: &depapi.Status{
	////					Message: &msg,
	////				},
	////			}
	////			return resp, nil
	////		}).
	////		AnyTimes()
	////
	////	mockClient.EXPECT().DeploymentServiceDeleteDeploymentWithResponse(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).DoAndReturn(
	////		func(_ context.Context, _ string, _ *depapi.DeploymentServiceDeleteDeploymentParams, _ ...depapi.RequestEditorFn) (*depapi.DeploymentServiceDeleteDeploymentResponse, error) {
	////			resp := &depapi.DeploymentServiceDeleteDeploymentResponse{
	////				JSONDefault: &depapi.Status{
	////					Message: &msg,
	////				},
	////			}
	////			return resp, nil
	////		}).
	////		AnyTimes()
	////
	//	return mockClient, nil
	//}
}

func (s *CLITestSuite) TearDownSuite() {
	auth.KeycloakFactory = nil
	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")
	viper.Set(auth.TrustCertField, "")
}

func (s *CLITestSuite) SetupTest() {
	s.proxy = restproxy.NewMockRestProxy(s.T())
	s.NotNil(s.proxy)
	err := s.login("u", "p")
	s.NoError(err)
}

func (s *CLITestSuite) TearDownTest() {
	s.NoError(s.proxy.Close())
	viper.Set(auth.UserName, "")
	viper.Set(auth.RefreshTokenField, "")
	viper.Set(auth.ClientIDField, "")
	viper.Set(auth.KeycloakEndpointField, "")
	viper.Set(auth.TrustCertField, "")
}

func TestCLI(t *testing.T) {
	t.Skip("defunct; to be reworked")
	suite.Run(t, &CLITestSuite{})
}

func (s *CLITestSuite) compareOutput(expected commandOutput, actual commandOutput) {
	for expectedK, expectedMap := range expected {
		actualMap := actual[expectedK]

		// Make sure there are no extra entries
		s.Equal(len(expectedMap), len(actualMap))

		// Make sure the entries match
		for k, v := range expectedMap {
			s.NotNil(actualMap[k])
			matches, _ := regexp.MatchString(v, actualMap[k])
			if !matches {
				s.True(matches, "Values don't match for %s", k)
			}
			s.True(matches, "Values don't match for %s", k)
		}
	}
}

func (s *CLITestSuite) runCommand(commandArgs string) (string, error) {
	c := s.proxy.RestClient().ClientInterface.(*restClient.Client)
	cmd := getRootCmd()
	args := strings.Fields(commandArgs)
	args = append(args, "--debug-headers")
	args = append(args, "--catalog-endpoint")
	args = append(args, c.Server)
	cmd.SetArgs(args)
	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	err := cmd.Execute()
	cmdOutput := stdout.String()
	return cmdOutput, err
}

func addCommandArgs(args commandArgs, commandString string) string {
	for argName, argValue := range args {
		commandString = commandString + fmt.Sprintf(` --%s %s `, argName, argValue)
	}
	return commandString
}

func mapCliOutput(output string) map[string]map[string]string {
	retval := make(map[string]map[string]string)
	lines := strings.Split(output, "\n")
	var headers []string

	for i, line := range lines {
		if i == 0 {
			// First line is the headers
			headers = strings.Split(line, "|")
		} else if line == "" {
			break
		} else {
			fields := strings.Fields(line)
			key := fields[0]
			retval[key] = make(map[string]string)

			for fieldNumber, field := range fields {
				headerKey := strings.Trim(strings.Trim(headers[fieldNumber], " "), "|")
				retval[key][headerKey] = strings.Trim(field, "|")
			}
		}
	}
	return retval
}

func mapVerboseCliOutput(output string) map[string]map[string]string {
	retval := make(map[string]map[string]string)
	lines := strings.Split(output, "\n")

	newOne := true
	key := ""

	for _, line := range lines {
		if line == "" {
			newOne = true
			continue
		}
		fields := strings.SplitN(line, ":", 2)
		value := strings.TrimSpace(fields[1])
		if newOne {
			newOne = false
			key = value
			retval[key] = make(map[string]string)
		}
		retval[key][fields[0]] = value
	}
	return retval
}

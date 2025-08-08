// SPDX-FileCopyrightText: (C) 2025 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/open-edge-platform/cli/pkg/auth"
	"github.com/spf13/viper"
)

func (s *CLITestSuite) login(u string, p string) error {
	cmd := getRootCmd()
	args := []string{"login", u, p, "--keycloak", kcTest, "--quiet"}
	cmd.SetArgs(args)
	stdout := new(bytes.Buffer)
	cmd.SetOut(stdout)
	err := cmd.Execute()
	//cmdOutput := stdout.String()
	return err
}

func (s *CLITestSuite) logout() error {
	cmd := getRootCmd()
	args := []string{"logout"}
	cmd.SetArgs(args)
	err := cmd.Execute()
	return err
}

func (s *CLITestSuite) TestLogin() {
	// empty user
	s.NoError(s.logout())
	err := s.login("", "")
	s.Error(err)
	s.Contains(err.Error(), "username cannot be blank")

	// Already a token present
	viper.Set(auth.RefreshTokenField, "bogus")
	err = s.login("u", "p")
	s.Contains(err.Error(), "already logged in - please logout first")

	// Attempt to log in - using mock should pass
	viper.Set(auth.RefreshTokenField, "")
	err = s.login("u", "p")
	s.NoError(err)

	s.NotEmpty(viper.Get(auth.RefreshTokenField))
	s.Equal("u", viper.Get("username"))
	s.Equal("system-client", viper.Get(auth.ClientIDField))
	s.Equal(kcTest, viper.Get(auth.KeycloakEndpointField))
	s.Equal(false, viper.GetBool(auth.TrustCertField))

	// Now call any function - should invoke auth.AddAuthHeader() and do the refresh flow
	_, err = s.listRegistries(project, false, true, "", "")
	s.NoError(err)
}

func (s *CLITestSuite) TestLogout() {
	dir, _ := os.MkdirTemp("", "")
	savedConfigFile := viper.ConfigFileUsed()

	defer func() {
		_ = os.RemoveAll(dir)
		viper.SetConfigFile(savedConfigFile)
	}()

	viper.SetConfigFile(dir)
	viper.Set(auth.RefreshTokenField, "bogus")
	s.Error(s.logout())
	s.Empty(viper.GetString(auth.RefreshTokenField))
	s.Empty(viper.GetString(auth.UserName))
	s.Empty(viper.GetString(auth.ClientIDField))
	s.Empty(viper.GetString(auth.KeycloakEndpointField))
	s.Empty(viper.GetString(auth.TrustCertField))
	s.NoError(s.logout())
}

func FuzzLogin(f *testing.F) {
	// Seed with some typical and edge-case inputs
	f.Add("", "")           // both empty
	f.Add("user", "")       // empty password
	f.Add("", "pass")       // empty username
	f.Add("user", "pass")   // normal login
	f.Add("user", "wrong")  // wrong password
	f.Add("admin", "admin") // common admin creds

	f.Fuzz(func(t *testing.T, username, password string) {
		testSuite := new(CLITestSuite)
		testSuite.SetT(t)
		testSuite.SetupSuite()
		defer testSuite.TearDownSuite()
		testSuite.SetupTest()
		defer testSuite.TearDownTest()

		// Always start with logout to clear state
		_ = testSuite.logout()

		// Simulate already logged in
		viper.Set(auth.RefreshTokenField, "bogus")
		err := testSuite.login(username, password)
		if viper.GetString(auth.RefreshTokenField) != "" {
			if err == nil || !strings.Contains(err.Error(), "already logged in") &&
				!strings.Contains(err.Error(), "accepts 1 arg(s), received 0") &&
				!strings.Contains(err.Error(), "accepts 1 arg(s), received 2") &&
				!strings.Contains(err.Error(), "accepts 1 arg(s), received 3") &&
				!strings.Contains(err.Error(), "unknown shorthand flag:") &&
				!strings.Contains(err.Error(), "unknown flag") {
				t.Errorf("Expected error for already logged in, got: %v", err)
			}
			// Clear token for next test
			viper.Set(auth.RefreshTokenField, "")
			return
		}

		// Test login with provided credentials
		err = testSuite.login(username, password)
		if username == "" {
			if err == nil || !strings.Contains(err.Error(), "username cannot be blank") {
				t.Errorf("Expected error for blank username, got: %v", err)
			}
			return
		}
		if password == "" {
			if err == nil || !strings.Contains(err.Error(), "password cannot be blank") {
				t.Errorf("Expected error for blank password, got: %v", err)
			}
			return
		}
		// Accept any error for wrong credentials, but no error for valid ones
		if username == "u" && password == "p" {
			if err != nil {
				t.Errorf("Unexpected error for valid login: %v", err)
			}
		}
	})
}

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

		// Test login with provided credentials
		err := testSuite.login(username, password)
		if err != nil && (strings.Contains(err.Error(), "not") ||
			strings.Contains(err.Error(), "unknown") ||
			strings.Contains(err.Error(), "match") ||
			strings.Contains(err.Error(), "invalid") ||
			strings.Contains(err.Error(), "required") ||
			strings.Contains(err.Error(), "requires") ||
			strings.Contains(err.Error(), "no such") ||
			strings.Contains(err.Error(), "missing") ||
			strings.Contains(err.Error(), "no") ||
			strings.Contains(err.Error(), "must") ||
			strings.Contains(err.Error(), "in form") ||
			strings.Contains(err.Error(), "incorrect") ||
			strings.Contains(err.Error(), "unexpected") ||
			strings.Contains(err.Error(), "expected") ||
			strings.Contains(err.Error(), "failed") ||
			strings.Contains(err.Error(), "is a") ||
			strings.Contains(err.Error(), "bad") ||
			strings.Contains(err.Error(), "exists") ||
			strings.Contains(err.Error(), "cannot") ||
			strings.Contains(err.Error(), "nonexistent") ||
			strings.Contains(err.Error(), "deleting") ||
			strings.Contains(err.Error(), "getting") ||
			strings.Contains(err.Error(), "listing") ||
			strings.Contains(err.Error(), "creating") ||
			strings.Contains(err.Error(), "Internal Server Error") ||
			strings.Contains(err.Error(), "accepts")) {
			t.Log("Expected error:", err)
		} else if !testSuite.NoError(err) {
			t.Errorf("Unexpected error for valid AMT Profile delete: %v", err)
		}
	})
}

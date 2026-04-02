// SPDX-FileCopyrightText: (C) 2026 Intel Corporation
// SPDX-License-Identifier: Apache-2.0

package keycloak

import (
	"context"
	"fmt"

	"github.com/open-edge-platform/cli/internal/cli/interfaces"
	kcapi "github.com/open-edge-platform/cli/pkg/rest/keycloak"
	"github.com/spf13/cobra"
	"go.uber.org/mock/gomock"
)

// CreateKeycloakAdminMock creates a mock Keycloak Admin factory function
func CreateKeycloakAdminMock(mctrl *gomock.Controller) interfaces.KeycloakAdminFactoryFunc {
	return func(cmd *cobra.Command) (context.Context, kcapi.ClientInterface, string, error) {
		mockClient := kcapi.NewMockClientInterface(mctrl)

		_ = cmd

		boolPtr := func(b bool) *bool { return &b }

		sampleUserID := "user-uuid-1234"
		sampleUser := kcapi.UserRepresentation{
			ID:        sampleUserID,
			Username:  "sample-user",
			Email:     "sample-user@sample-domain.com",
			FirstName: "sample",
			LastName:  "User",
			Enabled:   boolPtr(true),
		}

		adminUserID := "admin-uuid-0000"
		adminUser := kcapi.UserRepresentation{
			ID:        adminUserID,
			Username:  "admin",
			Email:     "admin@example.com",
			FirstName: "Admin",
			LastName:  "",
			Enabled:   boolPtr(true),
		}

		allGroups := []kcapi.GroupRepresentation{
			{ID: "group-uuid-1", Name: "org-admin-group", Path: "/org-admin-group"},
			{ID: "group-uuid-2", Name: "edge-manager-group", Path: "/edge-manager-group"},
			{ID: "group-uuid-3", Name: "edge-operator-group", Path: "/edge-operator-group"},
		}

		// Mock ListUsers
		mockClient.EXPECT().ListUsers(
			gomock.Any(), gomock.Any(),
		).Return([]kcapi.UserRepresentation{adminUser, sampleUser}, nil).AnyTimes()

		// Mock GetUserByUsername
		mockClient.EXPECT().GetUserByUsername(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, username string) (*kcapi.UserRepresentation, error) {
				switch username {
				case "sample-user":
					return &sampleUser, nil
				case "admin":
					return &adminUser, nil
				default:
					return nil, fmt.Errorf("user %q not found", username)
				}
			},
		).AnyTimes()

		// Mock GetUser
		mockClient.EXPECT().GetUser(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, userID string) (*kcapi.UserRepresentation, error) {
				switch userID {
				case sampleUserID:
					return &sampleUser, nil
				case adminUserID:
					return &adminUser, nil
				default:
					return nil, fmt.Errorf("user with ID %q not found", userID)
				}
			},
		).AnyTimes()

		// Mock CreateUser
		mockClient.EXPECT().CreateUser(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, user kcapi.UserRepresentation) error {
				if user.Username == "" {
					return fmt.Errorf("create user failed (400): username is required")
				}
				return nil
			},
		).AnyTimes()

		// Mock DeleteUser
		mockClient.EXPECT().DeleteUser(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, userID string) error {
				switch userID {
				case sampleUserID, adminUserID:
					return nil
				default:
					return fmt.Errorf("delete user failed (404): user not found")
				}
			},
		).AnyTimes()

		// Mock SetPassword
		mockClient.EXPECT().SetPassword(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil).AnyTimes()

		// Mock ListUserGroups
		mockClient.EXPECT().ListUserGroups(
			gomock.Any(), gomock.Any(), gomock.Any(),
		).DoAndReturn(
			func(_ context.Context, _ string, userID string) ([]kcapi.GroupRepresentation, error) {
				switch userID {
				case sampleUserID:
					return []kcapi.GroupRepresentation{allGroups[0]}, nil // org-admin-group
				default:
					return []kcapi.GroupRepresentation{}, nil
				}
			},
		).AnyTimes()

		// Mock AddUserToGroup
		mockClient.EXPECT().AddUserToGroup(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil).AnyTimes()

		// Mock RemoveUserFromGroup
		mockClient.EXPECT().RemoveUserFromGroup(
			gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(),
		).Return(nil).AnyTimes()

		// Mock ListGroups
		mockClient.EXPECT().ListGroups(
			gomock.Any(), gomock.Any(),
		).Return(allGroups, nil).AnyTimes()

		ctx := context.Background()
		return ctx, mockClient, "master", nil
	}
}

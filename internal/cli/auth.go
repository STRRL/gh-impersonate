package cli

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"github.com/strrl/gh-impersonate/internal/impersonate"
)

func newAuthCommand(v *viper.Viper) *cobra.Command {
	auth := &cobra.Command{
		Use:   "auth",
		Short: "Manage gh-impersonate credentials",
	}
	auth.AddCommand(newAuthLoginCommand(v))
	auth.AddCommand(newAuthStatusCommand(v))
	auth.AddCommand(newAuthLogoutCommand(v))
	return auth
}

func newAuthLoginCommand(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "login",
		Short: "Authorize a GitHub App user token through device flow",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			profileName, profile, err := loadSelectedProfile(v)
			if err != nil {
				return err
			}

			client := impersonate.NewGitHubClient()
			device, err := client.RequestDeviceCode(ctx, profile.ClientID)
			if err != nil {
				return err
			}

			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Open: %s\n", device.VerificationURI)
			fmt.Fprintf(out, "Code: %s\n", device.UserCode)
			fmt.Fprintln(out, "Waiting for authorization...")

			credential, err := client.PollDeviceToken(ctx, profile.ClientID, device)
			if err != nil {
				return err
			}
			if err := impersonate.SaveCredential(profileName, credential); err != nil {
				return err
			}
			fmt.Fprintf(out, "Logged in profile %q.\n", profileName)
			return nil
		},
	}
}

func newAuthStatusCommand(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Verify the selected profile credential with GitHub",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			profileName, credential, err := resolveCredential(ctx, v)
			if err != nil {
				return err
			}
			client := impersonate.NewGitHubClient()
			user, err := client.CurrentUser(ctx, credential.AccessToken)
			if err != nil {
				return err
			}
			out := cmd.OutOrStdout()
			fmt.Fprintf(out, "Profile: %s\n", profileName)
			fmt.Fprintf(out, "GitHub user: %s\n", user.Login)
			fmt.Fprintf(out, "Access token expires: %s\n", impersonate.FormatExpiry(credential.ExpiresAt))
			fmt.Fprintf(out, "Refresh token expires: %s\n", impersonate.FormatExpiry(credential.RefreshTokenExpiresAt))
			return nil
		},
	}
}

func newAuthLogoutCommand(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "logout",
		Short: "Delete the local credential for the selected profile",
		RunE: func(cmd *cobra.Command, args []string) error {
			profileName := selectedProfile(v)
			if err := impersonate.DeleteCredential(profileName); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged out profile %q.\n", profileName)
			return nil
		},
	}
}

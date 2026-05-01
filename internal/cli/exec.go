package cli

import (
	"context"
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newExecCommand(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "exec -- <gh args...>",
		Short: "Execute gh with the selected delegated agent identity",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return fmt.Errorf("missing gh command arguments")
			}
			_, credential, err := resolveCredential(context.Background(), v)
			if err != nil {
				return err
			}
			if os.Getenv("GH_TOKEN") != "" {
				fmt.Fprintln(cmd.ErrOrStderr(), "gh-impersonate: warning: overriding existing GH_TOKEN for impersonated command")
			}

			gh := exec.Command("gh", args...)
			gh.Stdin = os.Stdin
			gh.Stdout = os.Stdout
			gh.Stderr = os.Stderr
			gh.Env = withEnv(os.Environ(), "GH_TOKEN", credential.AccessToken)
			return gh.Run()
		},
	}
}

func withEnv(env []string, key string, value string) []string {
	prefix := key + "="
	out := make([]string, 0, len(env)+1)
	for _, item := range env {
		if len(item) >= len(prefix) && item[:len(prefix)] == prefix {
			continue
		}
		out = append(out, item)
	}
	return append(out, prefix+value)
}

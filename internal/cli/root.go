package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// SharedClientID is filled by release builds once the shared GitHub App exists.
// Development builds can still use profiles.default.client_id in config.yaml.
var SharedClientID = ""

func NewRootCommand() *cobra.Command {
	v := viper.New()
	v.SetEnvPrefix("GH_IMPERSONATE")
	v.AutomaticEnv()
	_ = v.BindEnv("profile", "GH_IMPERSONATE_PROFILE")

	var profile string
	root := &cobra.Command{
		Use:           "gh-impersonate",
		Short:         "Run gh commands through a GitHub App delegated agent identity",
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().StringVar(&profile, "profile", "", "App Identity Profile name")
	_ = v.BindPFlag("profile", root.PersistentFlags().Lookup("profile"))

	root.AddCommand(newAliasCommand(v))
	root.AddCommand(newAuthCommand(v))
	root.AddCommand(newExecCommand(v))
	return root
}

func selectedProfile(v *viper.Viper) string {
	if profile := v.GetString("profile"); profile != "" {
		return profile
	}
	return "default"
}

func fixedProfileFromFlag(cmd *cobra.Command) string {
	flag := cmd.Flags().Lookup("profile")
	if flag == nil {
		flag = cmd.Root().PersistentFlags().Lookup("profile")
	}
	if flag != nil && flag.Changed {
		return flag.Value.String()
	}
	return ""
}

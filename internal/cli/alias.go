package cli

import (
	"fmt"
	"strconv"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func newAliasCommand(v *viper.Viper) *cobra.Command {
	return &cobra.Command{
		Use:   "alias",
		Short: "Print a shell function that routes gh through gh-impersonate",
		RunE: func(cmd *cobra.Command, args []string) error {
			fixedProfile := fixedProfileFromFlag(cmd)
			fmt.Fprint(cmd.OutOrStdout(), aliasFunction(fixedProfile))
			return nil
		},
	}
}

func aliasFunction(fixedProfile string) string {
	profileArgs := ""
	if fixedProfile != "" {
		profileArgs = " --profile " + strconv.Quote(fixedProfile)
	}
	return fmt.Sprintf(`gh() {
  if [[ "${GH_IMPERSONATE:-}" != "1" ]]; then
    command gh "$@"
    return $?
  fi

  case "${1:-}" in
    impersonate|auth|extension|help|version|--version)
      command gh "$@"
      return $?
      ;;
  esac

  command gh impersonate%s exec -- "$@"
}
`, profileArgs)
}

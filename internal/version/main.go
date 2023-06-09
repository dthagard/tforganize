package version

import (
	"fmt"

	"github.com/spf13/cobra"

	info "github.com/dthagard/tfsort/internal/info"
)

func GetCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of tfsort",
		Long:  `The current version of this tfsort command-line tool.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tfsort %s\n", info.AppVersion)
		}}

	return versionCmd
}

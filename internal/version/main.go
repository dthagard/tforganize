package version

import (
	"fmt"

	"github.com/spf13/cobra"

	info "github.com/dthagard/tforganize/internal/info"
)

func GetCommand() *cobra.Command {
	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Print the version number of tforganize",
		Long:  `The current version of this tforganize command-line tool.`,
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Printf("tforganize %s\n", info.AppVersion)
		}}

	return versionCmd
}

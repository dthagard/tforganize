package sort

import (
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

// flags holds the CLI flags for the Sort command
var flags = &Params{}

func GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args:    cobra.ExactArgs(1),
		Example: `tforganize sort main.tf`,
		Long:    `Sort reads a Terraform file or folder and sorts the resources found alphabetically ascending by resource type and name.`,
		Run: func(cmd *cobra.Command, args []string) {
			Sort(args[0], flags)
		},
		Short: "Sort a Terraform file or folder.",
		Use:   "sort <file | folder>",
	}

	SetFileSystem(afero.NewOsFs())

	setFlags(cmd)

	return cmd
}

func setFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&flags.GroupByType, "group-by-type", "g", false, "organize the resources by type in the output files")
	cmd.PersistentFlags().BoolVarP(&flags.HasHeader, "has-header", "e", false, "the input files have a header")
	cmd.PersistentFlags().StringVarP(&flags.HeaderPattern, "header-pattern", "p", "", "the header pattern to find the header in the input files")
	cmd.PersistentFlags().BoolVarP(&flags.KeepHeader, "keep-header", "k", false, "keep the header matched in the header pattern in the output files")
	cmd.PersistentFlags().StringVarP(&flags.OutputDir, "output-dir", "o", "", "output the results to a specific folder")
	cmd.PersistentFlags().BoolVarP(&flags.RemoveComments, "remove-comments", "r", false, "remove comments in the sorted file(s)")
}

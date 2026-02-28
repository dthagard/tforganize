package sort

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// flags holds the CLI flags for the Sort command
var flags = &Params{}

func GetCommand() *cobra.Command {
	cmd := &cobra.Command{
		Args: cobra.MaximumNArgs(1),
		Example: `  tforganize sort main.tf
  tforganize sort ./terraform/
  cat main.tf | tforganize sort -`,
		Long: `Sort reads a Terraform file or folder and sorts the resources found alphabetically ascending by resource type and name.

When the argument is "-" or omitted and stdin is piped, input is read from stdin and the sorted output is written to stdout.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Determine if we should read from stdin.
			readStdin := false
			if len(args) == 0 {
				// No args: check if stdin is a pipe.
				stat, _ := os.Stdin.Stat()
				if stat != nil && (stat.Mode()&os.ModeCharDevice) == 0 {
					readStdin = true
				} else {
					return fmt.Errorf("no target specified; provide a file/folder path or pipe content via stdin")
				}
			} else if args[0] == "-" {
				readStdin = true
			}

			if readStdin {
				if flags.Inline {
					return fmt.Errorf("the --inline flag cannot be used with stdin")
				}
				content, err := io.ReadAll(os.Stdin)
				if err != nil {
					return fmt.Errorf("could not read stdin: %w", err)
				}
				sorted, err := SortBytes(content, "stdin.tf", flags)
				if err != nil {
					return err
				}
				fmt.Print(string(sorted))
				return nil
			}

			return Sort(args[0], flags)
		},
		Short: "Sort a Terraform file or folder.",
		Use:   "sort [file | folder | -]",
	}

	setFlags(cmd)

	return cmd
}

func setFlags(cmd *cobra.Command) {
	cmd.PersistentFlags().BoolVarP(&flags.GroupByType, "group-by-type", "g", false, "organize the resources by type in the output files")
	cmd.PersistentFlags().BoolVarP(&flags.HasHeader, "has-header", "e", false, "the input files have a header")
	cmd.PersistentFlags().StringVarP(&flags.HeaderPattern, "header-pattern", "p", "", "the header pattern to find the header in the input files")
	cmd.PersistentFlags().StringVar(&flags.HeaderEndPattern, "header-end-pattern", "", "pattern marking the end of a multi-line header block (e.g. '**/' or '*/')")
	cmd.PersistentFlags().BoolVarP(&flags.KeepHeader, "keep-header", "k", false, "keep the header matched in the header pattern in the output files")
	cmd.PersistentFlags().BoolVarP(&flags.Inline, "inline", "i", false, "sort the resources in the input file(s) in place")
	cmd.PersistentFlags().StringVarP(&flags.OutputDir, "output-dir", "o", "", "output the results to a specific folder")
	cmd.PersistentFlags().BoolVarP(&flags.RemoveComments, "remove-comments", "r", false, "remove comments in the sorted file(s)")
	cmd.PersistentFlags().BoolVarP(&flags.Check, "check", "c", false, "check whether files are already sorted without writing changes; exits non-zero if any file would change")
	cmd.PersistentFlags().BoolVarP(&flags.Recursive, "recursive", "R", false, "recursively sort all nested directories containing .tf files")
	cmd.PersistentFlags().BoolVar(&flags.Diff, "diff", false, "show a unified diff of changes instead of writing files")
	cmd.PersistentFlags().BoolVar(&flags.NoSortByType, "no-sort-by-type", false, "sort blocks alphabetically by type instead of using logical type ordering")
	cmd.PersistentFlags().BoolVar(&flags.StripSectionComments, "strip-section-comments", false, "remove section-divider comments (e.g. # === Section ===, # ---) from the output")
	cmd.PersistentFlags().StringArrayVarP(&flags.Excludes, "exclude", "x", []string{}, "glob pattern to exclude from sorting (repeatable; supports **); e.g. --exclude '.terraform/**'")
}

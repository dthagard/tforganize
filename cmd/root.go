package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/dthagard/tforganize/v2/internal/info"
	"github.com/dthagard/tforganize/v2/internal/sort"
	"github.com/dthagard/tforganize/v2/internal/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	envPrefix = "TFORGANIZE"
)

var (
	// Used for flags.
	config string
)

type RootCommand struct {
	baseCmd *cobra.Command
}

func NewRootCommand() *RootCommand {
	rootCommand := &RootCommand{
		baseCmd: &cobra.Command{
			PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
				toggleDebug(cmd, args)
				return initConfig(cmd, args)
			},
			Short: "tforganize is a tool for sorting Terraform files and folders.",
		},
	}

	rootCommand.setFlags()
	rootCommand.registerSubCommands()

	return rootCommand
}

func (rc *RootCommand) setFlags() {
	rc.baseCmd.PersistentFlags().StringVar(&config, "config", "", "config file (default: project root, then $HOME/.tforganize.yaml)")
	rc.baseCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "verbose logging")
}

func (rc *RootCommand) registerSubCommands() {
	rc.baseCmd.AddCommand(
		sort.GetCommand(),
		version.GetCommand(),
	)
}

// Exit code 2 is used for --check failures; exit code 1 for all other errors.
//
// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func (rc *RootCommand) Execute() {
	rc.execute(os.Exit)
}

func (rc *RootCommand) execute(exit func(int)) {
	err := rc.baseCmd.Execute()
	if err != nil {
		if errors.Is(err, sort.ErrCheckFailed) {
			exit(2)
			return
		}
		exit(1)
	}
}

func initConfig(cmd *cobra.Command, args []string) error {
	log.Traceln("Starting initConfig()")

	v := viper.New()

	if config != "" {
		// Use config file from the flag.
		v.SetConfigFile(config)
	} else {
		project, err := projectConfigDir()
		if err != nil {
			return err
		}

		// Find home directory.
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}

		// Select the project config before home; do not merge the files.
		v.AddConfigPath(project)
		v.AddConfigPath(home)
		v.SetConfigType("yaml")
		v.SetConfigName(".tforganize")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debugln("No config file found")
		} else {
			return fmt.Errorf("error reading config file: %w", err)
		}
	} else {
		log.WithField("configFile", v.ConfigFileUsed()).Debugln("Found config file")
		log.WithField("config", v.AllKeys()).Debugln("Config file contents")
	}

	// When we bind flags to environment variables expect that the
	// environment variables are prefixed, e.g. a flag like --number
	// binds to an environment variable STING_NUMBER. This helps
	// avoid conflicts.
	v.SetEnvPrefix(envPrefix)

	// Environment variables can't have dashes in them, so bind them to their equivalent
	// keys with underscores, e.g. --favorite-color to TFORGANIZE_FAVORITE_COLOR
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	bindFlags(cmd, v)

	v.SetDefault("author", fmt.Sprintf("%s <%s>", info.AppRepoOwner, info.AppRepoOwnerEmail))
	v.SetDefault("license", info.AppLicense)
	return nil
}

// projectConfigDir finds the nearest Git root, or uses cwd outside Git.
// A .git file marks a worktree or submodule just as a directory marks a checkout.
func projectConfigDir() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for dir := cwd; ; dir = filepath.Dir(dir) {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir, nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if filepath.Dir(dir) == dir {
			return cwd, nil
		}
	}
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		configName := f.Name

		if !f.Changed && v.IsSet(configName) {
			// StringArray/StringSlice flags require one Set() call per element.
			// fmt.Sprintf("%v", val) would produce "[a b]" which pflag rejects.
			if f.Value.Type() == "stringArray" || f.Value.Type() == "stringSlice" {
				for _, s := range v.GetStringSlice(configName) {
					_ = cmd.Flags().Set(f.Name, s)
				}
			} else {
				val := v.Get(configName)
				_ = cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
			}
		}
	})
}

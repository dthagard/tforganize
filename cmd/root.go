/*
Copyright © 2023 Dan Thagard <dthagard@gmail.com
*/
package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/dthagard/tfsort/internal/info"
	"github.com/dthagard/tfsort/internal/sort"
	"github.com/dthagard/tfsort/internal/version"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	envPrefix = "TFSORT"
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
			PersistentPreRun: func(cmd *cobra.Command, args []string) {
				toggleDebug(cmd, args)
				initConfig(cmd, args)
			},
			Short: "TFSort is a tool for sorting Terraform files and folders.",
		},
	}

	rootCommand.setFlags()
	rootCommand.registerSubCommands()

	return rootCommand
}

func (rc *RootCommand) setFlags() {
	rc.baseCmd.PersistentFlags().StringVar(&config, "config", "", "config file (default is $HOME/.tfsort.yaml)")
	rc.baseCmd.PersistentFlags().BoolVarP(&debug, "debug", "d", false, "verbose logging")
}

func (rc *RootCommand) registerSubCommands() {
	rc.baseCmd.AddCommand(
		sort.GetCommand(),
		version.GetCommand(),
	)
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func (rc *RootCommand) Execute() {
	err := rc.baseCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func initConfig(cmd *cobra.Command, args []string) {
	log.Traceln("Starting initConfig()")

	v := viper.New()

	if config != "" {
		// Use config file from the flag.
		v.SetConfigFile(config)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".tfsort" (without extension).
		v.AddConfigPath(home)
		v.SetConfigType("yaml")
		v.SetConfigName(".tfsort")
	}

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			log.Debugln("No config file found")
		} else {
			log.WithError(err).Fatalln("Error reading config file")
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
	// keys with underscores, e.g. --favorite-color to TFSORT_FAVORITE_COLOR
	v.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	bindFlags(cmd, v)

	v.SetDefault("author", fmt.Sprintf("%s <%s>", info.AppRepoOwner, info.AppRepoOwnerEmail))
	v.SetDefault("license", info.AppLicense)
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Determine the naming convention of the flags when represented in the config file
		configName := f.Name

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(configName) {
			val := v.Get(configName)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

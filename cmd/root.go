package cmd

import (
	"fmt"
	"strings"

	"github.com/acouvreur/sablier/v2/config"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// The name of our config file, without the file extension because viper supports many different config file languages.
	defaultConfigFilename = "config"
)

var (
	rootCmd = &cobra.Command{
		Use:   "sablier",
		Short: "A webserver to start container on demand",
		Long: `Sablier is an API that start containers on demand.
It provides an integrations with multiple reverse proxies and different loading strategies.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
			return initializeConfig(cmd)
		},
	}
)

// Execute executes the root command.
func Execute() error {
	return rootCmd.Execute()
}

var conf = config.NewConfig()

func init() {

	rootCmd.AddCommand(startCmd)
	// Provider flags
	startCmd.Flags().StringVar(&conf.Provider.Name, "provider.name", "docker", fmt.Sprintf("Provider to use to manage containers %v", config.GetProviders()))
	viper.BindPFlag("provider.name", startCmd.Flags().Lookup("provider.name"))
	// Server flags
	startCmd.Flags().IntVar(&conf.Server.Port, "server.port", 10000, "The server port to use")
	viper.BindPFlag("server.port", startCmd.Flags().Lookup("server.port"))
	startCmd.Flags().StringVar(&conf.Server.BasePath, "server.base-path", "/", "The base path for the API")
	viper.BindPFlag("server.base-path", startCmd.Flags().Lookup("server.base-path"))
	// Storage flags
	startCmd.Flags().StringVar(&conf.Storage.File, "storage.file", "", "File path to save the state")
	viper.BindPFlag("storage.file", startCmd.Flags().Lookup("storage.file"))

	rootCmd.AddCommand(versionCmd)
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Set the base name of the config file, without the file extension.
	v.SetConfigName(defaultConfigFilename)

	// Set as many paths as you like where viper should look for the
	// config file. We are only looking in the current working directory.
	v.AddConfigPath(".")

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	// Bind to environment variables
	// Works great for simple config names, but needs help for names
	// like --favorite-color which we fix in the bindFlags function
	v.AutomaticEnv()

	// Bind the current command's flags to viper
	bindFlags(cmd, v)

	return nil
}

// Bind each cobra flag to its associated viper configuration (config file and environment variable)
func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
		envVarSuffix = strings.ToUpper(strings.ReplaceAll(envVarSuffix, ".", "_"))
		v.BindEnv(f.Name, envVarSuffix)

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/acouvreur/sablier/config"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"
)

const (
	// The name of our config file, without the file extension because viper supports many different config file languages.
	defaultConfigFilename = "sablier"
)

var conf = config.NewConfig()
var cfgFile string

func Execute() {
	cmd := NewRootCommand()
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func NewRootCommand() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sablier",
		Short: "A webserver to start container on demand",
		Long: `Sablier is an API that start containers on demand.
It provides an integrations with multiple reverse proxies and different loading strategies.`,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			// You can bind cobra and viper in a few locations, but PersistencePreRunE on the root command works well
			return initializeConfig(cmd)
		},
	}

	rootCmd.PersistentFlags().StringVar(&cfgFile, "configFile", "", "Config file path. If not defined, looks for sablier.(yml|yaml|toml) in /etc/sablier/ > $XDG_CONFIG_HOME > $HOME/.config/ and current directory")

	startCmd := newStartCommand()
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
	// Sessions flags
	startCmd.Flags().DurationVar(&conf.Sessions.DefaultDuration, "sessions.default-duration", time.Duration(5)*time.Minute, "The default session duration")
	viper.BindPFlag("sessions.default-duration", startCmd.Flags().Lookup("sessions.default-duration"))
	startCmd.Flags().DurationVar(&conf.Sessions.ExpirationInterval, "sessions.expiration-interval", time.Duration(20)*time.Second, "The expiration checking interval. Higher duration gives less stress on CPU. If you only use sessions of 1h, setting this to 5m is a good trade-off.")
	viper.BindPFlag("sessions.expiration-interval", startCmd.Flags().Lookup("sessions.expiration-interval"))

	// logging level
	rootCmd.PersistentFlags().StringVar(&conf.Logging.Level, "logging.level", log.InfoLevel.String(), "The logging level. Can be one of [panic, fatal, error, warn, info, debug, trace]")
	viper.BindPFlag("logging.level", rootCmd.PersistentFlags().Lookup("logging.level"))

	// strategy
	startCmd.Flags().StringVar(&conf.Strategy.Dynamic.CustomThemesPath, "strategy.dynamic.custom-themes-path", "", "Custom themes folder, will load all .html files recursively")
	viper.BindPFlag("strategy.dynamic.custom-themes-path", startCmd.Flags().Lookup("strategy.dynamic.custom-themes-path"))
	startCmd.Flags().StringVar(&conf.Strategy.Dynamic.DefaultTheme, "strategy.dynamic.default-theme", "hacker-terminal", "Default theme used for dynamic strategy")
	viper.BindPFlag("strategy.dynamic.default-theme", startCmd.Flags().Lookup("strategy.dynamic.default-theme"))
	startCmd.Flags().BoolVar(&conf.Strategy.Dynamic.ShowDetailsByDefault, "strategy.dynamic.show-details-by-default", true, "Show the loading instances details by default")
	viper.BindPFlag("strategy.dynamic.show-details-by-default", startCmd.Flags().Lookup("strategy.dynamic.show-details-by-default"))
	startCmd.Flags().DurationVar(&conf.Strategy.Dynamic.DefaultRefreshFrequency, "strategy.dynamic.default-refresh-frequency", 5*time.Second, "Default refresh frequency in the HTML page for dynamic strategy")
	viper.BindPFlag("strategy.dynamic.default-refresh-frequency", startCmd.Flags().Lookup("strategy.dynamic.default-refresh-frequency"))
	startCmd.Flags().DurationVar(&conf.Strategy.Blocking.DefaultTimeout, "strategy.blocking.default-timeout", 1*time.Minute, "Default timeout used for blocking strategy")
	viper.BindPFlag("strategy.blocking.default-timeout", startCmd.Flags().Lookup("strategy.blocking.default-timeout"))

	rootCmd.AddCommand(startCmd)
	rootCmd.AddCommand(versionCmd)

	healthCmd.Flags().String("url", "http://localhost:10000/health", "Sablier health endpoint")
	rootCmd.AddCommand(healthCmd)

	return rootCmd
}

func initializeConfig(cmd *cobra.Command) error {
	v := viper.New()

	// Set the base name of the config file, without the file extension.
	v.SetConfigName(defaultConfigFilename)

	v.AddConfigPath("/etc/sablier/")
	v.AddConfigPath("$XDG_CONFIG_HOME")
	v.AddConfigPath("$HOME/.config/")
	v.AddConfigPath(".")

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	}

	// Attempt to read the config file, gracefully ignoring errors
	// caused by a config file not being found. Return an error
	// if we cannot parse the config file.
	if err := v.ReadInConfig(); err != nil {
		// It's okay if there isn't a config file
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		} else if cfgFile != "" {
			// But if we explicitely defined the config file it should return the error
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

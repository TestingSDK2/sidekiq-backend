package cmd

import (
	"time"

	"github.com/ProImaging/sidekiq-backend/sidekiq-people/cmd/server"

	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/crypto/ssh/terminal"
	"golang.org/x/sys/unix"
)

func New() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sidekiq",
		Short: "Rockstar Systems Sidekiq Application",
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			verbose, _ := cmd.Flags().GetBool("verbose")
			if verbose {
				logrus.SetLevel(logrus.DebugLevel)
			}

			if !verbose && !terminal.IsTerminal(unix.Stdout) {
				logrus.SetFormatter(&logrus.JSONFormatter{
					TimestampFormat: time.RFC3339Nano,
				})
			} else {
				logrus.SetFormatter(&logrus.TextFormatter{
					ForceColors:     true,
					FullTimestamp:   true,
					TimestampFormat: time.RFC3339Nano,
				})
			}
		},
	}

	var configFile string
	var initConfig = func() {
		if configFile != "" {
			viper.SetConfigFile(configFile)
		} else {
			viper.SetConfigName("default")
			viper.AddConfigPath(".")
			viper.AddConfigPath("/etc/sidekiq")
			viper.AddConfigPath("$HOME/.sidekiq")
		}
		// viper.SetConfigType("yaml")
		viper.AutomaticEnv()

		if err := viper.ReadInConfig(); err != nil {
			logrus.WithError(err).Fatalf("unable to read config from file")
		}
	}

	cobra.OnInitialize(initConfig)
	cmd.PersistentFlags().BoolP("verbose", "v", false, "make output more verbose")
	cmd.PersistentFlags().StringVar(&configFile, "config", "", "config file (default is default.yaml)")

	cmd.AddCommand(
		NewVersionCommand(),
		server.NewServeCommand(),
	)
	return cmd
}

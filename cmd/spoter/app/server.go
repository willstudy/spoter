package app

import (
	"fmt"
	"os"
  "context"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

  "github.com/willstudy/spoter/pkg/configs"
  "github.com/willstudy/spoter/pkg/spoter"
)

var configFile string

var serverCmd = &cobra.Command{
	Use:           "server",
	Short:         "Lanuch spoter.",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		newLog := log.New()
		newLog.SetLevel(log.Level(debugLevel))
		logger := newLog.WithFields(log.Fields{
			"app": "spoter",
		})

		kubeConfig := os.Getenv(configs.KubeConfig)
		if kubeConfig == "" {
			return fmt.Errorf("Can not find kube config, please provide kube config file.")
		}

    if configFile == "" {
      return fmt.Errorf("Can not find args for --configFile, please provide config file.")
    }

		cfg := &spoter.ControllerConfig{
      ConfigFile: configFile,
			Logger:     logger,
		}
		controller, err := spoter.NewSpoterController(cfg)
		if err != nil {
			return err
		}

		ctx := context.TODO()
		quit := make(chan struct{})

		return controller.Serve(ctx, quit)
	},
}

func init() {
	serverCmd.Flags().StringVar(&configFile, "configFile", "",
		"spoter's config file")

	rootCmd.AddCommand(serverCmd)
}

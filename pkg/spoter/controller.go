package spoter

import (
  "context"
  "fmt"
  "os"

  log "github.com/sirupsen/logrus"
)

type SpoterControllerInterface interface {
  Serve(ctx context.Context, quit <-chan struct{}) error
}

type ControllerConfig struct {
  ConfigFile string
  Logger     *log.Entry
}

type spoterController struct {
  configFile string
  logger     *log.Entry
}

func NewSpoterController(config *ControllerConfig) (SpoterControllerInterface, error) {
  if err := checkControllerConfig(config); err != nil {
    return nil, fmt.Errorf("config failed with %v", err)
  }
  return &spoterController{
    configFile: config.ConfigFile,
    logger:     config.Logger,
  }, nil
}

func checkControllerConfig(config *ControllerConfig) error {
  if config.Logger == nil {
    config.Logger = log.WithFields(log.Fields{
			"app": "spoter",
		})
  }

  if _, err := os.Stat(config.ConfigFile); err != nil {
    return fmt.Errorf("config file not existed.")
  }

  return nil
}

func (s *spoterController) Serve(ctx context.Context, quit <-chan struct{}) error {
  //TODO: controller
  return nil
}

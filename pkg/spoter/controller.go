package spoter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"time"

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

func (s *spoterController) parseConfigs() (SpoterConfig, error) {
	logger := s.logger.WithFields(log.Fields{
		"func": "parseConfigs",
	})
	var m SpoterConfig
	data, err := ioutil.ReadFile(s.configFile)
	if err != nil {
		logger.Errorf("Failed to read config file, due to %v", err)
		return m, err
	}

	if err := json.Unmarshal([]byte(data), &m); err != nil {
		logger.Errorf("Json Unmarshal failed with %v", err)
		return m, err
	}

	return m, nil
}

func (s *spoterController) getClusterStatus() (SpoterModel, error) {
	var r SpoterModel
	// TODO: get cluster status
	return r, nil
}

func (s *spoterController) rebalance(config, status SpoterModel) {
	// TODO: rebalance the k8s cluster
	return
}

func (s *spoterController) Serve(ctx context.Context, quit <-chan struct{}) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "Serve",
	})
	config, err := s.parseConfigs()
	if err != nil {
		logger.Errorf("Parse config failed with %v", err)
		return err
	}
	logger.Debugf("config content: %#v", config)

	for {
		select {
		case <-quit:
			logger.Debug("Receive TERM, exit.")
			break
		default:
			clusterStatus, err := s.getClusterStatus()
			if err != nil {
				logger.Warnf("Failed to get cluster status, due to %v", err)
				continue
			} else {
				logger.Debugf("Cluster Status: %v", clusterStatus)
				s.rebalance(config.Model, clusterStatus)
				logger.Debugf("Rebalance Done.")
			}
		}
		time.Sleep(time.Duration(config.CheckInterval) * time.Second)
		config, err = s.parseConfigs()
		if err != nil {
			logger.Warnf("Parse config failed with %v", err)
		}
	}
	return nil
}

package spoter

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/common"
	"github.com/willstudy/spoter/pkg/configs"
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
	logger := s.logger.WithFields(log.Fields{
		"func": "getClusterStatus",
	})

	r := make(map[string]MachineInfo)
	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubectlCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"get",
		"no",
		"--show-labels",
	}
	logger.Infof("CMD: %v.", cmds)

	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		output, err := common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v.", i, err)
		} else {
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if !strings.Contains(line, configs.AliyunECSLabel) {
					continue
				}
				logger.Debugf("line: %s", line)
				fields := strings.Split(line, " ")
				logger.Debugf("labels: %s", fields[len(fields)-1])
				labels := strings.Split(fields[len(fields)-1], ",")
				for _, field := range labels {
					if strings.Contains(field, configs.AliyunECSLabel) {
						logger.Debugf("label: %s", field)
						keys := strings.Split(field, "=")
						if _, ok := r[keys[0]]; ok {
							m := r[keys[1]]
							m.Num += 1
							r[keys[1]] = m
						} else {
							var m MachineInfo
							m.Num = 1
							r[keys[1]] = m
						}
					}
				}
			}
			return r, nil
		}
	}
	logger.Debugf("result: %v", r)
	return r, nil
}

func (s *spoterController) rebalance(config, status SpoterModel) {
	logger := s.logger.WithFields(log.Fields{
		"func": "rebalance",
	})

	for label, machineInfo := range config {
		if _, ok := status[label]; !ok {
			for i := 0; int32(i) < machineInfo.Num; i++ {
				logger.Infof("Add a machine, label: %s, price: %v\n", label, machineInfo.Price)
				s.joinNode(label, machineInfo.Price)
			}
		} else {
			delta := machineInfo.Num - status[label].Num
			if delta > 0 {
				for i := 0; int32(i) < delta; i++ {
					logger.Infof("Add a machine, label: %s, price: %v\n", label, machineInfo.Price)
					s.joinNode(label, machineInfo.Price)
				}
			}
		}
	}
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
	/*
		var resp AllocMachineResponse
		str := "{\"EipAddress\":\"39.105.2.200\",\"msg\":\"CreateECSsuccessfully.\",\"Hostname\":\"iZ2zeifctth7468lg6e225Z\",\"code\":0}"
		logger.Debugf("str: %v\n", str)
		if err = json.Unmarshal([]byte(str), &resp); err != nil {
			logger.Errorf("Json Unmarshal failed with %v", err)
			//return "", "", err
		}
		logger.Debugf("resp: %#v\n", resp)
	*/
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

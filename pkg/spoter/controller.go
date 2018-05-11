package spoter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
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

type K8sMachine struct {
	Hostname     string
	ImageId      string
	Region       string
	InstanceType string

	SpotWithPriceLimit float64
	BandWidth          int32

	InstanceID string
	PublicIP   string
	PrivateIP  string
	Status     string
}

type spoterController struct {
	configFile string
	logger     *log.Entry
	dbCon      *sql.DB
	k8sMachine map[string]K8sMachine
}

func NewSpoterController(config *ControllerConfig) (SpoterControllerInterface, error) {
	if err := checkControllerConfig(config); err != nil {
		return nil, fmt.Errorf("config failed with %v", err)
	}

	db, err := sql.Open("mysql", configs.SQLDSN)
	if err != nil {
		return nil, fmt.Errorf("open mysql failed with %v", err)
	}

	return &spoterController{
		configFile: config.ConfigFile,
		logger:     config.Logger,
		dbCon:      db,
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

func (s *spoterController) initFromDB() error {
	logger := s.logger.WithFields(log.Fields{
		"func": "initFromDB",
	})

	sql := "select hostname, image_id, region, instance_type, spot_price_limit,"
	sql += " bandwith, instance_id, public_ip, private_ip, status from machine_info"
	logger.Debugf("sql: %s", sql)

	rows, err := s.dbCon.Query(sql)
	defer rows.Close()
	if err != nil {
		return err
	}

	s.k8sMachine = make(map[string]K8sMachine)

	for rows.Next() {
		var m K8sMachine
		if err := rows.Scan(&m.Hostname, &m.ImageId, &m.Region, &m.InstanceType,
			&m.SpotWithPriceLimit, &m.BandWidth, &m.InstanceID, &m.PublicIP,
			&m.PrivateIP, &m.Status); err != nil {
			logger.Fatal("failed to read from row, due to %v", err)
			return err
		}
		if m.Status == configs.MachineDeleted {
			continue
		}
		s.k8sMachine[m.InstanceID] = m
		logger.Debugf("Load a machine from DB, machine info: %v", m)
	}
	return nil
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
				s.joinNode(label, machineInfo.Price, machineInfo.BandWidth)
			}
		} else {
			delta := machineInfo.Num - status[label].Num
			if delta > 0 {
				for i := 0; int32(i) < delta; i++ {
					logger.Infof("Add a machine, label: %s, price: %v\n", label, machineInfo.Price)
					s.joinNode(label, machineInfo.Price, machineInfo.BandWidth)
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

	if err = s.initFromDB(); err != nil {
		logger.Errorf("Init from db failed with %v", err)
	}
	// 恢复正在删除的 machine 和 刚刚创建的 machine 的动作
	s.restoreFromDB()

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

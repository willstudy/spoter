package spoter

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
	log "github.com/sirupsen/logrus"

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
	lock       *sync.Mutex
}

func NewSpoterController(config *ControllerConfig) (SpoterControllerInterface, error) {
	if err := checkControllerConfig(config); err != nil {
		return nil, fmt.Errorf("config failed with %v", err)
	}

	db, err := sql.Open("mysql", configs.SQLDSN)
	if err != nil {
		return nil, err
	}

	return &spoterController{
		configFile: config.ConfigFile,
		logger:     config.Logger,
		dbCon:      db,
		lock:       new(sync.Mutex),
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

func (s *spoterController) loadMachineInfo() error {
	logger := s.logger.WithFields(log.Fields{
		"func": "loadMachineInfo",
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

	// 对 k8sMachine 加锁
	s.lock.Lock()
	defer s.lock.Unlock()
	for rows.Next() {
		var m K8sMachine
		if err := rows.Scan(&m.Hostname, &m.ImageId, &m.Region, &m.InstanceType,
			&m.SpotWithPriceLimit, &m.BandWidth, &m.InstanceID, &m.PublicIP,
			&m.PrivateIP, &m.Status); err != nil {
			logger.Fatal("failed to read from row, due to %v", err)
			return err
		}
		if m.Status == configs.MachineDeleted {
			logger.Debugf("Machine: %s has deleted, skipped.\n", m.Hostname)
			continue
		}

		// 如果存在的话，则删除该记录
		if _, ok := s.k8sMachine[m.InstanceID]; ok {
			delete(s.k8sMachine, m.InstanceID)
		}

		s.k8sMachine[m.InstanceID] = m
		logger.Debugf("Load a machine from DB, machine info: %v", m)
	}
	return nil
}

func (s *spoterController) getInstanceID(instanceType string) string {
	// 对 k8sMachine 加锁
	s.lock.Lock()
	defer s.lock.Unlock()
	for i, m := range s.k8sMachine {
		if m.InstanceType == instanceType {
			return i
		}
	}
	return ""
}

func (s *spoterController) rebalance(config SpoterModel) {
	logger := s.logger.WithFields(log.Fields{
		"func": "rebalance",
	})
	// 目前集群实例型号的数目信息
	insTypeInfo := make(map[string]int32)
	for _, m := range s.k8sMachine {
		if _, ok := insTypeInfo[m.InstanceType]; ok {
			insTypeInfo[m.InstanceType] += 1
		} else {
			insTypeInfo[m.InstanceType] = 1
		}
	}

	for label, machineInfo := range config {
		if _, ok := insTypeInfo[label]; !ok {
			for i := 0; int32(i) < machineInfo.Num; i++ {
				logger.Infof("Add a machine, label: %s, price: %v\n", label, machineInfo.Price)
				s.joinNode(label, machineInfo.Price, machineInfo.BandWidth)
			}
		} else {
			delta := machineInfo.Num - insTypeInfo[label]
			if delta > 0 {
				for i := 0; int32(i) < delta; i++ {
					logger.Infof("Add a machine, label: %s, price: %v\n", label, machineInfo.Price)
					s.joinNode(label, machineInfo.Price, machineInfo.BandWidth)
				}
			} else {
				for i := 0; int32(i) < abs(delta); i++ {
					insID := s.getInstanceID(label)
					logger.Infof("Delete a machine, instanceID: %s\n", insID)
					s.deleteNode(insID)
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

	if err = s.loadMachineInfo(); err != nil {
		logger.Errorf("Init from db failed with %v", err)
	}
	logger.Debugf("Machine status: %v", s.k8sMachine)

	// 恢复正在删除的 machine 和 其他中断的 machine 的动作
	s.restoreFromDB()

	// 后台不停的检测 spot instance 是否过期
	go s.detectController(ctx, quit)

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
			// 优雅退出
			logger.Debug("Receive TERM, exit.")
			break
		default:
			logger.Debugf("Machine status: %v", s.k8sMachine)
			s.rebalance(config.Model)
			logger.Debugf("Rebalance Done.")
		}
		time.Sleep(time.Duration(config.CheckInterval) * time.Second)
		config, err = s.parseConfigs()
		if err != nil {
			logger.Warnf("Parse config failed with %v", err)
		}
	}
	return nil
}

package spoter

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/common"
	"github.com/willstudy/spoter/pkg/configs"
)

func (s *spoterController) detectController(ctx context.Context, quit <-chan struct{}) {
	logger := s.logger.WithFields(log.Fields{
		"func": "detectSpotInstance",
	})

	for {
		select {
		case <-quit:
			// 优雅退出
			logger.Debug("Receive TERM, exit.")
			break
		default:
			// 遍历每一个 running 的 machine
			for i, m := range s.k8sMachine {
				if m.Status == configs.MachineRunning {
					logger.Debugf("instance id: %s is running, begin to detect this instance.\n", i)
					s.detectInstance(ctx, m.PrivateIP, i)
				}
			}
			logger.Debugf("Detect Done.")
		}
		time.Sleep(30 * time.Second)
	}
	return
}

func (s *spoterController) detectInstance(ctx context.Context, ip, instanceID string) {
	logger := s.logger.WithFields(log.Fields{
		"func": "detectController",
	})

	cmds := []string{
		configs.PythonCMD,
		configs.AllocScript,
		"--accessKey=" + configs.AccessKey,
		"--secretKey=" + configs.SecretKey,
		"--region=" + configs.Region,
		"--action=" + configs.StatusAction,
		"--instanceID=" + instanceID,
	}
	logger.Infof("CMD: %v.", cmds)

	var resp AllocMachineResponse
	output, err := common.ExecCmd(ctx, cmds)

	if err != nil {
		logger.Errorf("Get ecs status error with %v. Output: %s\n", err, output)
		return
	}

	output = strings.Replace(output, " ", "", -1)
	output = strings.Replace(output, "\n", "", -1)
	output = strings.Replace(output, "\t", "", -1)
	logger.Debugf("Get ecs status OK: %s\n", output)
	if err = json.Unmarshal([]byte(output), &resp); err != nil {
		logger.Errorf("Json Unmarshal failed with %v\n", err)
		return
	}

	logger.Debugf("instance: %s expired time: %s, lock reason\n", instanceID,
		resp.ExpiredTime, resp.LockReason)

	isExpired := false

	if ok, _ := rfc3339Expired(resp.ExpiredTime); ok == true {
		isExpired = true
	}

	if resp.LockReason == "Recycling" {
		isExpired = true
	}

	if isExpired == false {
		logger.Debugf("instance: %s is still running, skipped.\n", instanceID)
		return
	}

	logger.Warnf("instance: %s has been expired, begin to update machine status.\n", instanceID)
	if err := s.updateMachineStatus(instanceID, configs.MachineExpired); err != nil {
		logger.Warnf("Failed to update machine status, due to %v\n", err)
		return
	}

	logger.Debug("update machine status [machine-expired] OK.\n")
	return
}

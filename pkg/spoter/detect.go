package spoter

import (
    "context"
    "time"

    log "github.com/sirupsen/logrus"

    "github.com/willstudy/spoter/pkg/configs"
    "github.com/willstudy/spoter/pkg/common"
)


func checkExpired(termination string) bool {
    // TODO:
    return false
}

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
        "ssh",
        "-o",
        "StrictHostKeyChecking=no",
        ip,
        configs.DetectAction,
    }
    logger.Debugf("CMD: %v\n", cmds)

    expired := false
    retry := 3
    for i := 0; i < retry; i++ {
		output, err := common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v. Output: %s.", i, err, output)
		} else {
			logger.Debugf("get instance termination-time OK, output: %s.\n", output)
            if checkExpired(output) == true {
                expired = true
            }
			break
		}
	}

    if expired == false {
        return
    }

    if err := s.updateMachineStatus(instanceID, configs.MachineExpired); err != nil {
		logger.Warnf("Failed to update machine status, due to %v\n", err)
		return
	}

	logger.Debug("update machine status [machine-running] OK.\n")
    return
}

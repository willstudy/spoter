package spoter

import (
	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/configs"
)

func (s *spoterController) restoreFromDB() {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreFromDB",
	})

	logger.Info("Begin to restore from db.")
	for instanceId, machineInfo := range s.k8sMachine {

		if machineInfo.Status == configs.MachineCreated {
			logger.Debugf("Instance: %s, status: %s, continue from <install k8s-base>.", instanceId, machineInfo.Status)
			// TODO:
			// install k8s base
			// join into k8s
			// label this node
		}

		if machineInfo.Status == configs.MachineInstalled {
			logger.Debugf("Instance: %s, status: %s, continue from <join into k8s>.", instanceId, machineInfo.Status)
			// TODO:
			// join into k8s
			// label this node
		}

		if machineInfo.Status == configs.MachineJoined {
			logger.Debugf("Instance: %s, status: %s, continue from <label this node>.", instanceId, machineInfo.Status)
			// TODO:
			// label this node
		}

		if machineInfo.Status == configs.MachineRunning {
			logger.Debugf("Instance: %s has running, skipped.", instanceId)
			continue
		}

		if machineInfo.Status == configs.MachineDestory {
			logger.Debugf("Instance: %s, status: %s, continue from <destory this node>.", instanceId, machineInfo.Status)
			// TODO:
			// remove this node
		}
	}

}

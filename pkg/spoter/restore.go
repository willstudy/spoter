package spoter

import (
	"fmt"

	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/configs"
)

func (s *spoterController) restoreAction(restoreIndex int32, m K8sMachine) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreAction",
	})

	if restoreIndex < configs.RESTORE_ACTION_FROM_MACHINE_CREATED ||
		restoreIndex > configs.RESTORE_ACTION_FROM_MACHINE_JOINED {
		return fmt.Errorf("Incorrect restore action index: %v", restoreIndex)
	}

	if restoreIndex <= configs.RESTORE_ACTION_FROM_MACHINE_CREATED {
		if err := s.installK8sBase(m.PrivateIP, m.Hostname); err != nil {
			logger.Errorf("Failed to install k8s base, due to: %v", err)
			return fmt.Errorf("Failed to install k8s base, due to: %v", err)
		}
	}

	if restoreIndex <= configs.RESTORE_ACTION_FROM_MACHINE_INSTALLED {
		kubeToken, err := s.getKubeToken()
		if err != nil {
			logger.Errorf("Failed to get kubeToken, due to: %v", err)
			return fmt.Errorf("Failed to get kubeToken, due to: %v", err)
		}

		if err = s.joinIntoK8s(m.PrivateIP, kubeToken, m.Hostname); err != nil {
			logger.Errorf("Failed to get join into k8s, due to: %v", err)
			return fmt.Errorf("Failed to get join into k8s, due to: %v", err)
		}
	}

	if restoreIndex <= configs.RESTORE_ACTION_FROM_MACHINE_JOINED {
		s.waitNodeReady(m.Hostname)
		if err := s.labelNode(m.Hostname, m.InstanceType); err != nil {
			logger.Errorf("Failed to label node, due to: %v", err)
			return fmt.Errorf("Failed to label node, due to: %v", err)
		}
	}

	logger.Info("Restore OK.")
	return nil
}

func (s *spoterController) restoreFromDB() {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreFromDB",
	})

	logger.Info("Begin to restore from db.")
	for instanceId, machineInfo := range s.k8sMachine {

		if machineInfo.Status == configs.MachineCreated {
			logger.Debugf("Instance: %s, status: %s, continue from <install k8s-base>.\n", instanceId, machineInfo.Status)
			// install k8s base
			// join into k8s
			// label this node
			if err := s.restoreAction(configs.RESTORE_ACTION_FROM_MACHINE_CREATED, machineInfo); err != nil {
				logger.Warnf("Failed to restore instance, due to: %v\n", err)
			}
			continue
		}

		if machineInfo.Status == configs.MachineInstalled {
			logger.Debugf("Instance: %s, status: %s, continue from <join into k8s>.", instanceId, machineInfo.Status)
			// join into k8s
			// label this node
			if err := s.restoreAction(configs.RESTORE_ACTION_FROM_MACHINE_INSTALLED, machineInfo); err != nil {
				logger.Warnf("Failed to restore instance, due to: %v\n", err)
			}
			continue
		}

		if machineInfo.Status == configs.MachineJoined {
			logger.Debugf("Instance: %s, status: %s, continue from <label this node>.", instanceId, machineInfo.Status)
			// label this node
			if err := s.restoreAction(configs.RESTORE_ACTION_FROM_MACHINE_JOINED, machineInfo); err != nil {
				logger.Warnf("Failed to restore instance, due to: %v\n", err)
			}
			continue
		}

		if machineInfo.Status == configs.MachineRunning {
			logger.Debugf("Instance: %s has running, skipped.", instanceId)
			continue
		}

		if machineInfo.Status == configs.MachineExpired {
			logger.Debugf("Instance: %s, status: %s, continue from <this node expired>.", instanceId, machineInfo.Status)
			// TODO:
			// remove this node
			// delete ecs
		}

		if machineInfo.Status == configs.MachineRemoved {
			logger.Debugf("Instance: %s, status: %s, continue from <remove this node>.", instanceId, machineInfo.Status)
			// TODO:
			// delete ecs
		}

		if machineInfo.Status == configs.MachineDeleted {
			logger.Debugf("Instance: %s has deleted, skipped.", instanceId)
			continue
		}
		break
	}
}

package spoter

import (
	"errors"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/configs"
)

type Step func(machineInfo *K8sMachine) (Step, error)

func restoreBeginWithInstallK8sBase(s *spoterController) Step {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreBeginWithInstallK8sBase",
	})
	// install k8s base
	// join into k8s
	// label this node
	return func(machineInfo *K8sMachine) (Step, error) {
		logger.Debugf("Instance: %s, status: %s, continue from <install k8s-base>.", machineInfo.InstanceID, machineInfo.Status)
		if err := s.installK8sBase(machineInfo.PrivateIP, machineInfo.Hostname); err != nil {
			logger.Errorf("Failed to install k8s base, due to: %v", err)
			return nil, err
		}
		machineInfo.Status = configs.MachineInstalled
		return restoreBeginWithJoinIntoK8s(s), nil
	}
}

func restoreBeginWithJoinIntoK8s(s *spoterController) Step {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreBeginWithJoinIntoK8s",
	})
	// join into k8s
	// label this node
	return func(machineInfo *K8sMachine) (Step, error) {
		logger.Debugf("Instance: %s, status: %s, continue from <join into k8s>.", machineInfo.InstanceID, machineInfo.Status)
		kubeToken, err := s.getKubeToken()
		if err != nil {
			return nil, err
		}
		err = s.joinIntoK8s(machineInfo.PrivateIP, kubeToken, machineInfo.Hostname)
		if err != nil {
			logger.Errorf("Failed to get join into k8s, due to: %v", err)
			return nil, err
		}
		machineInfo.Status = configs.MachineJoined
		return restoreBeginWithLabelNode(s), nil
	}
}

func restoreBeginWithLabelNode(s *spoterController) Step {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreBeginWithLabelNode",
	})
	// label this node
	return func(machineInfo *K8sMachine) (Step, error) {
		logger.Debugf("Instance: %s, status: %s, continue from <label this node>.", machineInfo.InstanceID, machineInfo.Status)
		if err := s.labelNode(machineInfo.Hostname, machineInfo.InstanceType); err != nil {
			logger.Errorf("Failed to label node, due to: %v", err)
			return nil, err
		}
		machineInfo.Status = configs.MachineRunning
		return nil, nil
	}
}

func restoreBeginWithRemoveNodeFromK8s(s *spoterController) Step {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreBeginWithRemoveNodeFromK8s",
	})
	// remove this node
	// delete ecs
	return func(machineInfo *K8sMachine) (Step, error) {
		logger.Debugf("Instance: %s, status: %s, continue from <this node expired>.", machineInfo.InstanceID, machineInfo.Status)
		if err := s.removeNodeFromK8s(machineInfo.InstanceID); err != nil {
			logger.Errorf("Failed to remove node label, due to: %v", err)
			return nil, err
		}
		machineInfo.Status = configs.MachineRemoved
		return restoreBeginWithDeleteECS(s), nil
	}
}

func restoreBeginWithDeleteECS(s *spoterController) Step {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreBeginWithDeleteECS",
	})
	// delete ecs
	return func(machineInfo *K8sMachine) (Step, error) {
		logger.Debugf("Instance: %s, status: %s, continue from <remove this node>.", machineInfo.InstanceID, machineInfo.Status)
		if err := s.deleteECS(machineInfo.InstanceID); err != nil {
			logger.Errorf("Failed to delete node, due to: %v", err)
			return nil, err
		}
		machineInfo.Status = configs.MachineDeleted
		return nil, nil
	}
}

func (s *spoterController) restoreFromDB() error {
	logger := s.logger.WithFields(log.Fields{
		"func": "restoreFromDB",
	})

	restoreWithStatus := map[string]Step{
		configs.MachineCreated:   restoreBeginWithInstallK8sBase(s),
		configs.MachineInstalled: restoreBeginWithJoinIntoK8s(s),
		configs.MachineJoined:    restoreBeginWithLabelNode(s),
		configs.MachineRunning:   nil,
		configs.MachineExpired:   restoreBeginWithRemoveNodeFromK8s(s),
		configs.MachineRemoved:   restoreBeginWithDeleteECS(s),
		configs.MachineDeleted:   nil,
	}

	logger.Info("Begin to restore from db.")
	for instanceId, machineInfo := range s.k8sMachine {

		if machineInfo.Status == configs.MachineRunning {
			logger.Debugf("Instance: %s has running, skipped.", instanceId)
			continue
		}

		if machineInfo.Status == configs.MachineDeleted {
			logger.Debugf("Instance: %s has deleted, skipped.", instanceId)
			continue
		}

		var (
			next Step
			ok   bool
			err  error
		)

		next, ok = restoreWithStatus[machineInfo.Status]
		if !ok {
			logger.Error("Instance: %s has invaild status %s.", instanceId, machineInfo.Status)
			return errors.New("invaild machine status")
		}

		for next != nil {
			next, err = next(&machineInfo)
			if err != nil {
				logger.Error("Instance: %s has restore failed due to %v.", instanceId, err)
				return err
			}
		}
		time.Sleep(30 * time.Second)
		// reload from DB
		s.loadMachineInfo()
	}
	return nil
}

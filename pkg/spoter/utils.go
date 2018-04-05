package spoter

import (
	"context"

	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/common"
	"github.com/willstudy/spoter/pkg/configs"
)

func (s *spoterController) allocMachine(label string, price float64) (string, string, error) {
	return "", "", nil
}

func (s *spoterController) installK8sBase(hostIp string) error {
	return nil
}

func (s *spoterController) joinNode(label string, price float64) {
	logger := s.logger.WithFields(log.Fields{
		"func": "rebalance",
	})
	hostIp, hostName, err := s.allocMachine(label, price)
	if err != nil {
		logger.Errorf("Failed to alloc Machine, due to: %v", err)
		return
	}
	if err := s.installK8sBase(hostIp); err != nil {
		logger.Errorf("Failed to install k8s base, due to: %v", err)
		return
	}

	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubeadmCMD,
		"join",
		"--token",
		configs.KubeToken,
		configs.KubeMaster,
		"--discovery-token-ca-cert-hash",
		configs.DiscoveryToken,
	}
	logger.Infof("CMD: %v.", cmds)
	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		_, err := common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v.", i, err)
		} else {
			logger.Debugf("Join new node OK.")
			break
		}
	}

	cmds = []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubectlCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"label",
		"no",
		hostName,
		configs.AliyunECSLabel + "=" + label,
	}
	logger.Infof("CMD: %v.", cmds)
	for i := 0; i < retry; i++ {
		_, err := common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v.", i, err)
		} else {
			logger.Debugf("Label new node OK.")
			break
		}
	}
	return
}

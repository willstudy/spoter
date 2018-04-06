package spoter

import (
	"context"
	"encoding/json"

	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/common"
	"github.com/willstudy/spoter/pkg/configs"
)

func (s *spoterController) allocMachine(label string, price float64) (string, string, error) {
	logger := s.logger.WithFields(log.Fields{
		"func": "allocMachine",
	})
	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.PythonCMD,
		configs.AllocScript,
		"--accessKey=" + configs.AccessKey,
		"--secretKey=" + configs.SecretKey,
		"--region=" + configs.Region,
		"--imageID=" + configs.ImageID,
		"--instanceType=" + configs.InstanceType,
		"--groupID=" + configs.InstanceType,
		"--price=" + configs.SpotPriceLimit,
		"--keyName=" + configs.SSHKeyName,
	}
	logger.Infof("CMD: %v.", cmds)

	var resp AllocMachineResponse
	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		output, err := common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v.\n", i, err)
		} else {
			logger.Debugf("Alloc Machine OK, output: %s\n", output)
			if err := json.Unmarshal([]byte(output), &resp); err != nil {
				logger.Errorf("Json Unmarshal failed with %v", err)
				return "", "", err
			}
			break
		}
	}
	return resp.EipAddress, resp.Hostname, nil
}

func (s *spoterController) installK8sBase(hostIp string) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "installK8sBase",
	})
	cmds := []string{
		"/bin/bash",
		"-x",
		configs.InstallK8sScript,
		hostIp,
	}
	ctx := context.TODO()
	output, err := common.ExecCmd(ctx, cmds)
	logger.Debugf("install k8s base output: %v\n", output)
	return err
}

func (s *spoterController) getKubeToken() (string, error) {
	logger := s.logger.WithFields(log.Fields{
		"func": "getKubeToken",
	})
	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubeadmCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"token",
		"create",
	}
	logger.Infof("CMD: %v.", cmds)

	var output string
	var err error
	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		output, err = common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v.", i, err)
		} else {
			logger.Debugf("create kube token OK, token: %s\n", output)
			return output, nil
		}
	}
	return "", err
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
	kubeToken, err := s.getKubeToken()
	if err != nil {
		logger.Errorf("Failed to get kubeToken, due to: %v", err)
		return
	}

	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubeadmCMD,
		"join",
		"--token",
		kubeToken,
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

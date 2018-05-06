package spoter

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/willstudy/spoter/pkg/common"
	"github.com/willstudy/spoter/pkg/configs"
)

func (s *spoterController) allocMachine(label string, price float64,
	bandwidth int32) (string, string, error) {
	logger := s.logger.WithFields(log.Fields{
		"func": "allocMachine",
	})

	cmds := []string{
		configs.PythonCMD,
		configs.AllocScript,
		"--accessKey=" + configs.AccessKey,
		"--secretKey=" + configs.SecretKey,
		"--region=" + configs.Region,
		"--imageID=" + configs.ImageID,
		"--instanceType=" + configs.InstanceType,
		"--groupID=" + configs.SecurityGroupID,
		"--keyName=" + configs.SSHKeyName,
		"--price=" + strconv.FormatFloat(price, 'E', -1, 64),
		"--bandwidth=" + strconv.FormatInt(int64(bandwidth), 10),
		"--action=" + configs.CreateAction,
	}
	logger.Infof("CMD: %v.", cmds)

	var resp AllocMachineResponse
	ctx := context.TODO()
	output, err := common.ExecCmd(ctx, cmds)

	if err != nil {
		logger.Errorf("Alloc Machine error with %v. Output: %s\n", err, output)
		return "", "", err
	}

	output = strings.Replace(output, " ", "", -1)
	output = strings.Replace(output, "\n", "", -1)
	output = strings.Replace(output, "\t", "", -1)
	logger.Debugf("Alloc Machine OK, output: %s\n", output)
	if err = json.Unmarshal([]byte(output), &resp); err != nil {
		logger.Errorf("Json Unmarshal failed with %v", err)
		return "", "", err
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
	logger.Infof("CMD: %v.", cmds)
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
			logger.Warnf("Try %d time, error: %v. Output: %s.", i, err, output)
		} else {
			logger.Debugf("create kube token OK, token: %s.", output)
			return output, nil
		}
	}
	return "", err
}

func (s *spoterController) joinIntoK8s(hostIp, kubeToken string) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "joinIntoK8s",
	})

	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		"ssh",
		"-o",
		"StrictHostKeyChecking=no",
		"root@" + hostIp,
		configs.RemoteKubeadmCMD,
		"join",
		"--token",
		strings.Trim(kubeToken, "\n"),
		configs.KubeMaster,
		"--discovery-token-ca-cert-hash",
		configs.DiscoveryToken,
	}
	logger.Infof("CMD: %v.", cmds)

	var output string
	var err error
	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		output, err = common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, output: %s, error: %v.", i, output, err)
		} else {
			logger.Debugf("Join new node OK.")
			break
		}
	}
	return err
}

func (s *spoterController) waitNodeReady(hostName string) {
	logger := s.logger.WithFields(log.Fields{
		"func": "waitNodeReady",
	})
	retry := 5
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubectlCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"get",
		"no",
		strings.ToLower(hostName),
	}
	logger.Infof("CMD: %v.", cmds)

	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		output, err := common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, output: %s, error: %v.", i, output, err)
		} else {
			logger.Debugf("New node has been ready.")
			break
		}
		time.Sleep(5 * time.Second)
	}
}

func (s *spoterController) labelNode(hostName, label string) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "labelNode",
	})
	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubectlCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"label",
		"no",
		strings.ToLower(hostName),
		configs.AliyunECSLabel + "=" + label,
	}
	logger.Infof("CMD: %v.", cmds)

	var err error
	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		_, err = common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v.", i, err)
		} else {
			logger.Debugf("Label new node OK.")
			break
		}
	}
	return err
}
func (s *spoterController) joinNode(label string, price float64, bandwidth int32) {
	logger := s.logger.WithFields(log.Fields{
		"func": "rebalance",
	})

	hostIp, hostName, err := s.allocMachine(label, price, bandwidth)
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

	if err = s.joinIntoK8s(hostIp, kubeToken); err != nil {
		logger.Errorf("Failed to get join into k8s, due to: %v", err)
		return
	}

	s.waitNodeReady(hostName)

	if err = s.labelNode(hostName, label); err != nil {
		logger.Errorf("Failed to label node, due to: %v", err)
		return
	}

	logger.Info("Join a new node OK.")
	return
}

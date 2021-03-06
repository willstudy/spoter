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

func getK8SNodeName(ecsHostNmae string) string {
	return strings.ToLower(ecsHostNmae)
}

func abs(n int32) int32 {
	if n < 0 {
		return -n
	} else {
		return n
	}
}

func (s *spoterController) setK8sMachine(instanceID, status string) {
	s.lock.Lock()
	m := s.k8sMachine[instanceID]
	m.Status = status
	delete(s.k8sMachine, instanceID)
	s.k8sMachine[instanceID] = m
	s.lock.Unlock()
}

func (s *spoterController) updateMachineStatus(instanceID, status string) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "updateMachineStatus",
	})

	sql := "UPDATE machine_info set status = ? where instance_id = ?"
	logger.Debugf("sql : %s\n", sql)

	stmtIns, err := s.dbCon.Prepare(sql)
	if err != nil {
		logger.Fatal("Failed to prepare sql with: %v\n", err)
		return err
	}
	defer stmtIns.Close()

	if _, err := stmtIns.Exec(instanceID, status); err != nil {
		logger.Fatal("Failed to update machine info with: %v\n", err)
		return err
	}
	return nil
}

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
		"--instanceType=" + label,
		"--groupID=" + configs.SecurityGroupID,
		"--vSwitchID=" + configs.VSwitchID,
		"--keyName=" + configs.SSHKeyName,
		"--price=" + strconv.FormatFloat(price, 'g', -1, 64),
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
		logger.Errorf("Json Unmarshal failed with %v\n", err)
		return "", "", err
	}

	var m K8sMachine
	timestamp := time.Now().Unix()
	tm_str := time.Unix(timestamp, 0)

	m.Hostname = getK8SNodeName(resp.Hostname)
	m.Region = configs.Region
	m.ImageId = configs.ImageID
	m.InstanceType = label
	m.SpotWithPriceLimit = price
	m.BandWidth = bandwidth
	m.PublicIP = resp.EipAddress
	m.PrivateIP = resp.InnerAddress
	m.Status = configs.MachineCreated
	m.CreatedTime = tm_str.Format("2006-01-02 03:04:05 PM")

	sql := "INSERT INTO machine_info(hostname, region, image_id, instance_type,"
	sql += " spot_price_limit, bandwith, instance_id, public_ip, private_ip,"
	sql += " status, create_time) values(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)"

	logger.Debugf("sql : %s\n", sql)

	stmtIns, err := s.dbCon.Prepare(sql)
	if err != nil {
		logger.Fatal("Failed to prepare sql with: %v\n", err)
		return "", "", err
	}
	defer stmtIns.Close()

	if _, err := stmtIns.Exec(m.Hostname, m.Region, m.ImageId, m.InstanceType,
		m.SpotWithPriceLimit, m.BandWidth, resp.InstanceID, m.PublicIP, m.PrivateIP,
		m.Status, m.CreatedTime); err != nil {
		logger.Fatal("Failed to insert into mysql with: %v\n", err)
		return "", "", err
	}

	s.lock.Lock()
	delete(s.k8sMachine, resp.InstanceID)
	s.k8sMachine[resp.InstanceID] = m
	s.lock.Unlock()

	logger.Debug("update machine status [machine-created] OK.\n")
	return resp.InnerAddress, resp.InstanceID, nil
}

func (s *spoterController) installK8sBase(hostIp, instanceID string) error {
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

	if err != nil {
		logger.Warn("Failed install k8s base.\n")
		return err
	}

	if err = s.updateMachineStatus(instanceID, configs.MachineInstalled); err != nil {
		logger.Warnf("Failed to update machine status, due to %v\n", err)
		return err
	}

	s.setK8sMachine(instanceID, configs.MachineInstalled)

	logger.Debug("update machine status [machine-installed] OK.\n")
	return nil
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

func (s *spoterController) joinIntoK8s(hostIp, kubeToken, instanceID string) error {
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

	if err != nil {
		return err
	}

	if err = s.updateMachineStatus(instanceID, configs.MachineJoined); err != nil {
		logger.Warnf("Failed to update machine status, due to %v\n", err)
		return err
	}

	s.setK8sMachine(instanceID, configs.MachineJoined)

	logger.Debug("update machine status [machine-joined] OK.\n")
	return nil
}

func (s *spoterController) waitNodeReady(instanceID string) {
	logger := s.logger.WithFields(log.Fields{
		"func": "waitNodeReady",
	})

	retry := 5
	m := s.k8sMachine[instanceID]

	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubectlCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"get",
		"no",
		getK8SNodeName(m.Hostname),
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

func (s *spoterController) labelNode(instanceID, label string) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "labelNode",
	})

	retry := 3
	m := s.k8sMachine[instanceID]

	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubectlCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"label",
		"no",
		getK8SNodeName(m.Hostname),
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

	if err != nil {
		return err
	}

	if err = s.updateMachineStatus(instanceID, configs.MachineRunning); err != nil {
		logger.Warnf("Failed to update machine status, due to %v\n", err)
		return err
	}

	s.setK8sMachine(instanceID, configs.MachineRunning)

	logger.Debug("update machine status [machine-running] OK.\n")
	return nil
}
func (s *spoterController) joinNode(label string, price float64, bandwidth int32) {
	logger := s.logger.WithFields(log.Fields{
		"func": "rebalance",
	})

	hostIp, instanceID, err := s.allocMachine(label, price, bandwidth)
	if err != nil {
		logger.Errorf("Failed to alloc Machine, due to: %v", err)
		return
	}

	if err := s.installK8sBase(hostIp, instanceID); err != nil {
		logger.Errorf("Failed to install k8s base, due to: %v", err)
		return
	}

	kubeToken, err := s.getKubeToken()
	if err != nil {
		logger.Errorf("Failed to get kubeToken, due to: %v", err)
		return
	}

	if err = s.joinIntoK8s(hostIp, kubeToken, instanceID); err != nil {
		logger.Errorf("Failed to get join into k8s, due to: %v", err)
		return
	}

	s.waitNodeReady(instanceID)

	if err = s.labelNode(instanceID, label); err != nil {
		logger.Errorf("Failed to label node, due to: %v", err)
		return
	}

	logger.Info("Join a new node OK.")
	return
}

func (s *spoterController) removeNodeFromK8s(instanceID string) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "removeNodeFromK8s",
	})

	retry := 3
	cmds := []string{
		configs.TimeCMD,
		configs.TimeoutS,
		configs.KubectlCMD,
		"--kubeconfig=" + configs.KubeConfig,
		"delete",
		"no",
		getK8SNodeName(instanceID),
	}
	logger.Infof("CMD: %v.", cmds)

	var err error
	ctx := context.TODO()
	for i := 0; i < retry; i++ {
		_, err = common.ExecCmd(ctx, cmds)
		if err != nil {
			logger.Warnf("Try %d time, error: %v.", i, err)
		} else {
			logger.Debugf("remove node OK.")
			break
		}
	}

	if err != nil {
		return err
	}

	if err = s.updateMachineStatus(instanceID, configs.MachineRemoved); err != nil {
		logger.Warnf("Failed to update machine status, due to %v\n", err)
		return err
	}

	s.setK8sMachine(instanceID, configs.MachineRemoved)

	logger.Debug("update machine status [machine-removed] OK.\n")
	return nil
}

func (s *spoterController) deleteECS(instanceID string) error {
	logger := s.logger.WithFields(log.Fields{
		"func": "deleteECS",
	})

	cmds := []string{
		configs.PythonCMD,
		configs.AllocScript,
		"--accessKey=" + configs.AccessKey,
		"--secretKey=" + configs.SecretKey,
		"--region=" + configs.Region,
		"--action=" + configs.DeleteAction,
		"--instanceID=" + instanceID,
	}
	logger.Infof("CMD: %v.", cmds)

	ctx := context.TODO()
	output, err := common.ExecCmd(ctx, cmds)
	if err != nil {
		logger.Errorf("Delete ecs error with %v. Output: %s\n", err, output)
		return err
	}
	logger.Debugf("Delete ecs OK: %s\n", output)

	if err = s.updateMachineStatus(instanceID, configs.MachineDeleted); err != nil {
		logger.Warnf("Failed to update machine status, due to %v\n", err)
		return err
	}

	s.setK8sMachine(instanceID, configs.MachineDeleted)

	logger.Debug("update machine status [machine-deleted] OK.\n")
	return nil
}

func (s *spoterController) deleteNode(instanceID string) {
	logger := s.logger.WithFields(log.Fields{
		"func": "deleteNode",
	})

	if err := s.removeNodeFromK8s(instanceID); err != nil {
		logger.Errorf("Failed to remove node label, due to: %v", err)
		return
	}

	if err := s.deleteECS(instanceID); err != nil {
		logger.Errorf("Failed to delete node, due to: %v", err)
		return
	}
}

func rfc3339Expired(value string) (bool, error) {
	if value == "" {
		return false, nil
	}

	t, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return false, err
	}
	return t.Unix()-time.Now().Unix() <= configs.ExpireDuration, nil
}

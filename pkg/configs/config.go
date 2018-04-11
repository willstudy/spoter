package configs

const (
	KubeadmCMD       = "/home/spoter/k8s-base/kubeadm"
	RemoteKubeadmCMD = "/usr/bin/kubeadm"
	KubeMaster       = "139.59.169.189:6443"
	DiscoveryToken   = "sha256:203738e32ce7a9b85848fd49ae6dbb375949716a61413be605d7029cc4e4b700"
	//JoinCMD        = "kubeadm join --token ba3a9c.a8a982e69445c017 139.59.169.189:6443 --discovery-token-ca-cert-hash sha256:203738e32ce7a9b85848fd49ae6dbb375949716a61413be605d7029cc4e4b700"

	KubectlCMD = "/home/spoter/k8s-base/kubectl"
	KubeConfig = "/home/spoter/k8s-base/admin.conf"
	TimeCMD    = "/usr/bin/timeout"
	PythonCMD  = "/usr/bin/python"

	AllocScript      = "/home/spoter/k8s-base/scripts/alloc-machine.py"
	InstallK8sScript = "/home/spoter/k8s-base/scripts/install-k8s-base.sh"

	TimeoutS       = "120"
	AliyunECSLabel = "aliyun-ecs"
)

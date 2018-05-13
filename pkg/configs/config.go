package configs

const (
	KubeadmCMD       = "/home/spoter/k8s-base/kubeadm"
	RemoteKubeadmCMD = "/usr/bin/kubeadm"
	KubeMaster       = "172.31.250.28:6443"
	DiscoveryToken   = "sha256:b44288c64f48cff8280f0109b7329efff23d2c054475313216140633571283c2"
	//JoinCMD        = "kubeadm join --token ba3a9c.a8a982e69445c017 139.59.169.189:6443 --discovery-token-ca-cert-hash sha256:203738e32ce7a9b85848fd49ae6dbb375949716a61413be605d7029cc4e4b700"

	KubectlCMD = "/home/spoter/k8s-base/kubectl"
	KubeConfig = "/home/spoter/k8s-base/admin.conf"
	TimeCMD    = "/usr/bin/timeout"
	PythonCMD  = "/usr/bin/python"

	AllocScript      = "/home/spoter/k8s-base/scripts/alloc-machine.py"
	InstallK8sScript = "/home/spoter/k8s-base/scripts/install-k8s-base.sh"

	TimeoutS       = "120"
	AliyunECSLabel = "aliyun-ecs"

	CreateAction = "create"
	DeleteAction = "delete"

	SQLDSN = "root:banana@tcp(127.0.0.1:3306)/aliyun" // "user:password@/dbname"
)

const (
	MachineCreated = "created"
	MachineWorking = "working"
	MachineDestory = "Destory"
	MachineDeleted = "Deleted"
)

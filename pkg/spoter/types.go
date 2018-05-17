package spoter

type MachineInfo struct {
	Num       int32   `json:"num"`
	Price     float64 `json:"price"`
	BandWidth int32   `json:"bandwidth"`
}

type SpoterModel map[string]MachineInfo
type SpoterConfig struct {
	Model         SpoterModel `json:"model"`
	CheckInterval int32       `json:"checkInterval"`
}

type AllocMachineResponse struct {
	Code         int32  `json:"code"`
	Msg          string `json:"msg"`
	EipAddress   string `json:"EipAddress,omitempty"`
	InnerAddress string `json:"InnerAddress,omitempty"`
	Hostname     string `json:"Hostname,omitempty"`
	InstanceID   string `json:"InstanceID,omitempty"`
	ExpiredTime  string `json:"ExpiredTime,omitempty"`
	LockReason   string `json:"LockReason,omitempty"`
}

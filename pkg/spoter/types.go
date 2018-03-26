package spoter

type SpoterModel map[string]int32
type SpoterConfig struct {
	Model         SpoterModel `json:"model"`
	CheckInterval int32       `json:"checkInterval"`
}

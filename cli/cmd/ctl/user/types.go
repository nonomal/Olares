package user

type resourceLimit struct {
	memoryLimit string
	cpuLimit    string
}

type userInfo struct {
	UID               string   `json:"uid"`
	Name              string   `json:"name"`
	DisplayName       string   `json:"display_name"`
	Description       string   `json:"description"`
	Email             string   `json:"email"`
	State             string   `json:"state"`
	LastLoginTime     *int64   `json:"last_login_time"`
	CreationTimestamp int64    `json:"creation_timestamp"`
	Avatar            string   `json:"avatar"`
	TerminusName      string   `json:"terminusName"`
	WizardComplete    bool     `json:"wizard_complete"`
	Roles             []string `json:"roles"`
	MemoryLimit       string   `json:"memory_limit"`
	CpuLimit          string   `json:"cpu_limit"`
}

var (
	defaultMemoryLimit = "3G"
	defaultCPULimit    = "1"

	systemObjectName      = "terminus"
	systemObjectDomainKey = "domainName"
)

// Package scout implements remote server availability checking via SSH.
package scout

// ScoutConfig represents the top-level servers.yml structure.
type ScoutConfig struct {
	Servers []ServerEntry `yaml:"servers"`
	SSH     SSHConfig     `yaml:"ssh"`
}

// ServerEntry is a single entry in servers.yml (either name or pattern).
type ServerEntry struct {
	Name    string `yaml:"name,omitempty"`
	Pattern string `yaml:"pattern,omitempty"`
	HasGPU  bool   `yaml:"has_gpu,omitempty"`
}

// SSHConfig holds SSH connection parameters.
type SSHConfig struct {
	ProxyJump      string `yaml:"proxy_jump,omitempty"`
	ConnectTimeout int    `yaml:"connect_timeout,omitempty"` // seconds, default 10
}

// Server is an expanded, resolved server ready to check.
type Server struct {
	Name   string
	HasGPU bool
}

// ScoutResult is the top-level JSON output.
type ScoutResult struct {
	Servers []ServerStatus `json:"servers"`
}

// ServerStatus is one server's check result.
type ServerStatus struct {
	Name    string         `json:"name"`
	Status  string         `json:"status"` // "online" or "offline"
	Error   string         `json:"error,omitempty"`
	Metrics *ServerMetrics `json:"metrics,omitempty"`
}

// ServerMetrics holds parsed metric values.
type ServerMetrics struct {
	CPUPercent    float64   `json:"cpu_percent"`
	MemoryPercent float64   `json:"memory_percent"`
	LoadAvg1      float64   `json:"load_avg_1min"`
	LoadAvg5      float64   `json:"load_avg_5min"`
	LoadAvg15     float64   `json:"load_avg_15min"`
	GPUs          []GPUInfo `json:"gpus,omitempty"`
	TopUsers      []UserCPU `json:"top_users,omitempty"`
}

// GPUInfo holds per-GPU metrics.
type GPUInfo struct {
	UtilizationPercent int `json:"utilization_percent"`
	MemoryUsedMB       int `json:"memory_used_mb"`
	MemoryTotalMB      int `json:"memory_total_mb"`
}

// UserCPU holds aggregated CPU usage for a single user.
type UserCPU struct {
	User       string  `json:"user"`
	CPUPercent float64 `json:"cpu_percent"`
}

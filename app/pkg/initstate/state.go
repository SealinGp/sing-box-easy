package initstate

// InitStateManager defines the interface for managing initialization state
type InitStateManager interface {
	Init() error
	GetState() *State
	SetSingBoxInstalled(version string) error
	SetDashboardInstalled() error
	CompleteInitialization() error
	Reset() error
}

const (
	DefaultStatePath = "/etc/sing-box/init_state.json"
)

// State represents the initialization state
type State struct {
	Initialized        bool   `json:"initialized"`
	SingBoxInstalled   bool   `json:"sing_box_installed"`
	ConfigGenerated    bool   `json:"config_generated"`
	DashboardInstalled bool   `json:"dashboard_installed"`
	SingBoxVersion     string `json:"sing_box_version"`
	InitTime           string `json:"init_time,omitempty"`
	StatePath          string `json:"state_path,omitempty"`
}

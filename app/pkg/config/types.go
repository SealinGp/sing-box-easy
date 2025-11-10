package config

// SingBoxConfig represents the complete sing-box configuration
type SingBoxConfig struct {
	Log          *LogConfig          `json:"log,omitempty"`
	Experimental *ExperimentalConfig `json:"experimental,omitempty"`
	DNS          *DNSConfig          `json:"dns,omitempty"`
	Inbounds     []Inbound           `json:"inbounds,omitempty"`
	Outbounds    []Outbound          `json:"outbounds,omitempty"`
	Route        *RouteConfig        `json:"route,omitempty"`
}

// LogConfig represents log configuration
type LogConfig struct {
	Disabled  bool   `json:"disabled"`
	Level     string `json:"level"`
	Timestamp bool   `json:"timestamp"`
}

// ExperimentalConfig represents experimental features configuration
type ExperimentalConfig struct {
	ClashAPI  *ClashAPIConfig  `json:"clash_api,omitempty"`
	CacheFile *CacheFileConfig `json:"cache_file,omitempty"`
}

// ClashAPIConfig represents Clash API configuration
type ClashAPIConfig struct {
	ExternalController     string `json:"external_controller"`
	ExternalUI             string `json:"external_ui"`
	Secret                 string `json:"secret"`
	ExternalUIDownloadURL  string `json:"external_ui_download_url"`
	ExternalUIDownloadDetour string `json:"external_ui_download_detour"`
	DefaultMode            string `json:"default_mode"`
}

// CacheFileConfig represents cache file configuration
type CacheFileConfig struct {
	Enabled     bool   `json:"enabled"`
	Path        string `json:"path"`
	StoreFakeIP bool   `json:"store_fakeip"`
}

// DNSConfig represents DNS configuration
type DNSConfig struct {
	Servers  []DNSServer `json:"servers,omitempty"`
	Rules    []DNSRule   `json:"rules,omitempty"`
	Final    string      `json:"final,omitempty"`
	Strategy string      `json:"strategy,omitempty"`
}

// DNSServer represents a DNS server configuration
type DNSServer struct {
	Tag             string              `json:"tag"`
	Type            string              `json:"type,omitempty"`
	Address         string              `json:"address,omitempty"`
	AddressStrategy string              `json:"address_strategy,omitempty"`
	Strategy        string              `json:"strategy,omitempty"`
	Detour          string              `json:"detour,omitempty"`
	Predefined      map[string][]string `json:"predefined,omitempty"` // For hosts type
}

// DNSRule represents a DNS rule
type DNSRule struct {
	Server       string      `json:"server,omitempty"`
	Outbound     string      `json:"outbound,omitempty"`
	ClashMode    string      `json:"clash_mode,omitempty"`
	RuleSet      interface{} `json:"rule_set,omitempty"` // Can be string or []string
	Action       string      `json:"action,omitempty"`
	DisableCache bool        `json:"disable_cache,omitempty"`
	IPAcceptAny  bool        `json:"ip_accept_any,omitempty"`
}

// Inbound represents an inbound configuration
type Inbound struct {
	Tag          string   `json:"tag"`
	Type         string   `json:"type"`
	Address      []string `json:"address,omitempty"`
	MTU          int      `json:"mtu,omitempty"`
	AutoRoute    bool     `json:"auto_route,omitempty"`
	AutoRedirect bool     `json:"auto_redirect,omitempty"`
	Listen       string   `json:"listen,omitempty"`
	ListenPort   int      `json:"listen_port,omitempty"`
}

// Outbound represents an outbound configuration
type Outbound struct {
	Tag        string      `json:"tag"`
	Type       string      `json:"type"`
	Server     string      `json:"server,omitempty"`
	ServerPort int         `json:"server_port,omitempty"`
	Method     string      `json:"method,omitempty"`
	Password   string      `json:"password,omitempty"`
	UUID       string      `json:"uuid,omitempty"`
	AlterID    int         `json:"alter_id,omitempty"`
	Outbounds  []string    `json:"outbounds,omitempty"` // For selector/urltest
	URL        string      `json:"url,omitempty"`       // For urltest
	Interval   string      `json:"interval,omitempty"`  // For urltest
	Tolerance  int         `json:"tolerance,omitempty"` // For urltest
	Detour     string      `json:"detour,omitempty"`    // For chain proxy
}

// RouteConfig represents route configuration
type RouteConfig struct {
	AutoDetectInterface bool        `json:"auto_detect_interface"`
	Final               string      `json:"final"`
	Rules               []RouteRule `json:"rules,omitempty"`
	RuleSet             []RuleSet   `json:"rule_set,omitempty"`
}

// RouteRule represents a routing rule
type RouteRule struct {
	IPCidr         []string    `json:"ip_cidr,omitempty"`
	Outbound       string      `json:"outbound,omitempty"`
	Action         string      `json:"action,omitempty"`
	Protocol       string      `json:"protocol,omitempty"`
	ClashMode      string      `json:"clash_mode,omitempty"`
	Domain         []string    `json:"domain,omitempty"`
	DomainSuffix   []string    `json:"domain_suffix,omitempty"`
	RuleSet        interface{} `json:"rule_set,omitempty"` // Can be string or []string
}

// RuleSet represents a rule set configuration
type RuleSet struct {
	Tag            string `json:"tag"`
	Type           string `json:"type"`
	Format         string `json:"format"`
	URL            string `json:"url"`
	DownloadDetour string `json:"download_detour"`
}

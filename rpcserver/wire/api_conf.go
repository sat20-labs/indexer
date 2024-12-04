package wire

type RPCService struct {
	Addr    string  `yaml:"addr"`
	Proxy   string  `yaml:"proxy"`
	LogPath string  `yaml:"log_path"`
	Swagger Swagger `yaml:"swagger"`
	API     API     `yaml:"api"`
}

type Swagger struct {
	Host    string   `yaml:"host"`
	Schemes []string `yaml:"schemes"`
}

type API struct {
	APIKeyList      map[string]*APIKey `yaml:"apikey_list"`
	NoLimitApiList  []string           `yaml:"nolimit_api_list"`
	NoLimitHostList []string           `yaml:"nolimit_host_list"`
}

type APIKey struct {
	UserName  string     `yaml:"user_name"`
	RateLimit *RateLimit `yaml:"rate_limit"`
}

type RateLimit struct {
	PerSecond int `yaml:"per_second"`
	PerDay    int `yaml:"per_day"`
	Max       int `yaml:"max"`
	Burst     int `yaml:"burst"`
}

type APIInfo struct {
	RateLimit *RateLimit `yaml:"rate_limit"`
}

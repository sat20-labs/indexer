package conf

type YamlConf struct {
	Chain      string     `yaml:"chain"`
	DB         DB         `yaml:"db"`
	ShareRPC   ShareRPC   `yaml:"share_rpc"`
	Log        Log        `yaml:"log"`
	BasicIndex BasicIndex `yaml:"basic_index"`
	RPCService any        `yaml:"rpc_service"`
}

type DB struct {
	Path string `yaml:"path"`
}

type ShareRPC struct {
	Bitcoin Bitcoin `yaml:"bitcoin"`
}

type Bitcoin struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`
	User     string `yaml:"user"`
	Password string `yaml:"password"`
}

type Log struct {
	Level string `yaml:"level"`
	Path  string `yaml:"path"`
}

type BasicIndex struct {
	MaxIndexHeight  int64 `yaml:"max_index_height"`
	PeriodFlushToDB int   `yaml:"period_flush_to_db"`
}

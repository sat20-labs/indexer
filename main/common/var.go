package define

import (
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/main/conf"
)

var (
	YamlCfg *conf.YamlConf
)

func GetChain() string {
	chain := ""
	if YamlCfg != nil {
		chain = YamlCfg.Chain
	} else {
		chain = common.ChainMainnet
	}

	switch chain {
	case common.ChainTestnet:
		return common.ChainTestnet
	case common.ChainTestnet4:
		return common.ChainTestnet4
	case common.ChainMainnet:
		return common.ChainMainnet
	default:
		return common.ChainTestnet4
	}
}

package dkvs

import (
	"time"

	"github.com/libp2p/go-libp2p/core/peer"
)


type IdentityConfig struct {
	PrivKey      string
}

type NodeMode int32

const (
	ServiceMode NodeMode = iota
	LightMode
)

const (
	LightPort   = "0"
	ServicePort = "9000"
)

// NetworkConfig controls listen and annouce settings for the libp2p host.
type NetworkConfig struct {
	IsLocalNet              bool
	ListenAddrs             []string
	AnnounceAddrs           []string
	Libp2pForceReachability string
	Peers                   []peer.AddrInfo
	EnableMdns              bool
}

type DkvsConfig struct {
	Mode      NodeMode
	Network   NetworkConfig
	Bootstrap BootstrapConfig
	DHT       DHTConfig
	Identity  IdentityConfig
}


func NewDefaultDkvsConfig() *DkvsConfig {
	ret := DkvsConfig{

		Bootstrap: BootstrapConfig{
			BootstrapPeers: []string{
				// TODO add 4 dns peer bootstrap
				// "/dnsaddr/bootstrap.tinyverse.space/p2p/xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
				"/ip4/103.103.245.177/tcp/9000/p2p/12D3KooWFvycqvSRcrPPSEygV7pU6Vd2BrpGsMMFvzeKURbGtMva",
				// TODO add udp peer bootstrap
				// "/ip4/103.103.245.177/udp/9000/quic/p2p/12D3KooWFvycqvSRcrPPSEygV7pU6Vd2BrpGsMMFvzeKURbGtMva",
				"/ip4/156.251.179.141/tcp/9000/p2p/12D3KooWH743TTDbp2RLsLL2t2vVNdtKpm3AMyZffRVx5psBbbZ3",
				// TODO add udp peer bootstrap
				// "/ip4/156.251.179.141/udp/9000/quic/p2p/12D3KooWH743TTDbp2RLsLL2t2vVNdtKpm3AMyZffRVx5psBbbZ3",
				"/ip4/39.108.104.19/tcp/9000/p2p/12D3KooWNfbV19fQ9d39K84fUeFRmc6i4koEVNio9L6fPFtyPC9V",
				// TODO add udp peer bootstrap
				// "/ip4/39.108.104.19/udp/9000/quic/p2p/12D3KooWH743TTDbp2RLsLL2t2vVNdtKpm3AMyZffRVx5psBbbZ3",
			},
		},
		DHT: DHTConfig{
			DatastorePath:  "dht_data",
			ProtocolPrefix: "/sat20",
			ProtocolID:     "/sat20/1.0.0",
			MaxRecordAge:   time.Hour * 24 * 365 * 100,
		},
	}
	return &ret
}


func (cfg *DkvsConfig) InitMode(mode NodeMode) {
	cfg.Mode = mode
	switch cfg.Mode {
	case ServiceMode:
		cfg.Network.ListenAddrs = []string{
			"/ip4/0.0.0.0/udp/" + ServicePort + "/quic-v1",
			"/ip4/0.0.0.0/udp/" + ServicePort + "/quic-v1/webtransport",
			"/ip6/::/udp/" + ServicePort + "/quic-v1",
			"/ip6/::/udp/" + ServicePort + "/quic-v1/webtransport",
			"/ip4/0.0.0.0/tcp/" + ServicePort,
			"/ip6/::/tcp/" + ServicePort,
		}
	case LightMode:
		cfg.Network.ListenAddrs = []string{
			"/ip4/0.0.0.0/udp/" + LightPort + "/quic-v1",
			"/ip4/0.0.0.0/udp/" + LightPort + "/quic-v1/webtransport",
			"/ip6/::/udp/" + LightPort + "/quic-v1",
			"/ip6/::/udp/" + LightPort + "/quic-v1/webtransport",
			"/ip4/0.0.0.0/tcp/" + LightPort,
			"/ip6/::/tcp/" + LightPort,
		}
	}
}


// Package coloring implements the deterministic ARC20 transfer rules shared by
// the indexer and transaction planners. It has no database or indexer dependency.
package coloring

import (
	"fmt"

	"github.com/btcsuite/btcd/wire"
)

const (
	ActivationMainnet     = 808080
	DmintMainnet          = 819181
	CommitzMainnet        = 822800
	DensityMainnet        = 828128
	RolloverMainnet       = 828628
	CustomColoringMainnet = 848484
	ActivationTestnet4    = 27000
)

type ActivationHeights struct {
	Activation     int
	Dmint          int
	Commitz        int
	Density        int
	Rollover       int
	CustomColoring int
}

// HeightsForNetwork preserves the indexer's mainnet and testnet4 rules. Regtest
// uses its existing local test policy (27000), not mainnet's activation heights.
// Networks without a configured ARC20 policy must not silently use testnet4.
func HeightsForNetwork(network wire.BitcoinNet) (ActivationHeights, error) {
	switch network {
	case wire.MainNet:
		return ActivationHeights{
			Activation: ActivationMainnet, Dmint: DmintMainnet,
			Commitz: CommitzMainnet, Density: DensityMainnet,
			Rollover: RolloverMainnet, CustomColoring: CustomColoringMainnet,
		}, nil
	case wire.TestNet4, wire.TestNet:
		return ActivationHeights{
			Activation: ActivationTestnet4, Dmint: ActivationTestnet4,
			Commitz: ActivationTestnet4, Density: ActivationTestnet4,
			Rollover: ActivationTestnet4, CustomColoring: ActivationTestnet4,
		}, nil
	default:
		return ActivationHeights{}, fmt.Errorf("ARC20 activation is not configured for network %x", uint32(network))
	}
}

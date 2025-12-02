package ord0_14_1

type Epoch struct {
	value uint32
}

// FIRST_POST_SUBSIDY is the first epoch where the subsidy becomes 0.
var FIRST_POST_SUBSIDY = Epoch{value: 33}

func (e Epoch) Subsidy() uint64 {
	// Check if the current epoch is before the first post-subsidy epoch (Epoch 33)
	if e.value < FIRST_POST_SUBSIDY.value {
		// Calculate the initial reward (50 coins)
		initialReward := uint64(50) * COIN_VALUE

		// Epoch 0: initialReward >> 0
		// Epoch 1: initialReward >> 1 (half of initial)
		// Epoch 2: initialReward >> 2 (quarter of initial, or half of epoch 1)
		return initialReward >> e.value
	} else {
		// After the last halving (Epoch 33 onwards), the subsidy is 0.
		return 0
	}
}

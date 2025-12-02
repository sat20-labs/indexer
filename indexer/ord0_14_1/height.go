package ord0_14_1

// Height represents the block height.
type Height struct {
	Value uint32
}

// N returns the height value.
func (h Height) N() uint32 {
	return h.Value
}

// ToEpoch converts the current Height to its corresponding Epoch.
func (h Height) ToEpoch() Epoch {
	// The original Rust code:
	// Self(height.0 / SUBSIDY_HALVING_INTERVAL)
	return Epoch{value: h.Value / SUBSIDY_HALVING_INTERVAL}
}

// Subsidy returns the block reward for the current Height.
func (h Height) Subsidy() uint64 {
	// Convert the height to the epoch and calculate the subsidy.
	return h.ToEpoch().Subsidy()
}

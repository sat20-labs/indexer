package utils

// DirectionString is a helper function that returns a string that represents
// the direction of a connection (inbound or outbound).
func DirectionString(inbound bool) string {
	if inbound {
		return "inbound"
	}
	return "outbound"
}

// pickNoun returns the singular or plural form of a noun depending
// on the count n.
func PickNoun(n uint64, singular, plural string) string {
	if n == 1 {
		return singular
	}
	return plural
}

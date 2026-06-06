package bitcoin_rpc

import (
	"os"
	"strconv"
	"strings"
)

func debugBlockCount() (uint64, bool, error) {
	path := os.Getenv("ATOM_DEBUG_HEIGHT_FILE")
	if path == "" {
		return 0, false, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return 0, true, err
	}
	height, err := strconv.ParseUint(strings.TrimSpace(string(raw)), 10, 64)
	if err != nil {
		return 0, true, err
	}
	return height, true, nil
}

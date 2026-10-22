package coloring

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
)

type Asset struct {
	AtomicalID string
	Amount     int64
}

// Input is one previous output, in transaction input order. Assets must contain
// its complete ARC20 state; an asset-free input has an empty Assets slice. The
// caller remains responsible for authenticating these facts and other protocols.
type Input struct {
	Value  int64
	Assets []Asset
}

type Assignment struct {
	OutputIndex int
	Amount      int64
}

type Result struct {
	Order       []string
	Assignments map[string][]Assignment
	Burned      map[string]int64
	Fallback    bool
}

// AtomicalIDKey decodes the native location bytes used for ARC20 ordering. Human
// ticker names and non-canonical compact IDs cannot be used as asset identities.
func AtomicalIDKey(id string) ([]byte, error) {
	if len(id) < 66 || id[64] != 'i' || id[:64] != strings.ToLower(id[:64]) {
		return nil, fmt.Errorf("invalid ARC20 atomical ID %q", id)
	}
	hash, err := hex.DecodeString(id[:64])
	if err != nil || len(hash) != 32 {
		return nil, fmt.Errorf("invalid ARC20 atomical ID %q", id)
	}
	index, err := strconv.ParseUint(id[65:], 10, 32)
	if err != nil || strconv.FormatUint(index, 10) != id[65:] {
		return nil, fmt.Errorf("invalid ARC20 atomical ID %q", id)
	}
	key := make([]byte, 36)
	for i := range hash {
		key[i] = hash[31-i]
	}
	binary.LittleEndian.PutUint32(key[32:], uint32(index))
	return key, nil
}

// CompareAtomicalIDs preserves deterministic ordering for internal historical
// fixtures too. The wallet-facing entry points separately reject invalid IDs.
func CompareAtomicalIDs(a, b string) int {
	aKey, aErr := AtomicalIDKey(a)
	bKey, bErr := AtomicalIDKey(b)
	if aErr == nil && bErr == nil {
		return bytes.Compare(aKey, bKey)
	}
	if (aErr == nil) != (bErr == nil) {
		if aErr == nil {
			return -1
		}
		return 1
	}
	return strings.Compare(a, b)
}

func IsUnspendable(output *wire.TxOut) bool {
	if output == nil || len(output.PkScript) == 0 {
		return false
	}
	script := output.PkScript
	return script[0] == txscript.OP_RETURN ||
		(len(script) >= 2 && script[0] == txscript.OP_FALSE && script[1] == txscript.OP_RETURN)
}

// AssignRegular is the indexer's existing single-asset allocation rule. Before
// partial coloring is active an oversized output stops allocation. Afterwards
// the final output may hold fewer token units than its Bitcoin value.
func AssignRegular(outputs []*wire.TxOut, start int, amount int64, partial bool) ([]Assignment, bool) {
	result := make([]Assignment, 0)
	remaining := amount
	if start < 0 || amount < 0 {
		return result, false
	}
	for i := start; i < len(outputs) && remaining > 0; i++ {
		output := outputs[i]
		if output == nil || IsUnspendable(output) || output.Value <= 0 {
			continue
		}
		if !partial && output.Value > remaining {
			return result, false
		}
		assigned := output.Value
		if assigned > remaining {
			assigned = remaining
		}
		result = append(result, Assignment{OutputIndex: i, Amount: assigned})
		remaining -= assigned
	}
	return result, remaining == 0
}

// AllocateRegular applies native whole-transaction fallback after assets have
// been aggregated and ordered. It is also used by the indexer, which must track
// burns and overlapping fallback colors rather than reject valid Bitcoin txs.
func AllocateRegular(assets []Asset, outputs []*wire.TxOut, partial bool) (*Result, error) {
	result := &Result{
		Assignments: make(map[string][]Assignment, len(assets)),
		Burned:      make(map[string]int64),
	}
	for _, output := range outputs {
		if output == nil || output.Value < 0 {
			return nil, fmt.Errorf("invalid ARC20 transaction output")
		}
	}
	seen := make(map[string]bool, len(assets))
	for _, asset := range assets {
		if asset.AtomicalID == "" || asset.Amount <= 0 || seen[asset.AtomicalID] {
			return nil, fmt.Errorf("invalid or duplicate ARC20 aggregate %q", asset.AtomicalID)
		}
		seen[asset.AtomicalID] = true
		result.Order = append(result.Order, asset.AtomicalID)
	}
	start := 0
	for _, asset := range assets {
		assignments, complete := AssignRegular(outputs, start, asset.Amount, partial)
		if !complete && (!partial || len(assignments) == 0) {
			result.Fallback = true
			break
		}
		result.Assignments[asset.AtomicalID] = assignments
		if len(assignments) > 0 {
			start = assignments[len(assignments)-1].OutputIndex + 1
		}
	}
	if result.Fallback {
		for _, asset := range assets {
			assignments, _ := AssignRegular(outputs, 0, asset.Amount, partial)
			result.Assignments[asset.AtomicalID] = assignments
		}
	}
	for _, asset := range assets {
		remaining := asset.Amount
		for _, assignment := range result.Assignments[asset.AtomicalID] {
			remaining -= assignment.Amount
		}
		if remaining > 0 {
			result.Burned[asset.AtomicalID] = remaining
		}
	}
	return result, nil
}

// ColorRegular validates the supplied ARC20 facts, aggregates each native asset,
// and applies the activation-height-specific ordering. Bitcoin fees and unrelated
// protocols must still be checked by the caller. No ordinal offsets are involved.
func ColorRegular(inputs []Input, outputs []*wire.TxOut, height int, heights ActivationHeights) (*Result, error) {
	if heights.Activation <= 0 || heights.Dmint < heights.Activation ||
		heights.CustomColoring < heights.Dmint || height < heights.Activation {
		return nil, fmt.Errorf("ARC20 regular transfer is not active at height %d", height)
	}
	amounts := make(map[string]int64)
	ids := make([]string, 0)
	for index, input := range inputs {
		if input.Value < 0 {
			return nil, fmt.Errorf("negative ARC20 input value at %d", index)
		}
		local := make([]string, 0, len(input.Assets))
		seen := make(map[string]bool, len(input.Assets))
		for _, asset := range input.Assets {
			if _, err := AtomicalIDKey(asset.AtomicalID); err != nil {
				return nil, err
			}
			if seen[asset.AtomicalID] || asset.Amount <= 0 || asset.Amount > input.Value {
				return nil, fmt.Errorf("invalid ARC20 input allocation at %d", index)
			}
			seen[asset.AtomicalID] = true
			if amounts[asset.AtomicalID] > math.MaxInt64-asset.Amount {
				return nil, fmt.Errorf("ARC20 amount overflow")
			}
			if amounts[asset.AtomicalID] == 0 {
				local = append(local, asset.AtomicalID)
			}
			amounts[asset.AtomicalID] += asset.Amount
		}
		sort.Slice(local, func(i, j int) bool { return CompareAtomicalIDs(local[i], local[j]) < 0 })
		ids = append(ids, local...)
	}
	if height < heights.Dmint {
		sort.Slice(ids, func(i, j int) bool { return CompareAtomicalIDs(ids[i], ids[j]) < 0 })
	}
	assets := make([]Asset, 0, len(ids))
	for _, id := range ids {
		assets = append(assets, Asset{AtomicalID: id, Amount: amounts[id]})
	}
	return AllocateRegular(assets, outputs, height >= heights.CustomColoring)
}

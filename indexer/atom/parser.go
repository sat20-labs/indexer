package atom

import (
	"encoding/hex"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/fxamacker/cbor/v2"
	"github.com/sat20-labs/indexer/common"
)

var officialTickerPattern = regexp.MustCompile(`^[a-z0-9]{1,21}$`)
var bitworkPattern = regexp.MustCompile(`^[a-f0-9]{1,64}$`)
var hexPattern = regexp.MustCompile(`^[a-f0-9]+$`)

func ParseOperation(tx *common.Transaction, allowArgsBytes bool) *Operation {
	for inputIndex, input := range tx.Inputs {
		for _, script := range input.Witness {
			op, payload, ok := parseWitnessScript(script, allowArgsBytes)
			if !ok {
				continue
			}
			return &Operation{
				Op:          op,
				Payload:     payload,
				InputIndex:  inputIndex,
				CommitTxId:  input.TxID(),
				CommitIndex: int(input.OutPoint().Index),
			}
		}
	}
	return nil
}

func parseWitnessScript(script []byte, allowArgsBytes bool) (string, *Payload, bool) {
	if len(script) < 39 || script[0] != 0x20 {
		return "", nil, false
	}
	for i := 33; i < len(script)-6; i++ {
		if script[i] != txscript.OP_IF {
			continue
		}
		if hex.EncodeToString(script[i+1:i+6]) != "0461746f6d" {
			continue
		}
		op, next := parseOperationCode(script, i+6)
		if op == "" {
			continue
		}
		payloadBytes, err := parsePushPayload(script, next)
		if err != nil {
			return "", nil, false
		}
		payload := &Payload{Raw: payloadBytes, Args: make(map[string]any)}
		if len(payloadBytes) != 0 {
			var decoded map[string]any
			if err := cbor.Unmarshal(payloadBytes, &decoded); err != nil {
				return "", nil, false
			}
			if !sanitizePayloadFields(decoded, allowArgsBytes) {
				return "", nil, false
			}
			if args, ok := decoded["args"].(map[any]any); ok {
				payload.Args = normalizeMap(args)
			} else if args, ok := decoded["args"].(map[string]any); ok {
				payload.Args = normalizeStringMap(args)
			} else if op == OpSplit || op == OpCustomColor {
				payload.Args = normalizeStringMap(decoded)
			}
		}
		return op, payload, true
	}
	return "", nil, false
}

func sanitizePayloadFields(decoded map[string]any, allowArgsBytes bool) bool {
	return sanitizePayloadField(decoded, "meta", false) &&
		sanitizePayloadField(decoded, "args", allowArgsBytes) &&
		sanitizePayloadField(decoded, "ctx", false) &&
		sanitizePayloadField(decoded, "init", true)
}

func sanitizePayloadField(decoded map[string]any, name string, allowBytes bool) bool {
	raw, ok := decoded[name]
	if !ok || raw == nil {
		return true
	}
	switch v := raw.(type) {
	case map[string]any:
		return sanitizeDictWhitelistOnly(v, allowBytes)
	case map[any]any:
		return sanitizeDictWhitelistOnly(normalizeMap(v), allowBytes)
	default:
		return false
	}
}

func parseOperationCode(script []byte, pos int) (string, int) {
	if pos+4 <= len(script) {
		switch hex.EncodeToString(script[pos : pos+4]) {
		case "036e6674":
			return "nft", pos + 4
		case "03646674":
			return OpDeployDFT, pos + 4
		case "036d6f64":
			return "mod", pos + 4
		case "03657674":
			return "evt", pos + 4
		case "03646d74":
			return OpMintDFT, pos + 4
		case "03646174":
			return "dat", pos + 4
		}
	}
	if pos+3 <= len(script) {
		switch hex.EncodeToString(script[pos : pos+3]) {
		case "026674":
			return OpDirectFT, pos + 3
		case "02736c":
			return "sl", pos + 3
		}
	}
	if pos+2 <= len(script) {
		switch hex.EncodeToString(script[pos : pos+2]) {
		case "0178":
			return "x", pos + 2
		case "0179":
			return OpSplit, pos + 2
		case "017a":
			return OpCustomColor, pos + 2
		}
	}
	return "", -1
}

func parsePushPayload(script []byte, pos int) ([]byte, error) {
	payload := make([]byte, 0)
	for pos < len(script) {
		op := script[pos]
		pos++
		if op == txscript.OP_ENDIF {
			return payload, nil
		}
		data, next, ok := readPush(script, pos, op)
		if !ok {
			continue
		}
		payload = append(payload, data...)
		pos = next
	}
	return nil, fmt.Errorf("atomicals payload missing endif")
}

func readPush(script []byte, pos int, op byte) ([]byte, int, bool) {
	size := -1
	switch {
	case op < txscript.OP_PUSHDATA1:
		size = int(op)
	case op == txscript.OP_PUSHDATA1:
		if pos >= len(script) {
			return nil, pos, false
		}
		size = int(script[pos])
		pos++
	case op == txscript.OP_PUSHDATA2:
		if pos+2 > len(script) {
			return nil, pos, false
		}
		size = int(script[pos]) | int(script[pos+1])<<8
		pos += 2
	case op == txscript.OP_PUSHDATA4:
		if pos+4 > len(script) {
			return nil, pos, false
		}
		size = int(script[pos]) | int(script[pos+1])<<8 | int(script[pos+2])<<16 | int(script[pos+3])<<24
		pos += 4
	default:
		return nil, pos, false
	}
	if size < 0 || pos+size > len(script) {
		return nil, pos, false
	}
	return script[pos : pos+size], pos + size, true
}

func sanitizeDecodedMap(v any, allowBytes bool) bool {
	switch t := v.(type) {
	case map[string]any:
		for _, item := range t {
			if !sanitizeDecodedMap(item, allowBytes) {
				return false
			}
		}
	case map[any]any:
		for k, item := range t {
			if _, ok := k.(string); !ok {
				return false
			}
			if !sanitizeDecodedMap(item, allowBytes) {
				return false
			}
		}
	case []any:
		for _, item := range t {
			if !sanitizeDecodedMap(item, allowBytes) {
				return false
			}
		}
	case []byte:
		return allowBytes
	case string, uint64, int64, int, bool, nil:
		return true
	default:
		return false
	}
	return true
}

func sanitizeDictWhitelistOnly(raw map[string]any, allowBytes bool) bool {
	for _, v := range raw {
		switch item := v.(type) {
		case map[string]any:
			if !sanitizeDictWhitelistOnly(item, allowBytes) {
				return false
			}
		case map[any]any:
			if !sanitizeDictWhitelistOnly(normalizeMap(item), allowBytes) {
				return false
			}
		case []byte:
			if !allowBytes {
				return false
			}
		case int, int64, uint64, float32, float64, string, bool, []any, []string, []int, []uint64, []int64:
			continue
		default:
			return false
		}
	}
	return true
}

func normalizeMap(raw map[any]any) map[string]any {
	result := make(map[string]any, len(raw))
	for k, v := range raw {
		ks, ok := k.(string)
		if !ok {
			continue
		}
		result[ks] = normalizeValue(v)
	}
	return result
}

func normalizeStringMap(raw map[string]any) map[string]any {
	result := make(map[string]any, len(raw))
	for k, v := range raw {
		result[k] = normalizeValue(v)
	}
	return result
}

func normalizeValue(v any) any {
	switch mv := v.(type) {
	case map[any]any:
		return normalizeMap(mv)
	case map[string]any:
		return normalizeStringMap(mv)
	default:
		return v
	}
}

func isValidTicker(ticker string) bool {
	return officialTickerPattern.MatchString(ticker)
}

func IsValidTicker(ticker string) bool {
	return isValidTicker(ticker)
}

func stringArg(args map[string]any, name string) string {
	v, ok := args[name]
	if !ok || v == nil {
		return ""
	}
	switch t := v.(type) {
	case string:
		return t
	case []byte:
		return string(t)
	default:
		return fmt.Sprintf("%v", t)
	}
}

func intArg(args map[string]any, name string) (int64, bool) {
	v, ok := args[name]
	if !ok || v == nil {
		return 0, false
	}
	switch t := v.(type) {
	case int:
		return int64(t), true
	case int64:
		return t, true
	case uint64:
		if t > uint64(^uint64(0)>>1) {
			return 0, false
		}
		return int64(t), true
	case string:
		n, err := strconv.ParseInt(t, 10, 64)
		return n, err == nil
	default:
		return 0, false
	}
}

func boolArg(args map[string]any, name string) bool {
	v, ok := args[name]
	if !ok || v == nil {
		return false
	}
	switch t := v.(type) {
	case bool:
		return t
	case string:
		return t == "true" || t == "1"
	default:
		return false
	}
}

func parseBitwork(bitwork string) (string, int, bool) {
	if bitwork == "" {
		return "", 0, false
	}
	parts := strings.Split(bitwork, ".")
	if len(parts) > 2 || parts[0] == "" {
		return "", 0, false
	}
	if !bitworkPattern.MatchString(parts[0]) {
		return "", 0, false
	}
	ext := 0
	if len(parts) == 2 {
		n, err := strconv.Atoi(parts[1])
		if err != nil || n < 0 || n > 15 {
			return "", 0, false
		}
		ext = n
	}
	return parts[0], ext, true
}

func bitworkTarget(bitwork string) (int, bool) {
	prefix, ext, ok := parseBitwork(bitwork)
	if !ok {
		return 0, false
	}
	return len(prefix)*16 + ext, true
}

func isBitworkMatch(txid, bitwork string) bool {
	prefix, ext, ok := parseBitwork(bitwork)
	if !ok {
		return false
	}
	if !strings.HasPrefix(txid, prefix) {
		return false
	}
	if ext == 0 {
		return true
	}
	if len(txid) <= len(prefix) {
		return false
	}
	next, err := strconv.ParseInt(txid[len(prefix):len(prefix)+1], 16, 8)
	if err != nil {
		return false
	}
	return int(next) >= ext
}

func deriveBitworkPrefix(base string, target int64) (string, bool) {
	if target < 16 {
		return "", false
	}
	base = strings.ToLower(base)
	padded := base + strings.Repeat("0", 32)
	if len(padded) > 32 {
		padded = padded[:32]
	}
	fullAmount := int(target / 16)
	modulo := target % 16
	prefix := padded
	if fullAmount < 32 {
		prefix = padded[:fullAmount]
	}
	if modulo > 0 {
		return fmt.Sprintf("%s.%d", prefix, modulo), true
	}
	return prefix, true
}

func calculateExpectedBitwork(bitworkVec string, actualMints, maxMints, targetIncrement, startingTarget int64) (string, bool) {
	if startingTarget < 64 || startingTarget > 256 {
		return "", false
	}
	if maxMints < 1 || maxMints > 100000 {
		return "", false
	}
	if targetIncrement < 1 || targetIncrement > 64 {
		return "", false
	}
	targetSteps := actualMints / maxMints
	currentTarget := startingTarget + targetSteps*targetIncrement
	return deriveBitworkPrefix(bitworkVec, currentTarget)
}

func nextBitworkFullPrefix(bitworkVec string, currentPrefixLen int) string {
	padded := bitworkVec + strings.Repeat("0", 32)
	if len(padded) > 32 {
		padded = padded[:32]
	}
	if currentPrefixLen >= 31 {
		return padded
	}
	return padded[:currentPrefixLen+1]
}

func isPerpetualBitworkMatch(txid, bitworkVec string, actualMints, maxMints, targetIncrement, startingTarget int64, allowHigher bool) bool {
	expected, ok := calculateExpectedBitwork(bitworkVec, actualMints, maxMints, targetIncrement, startingTarget)
	if !ok {
		return false
	}
	if isBitworkMatch(txid, expected) {
		return true
	}
	if !allowHigher {
		return false
	}
	prefix, _, ok := parseBitwork(expected)
	if !ok {
		return false
	}
	return isBitworkMatch(txid, nextBitworkFullPrefix(bitworkVec, len(prefix)))
}

func compactId(txid string, index int) string {
	return fmt.Sprintf("%si%d", txid, index)
}

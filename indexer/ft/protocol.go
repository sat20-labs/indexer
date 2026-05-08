package ft

import (
	"fmt"

	"github.com/btcsuite/btcd/txscript"
)

const (
	// 需要跟 satoshinet/indexer/common/transcend.go 保持一致
	sat20MagicNumber    = txscript.OP_16
	contentTypeUnbind   = txscript.OP_DATA_40
	contentTypeFreeze   = txscript.OP_DATA_43
	contentTypeUnfreeze = txscript.OP_DATA_44
)

func readScriptInt(tokenizer *txscript.ScriptTokenizer) (int64, error) {
	switch tokenizer.Opcode() {
	case txscript.OP_0:
		return 0, nil
	case txscript.OP_1NEGATE:
		return -1, nil
	case txscript.OP_1, txscript.OP_2, txscript.OP_3, txscript.OP_4,
		txscript.OP_5, txscript.OP_6, txscript.OP_7, txscript.OP_8,
		txscript.OP_9, txscript.OP_10, txscript.OP_11, txscript.OP_12,
		txscript.OP_13, txscript.OP_14, txscript.OP_15, txscript.OP_16:
		return int64(tokenizer.Opcode() - (txscript.OP_1 - 1)), nil
	}

	data := tokenizer.Data()
	if data == nil {
		return 0, fmt.Errorf("opcode %d does not encode an integer", tokenizer.Opcode())
	}
	value, err := txscript.MakeScriptNum(data, true, 8)
	if err != nil {
		return 0, err
	}
	return int64(value), nil
}

func parseProtocolScript(script []byte, expectedType byte) (*txscript.ScriptTokenizer, bool, error) {
	tokenizer := txscript.MakeScriptTokenizer(0, script)
	if !tokenizer.Next() || tokenizer.Err() != nil || tokenizer.Opcode() != txscript.OP_RETURN {
		return nil, false, nil
	}
	if !tokenizer.Next() || tokenizer.Err() != nil || tokenizer.Opcode() != sat20MagicNumber {
		return nil, false, nil
	}
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return nil, false, fmt.Errorf("protocol script missing content type")
	}
	contentType, err := readScriptInt(&tokenizer)
	if err != nil {
		return nil, false, err
	}
	if contentType != int64(expectedType) {
		return nil, false, nil
	}
	return &tokenizer, true, nil
}

// ParseUnbindScript parses:
// OP_RETURN OP_16 40 <ticker> <vout>
// where ticker is a pushed string and vout uses txscript's compressed integer format.
func ParseUnbindScript(script []byte) (string, int, bool, error) {
	tokenizer, matched, err := parseProtocolScript(script, contentTypeUnbind)
	if err != nil || !matched {
		return "", 0, matched, err
	}
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return "", 0, true, fmt.Errorf("unbind script missing ticker")
	}
	tickerData := tokenizer.Data()
	if len(tickerData) == 0 {
		return "", 0, true, fmt.Errorf("unbind script has empty ticker")
	}
	ticker := string(tickerData)
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return "", 0, true, fmt.Errorf("unbind script missing vout")
	}
	value, err := readScriptInt(tokenizer)
	if err != nil {
		return "", 0, true, err
	}
	if value < 0 {
		return "", 0, true, fmt.Errorf("invalid vout %d", value)
	}
	if tokenizer.Next() {
		return "", 0, true, fmt.Errorf("unbind script has unexpected trailing data")
	}
	if tokenizer.Err() != nil {
		return "", 0, true, tokenizer.Err()
	}
	return ticker, int(value), true, nil
}

func ParseFreezeScript(script []byte) (string, string, int, bool, error) {
	tokenizer, matched, err := parseProtocolScript(script, contentTypeFreeze)
	if err != nil || !matched {
		return "", "", 0, matched, err
	}
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return "", "", 0, true, fmt.Errorf("freeze script missing ticker")
	}
	tickerData := tokenizer.Data()
	if len(tickerData) == 0 {
		return "", "", 0, true, fmt.Errorf("freeze script has empty ticker")
	}
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return "", "", 0, true, fmt.Errorf("freeze script missing address")
	}
	addressData := tokenizer.Data()
	if len(addressData) == 0 {
		return "", "", 0, true, fmt.Errorf("freeze script has empty address")
	}
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return "", "", 0, true, fmt.Errorf("freeze script missing height")
	}
	value, err := readScriptInt(tokenizer)
	if err != nil {
		return "", "", 0, true, err
	}
	if value < 0 {
		return "", "", 0, true, fmt.Errorf("invalid freeze height %d", value)
	}
	if tokenizer.Next() {
		return "", "", 0, true, fmt.Errorf("freeze script has unexpected trailing data")
	}
	if tokenizer.Err() != nil {
		return "", "", 0, true, tokenizer.Err()
	}
	return string(tickerData), string(addressData), int(value), true, nil
}

func ParseUnfreezeScript(script []byte) (string, string, bool, error) {
	tokenizer, matched, err := parseProtocolScript(script, contentTypeUnfreeze)
	if err != nil || !matched {
		return "", "", matched, err
	}
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return "", "", true, fmt.Errorf("unfreeze script missing ticker")
	}
	tickerData := tokenizer.Data()
	if len(tickerData) == 0 {
		return "", "", true, fmt.Errorf("unfreeze script has empty ticker")
	}
	if !tokenizer.Next() || tokenizer.Err() != nil {
		return "", "", true, fmt.Errorf("unfreeze script missing address")
	}
	addressData := tokenizer.Data()
	if len(addressData) == 0 {
		return "", "", true, fmt.Errorf("unfreeze script has empty address")
	}
	if tokenizer.Next() {
		return "", "", true, fmt.Errorf("unfreeze script has unexpected trailing data")
	}
	if tokenizer.Err() != nil {
		return "", "", true, tokenizer.Err()
	}
	return string(tickerData), string(addressData), true, nil
}

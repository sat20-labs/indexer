package brc20

import (
	"github.com/btcsuite/btcd/txscript"
	"github.com/sat20-labs/indexer/common"
)

type InscriptionStatus int

const (
    InscriptionValid InscriptionStatus = iota
    InscriptionCursed
    InscriptionVindicated
)

// 检查是否为符合规范的 inscription 封装
func isStandardEnvelope(script []byte) bool {
    // 标准格式: OP_0 OP_IF <inscription content> OP_ENDIF
    if len(script) < 3 {
        return false
    }
    return script[0] == 0x00 && script[1] == 0x63 && script[len(script)-1] == 0x68
}

// 检查是否包含“奇怪的push”导致cursed
func hasWeirdPush(script []byte, inscId int) bool {
    // 早期被判为cursed的典型情况：
    // 1. 使用 OP_PUSHNUM_x (0x51-0x60) 而不是 OP_PUSHBYTES
    // 2. 使用 OP_CAT, OP_2DROP 等不安全操作码
    // 3. 多重 inscription in one output（多个 OP_IF 封装）

	if inscId != 0 {
		return true
	}

	tokenizer := txscript.MakeScriptTokenizer(0, script)
	for tokenizer.Next() {
		b := tokenizer.Opcode()
		 if b >= 0x51 && b <= 0x60 { // OP_PUSHNUM_1 到 OP_PUSHNUM_16
            return true
        }
        // 旧规则禁止的
        if b == 0x7e || b == 0x6d || b == 0x6f { // OP_CAT, OP_2DROP, OP_IFDUP 等
            return true
        }
	}

    return false
}

// Jubilee 后 vindicated 的条件（简化版）
func isVindicated(height int) bool {
	// 816000, 使用ord v0.9版本的定义
	// 824544，jubilee，cursed 铭文得到vindicated
    // 从 Jubilee 起，部分早期 cursed 被“洗白”
    return height >= common.Jubilee_Height
        // ordinals 0.14+ 允许重新索引这些特例
}

// 判断铭文状态
func DetectInscriptionStatus(script []byte, inscId int, blockHeight int) InscriptionStatus {
	// 规则 1: 标准封装 OP_0 OP_IF ... OP_ENDIF
	if !isStandardEnvelope(script) {
		// 不符合标准结构
		if isVindicated(blockHeight) {
			return InscriptionVindicated
		}
		return InscriptionCursed
	}

	// 规则 2: 检查是否使用奇怪的 push（导致 early cursed）
	if hasWeirdPush(script, inscId) {
		if isVindicated(blockHeight) {
			return InscriptionVindicated
		}
		return InscriptionCursed
	}

	// 标准封装 + 无奇怪push → 有效铭文
	return InscriptionValid
}

func IsCursed(script []byte, inscId int, blockHeight int) bool {
	return DetectInscriptionStatus(script, inscId, blockHeight) == InscriptionCursed
}

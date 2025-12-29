package common

import (
	"encoding/base64"
	"fmt"
	"sort"
	"time"

	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
	"github.com/btcsuite/btcd/wire"
	"github.com/sat20-labs/indexer/common/pb"
)

const RANGE_IN_GLOBAL = false // true: Range 表示一个satoshi的全局编码，一个 [0, 2099999997690000) 的数字
// false: Range表示特殊聪在当前utxo中的范围。使用false，可以极大降低数据存储需求

type Range = pb.PbRange

type Input struct {
	Txid     string         `json:"txid"`
	UtxoId   uint64         `json:"utxoid"`
	Address  *ScriptPubKey  `json:"scriptPubKey"`
	Vout     int64          `json:"vout"`
	Value    int64          `json:"value"`
	Ordinals []*Range       `json:"ordinals"`
	Witness  wire.TxWitness `json:"witness"`
}

type ScriptPubKey struct {
	Addresses []string `json:"addresses"`
	Type      int      `json:"type"`
	ReqSig    int      `json:"reqSig"`
	PkScript  []byte   `json:"pkscript"`
}

type Output struct {
	Height   int           `json:"height"`
	TxId     int           `json:"txid"`
	Value    int64         `json:"value"`
	Address  *ScriptPubKey `json:"scriptPubKey"`
	N        int64         `json:"n"`
	Ordinals []*Range      `json:"ordinals"`
}

type TxInput struct {
	TxOutputV2 // 作为输入的utxo的信息
	Witness    wire.TxWitness
	// 当前交易的输入信息
	TxId      string // 作为输入时的交易
	InHeight  int
	InTxIndex int
	TxInIndex int
}

// TxOutputV2 是 TxOutput 的“编译期加速版本”，仅用于区块遍历 / 交易编译阶段。
// 
// 使用约束（Invariants）：
// 
// 1. 仅允许线性使用
//    - 同一个 TxOutputV2 实例，只能按 value 从小到大进行 Cut
//    - 不允许回退 Cut
//    - 不允许对同一个 TxOutputV2 做分叉式 Cut（例如同时切给多个消费者）
//    - offsetCursor / satCursor 均依赖这一前提
//
// 2. Append 阶段的 Assets 不是最终形态
//    - Append 不做去重、不做排序
//    - 同一个 AssetName 可能在 Assets 中出现多次
//    - Cut 阶段必须通过 TxAssetsBuilder 进行 rebuild
//    - 任何在 Cut 之前直接读取 Assets 的逻辑都是错误的
//
// 3. Offsets 与 SatBindingMap 的协议约束
//    - ord / ordx 资产：
//        * 必须存在 Offsets
//        * BindingSat > 0
//    - brc20 资产：
//        * 在“铸造结果 utxo”中必须存在 Offsets + SatBindingMap
//        * 一旦资产被转移到新 utxo：
//            - Offsets 与 SatBindingMap 必须被清空
//        * Cut 阶段假设：Offsets 中每个 range.Start 对应一个 SatBindingMap 项
//
// 4. SatBindingMap 的生命周期
//    - SatBindingMap 中的 *AssetInfo 不允许在多个 TxOutput / TxOutputV2 之间共享
//    - Append 阶段必须 Clone AssetInfo
//
// 5. 非绑定资产（BindingSat == 0）
//    - len(Offsets) == 0 被视为“不参与 Cut 的资产”
//    - 当前仅在编译期 brc20 / runes 场景下成立
//    - 若未来引入新的协议类型，需要重新审视该逻辑
//
// 违反以上任何一条，都会导致资产错配或 silently 错误。
type TxOutputV2 struct {
	TxOutput
	TxOutIndex  int
	OutTxIndex  int
	OutHeight   int
	AddressId   uint64
	AddressType int

	// 仅用于编译数据
	// asset -> 当前 offsets 游标（指向正在消费的 range）
	base int64 // 已经切出去的部分，所有剩余offset在切出去时都需要减去这部分
	offsetCursor map[AssetName]int
	// asset -> 已排序的 sat offsets（仅 brc20 使用）
	satKeys map[AssetName][]int
	// asset -> satKeys 游标
	satCursor map[AssetName]int
}

func (p *TxOutputV2) GetAddress() string {
	switch txscript.ScriptClass(p.AddressType) {
	case txscript.NullDataTy:
		return "OP_RETURN"
	}

	var chainParams *chaincfg.Params
	if IsMainnet() {
		chainParams = &chaincfg.MainNetParams
	} else {
		chainParams = &chaincfg.TestNet4Params
	}
	_, addresses, _, _ := txscript.ExtractPkScriptAddrs(p.OutValue.PkScript, chainParams)
	if len(addresses) == 0 {
		// txscript.MultiSigTy, NonStandardTy
		return base64.StdEncoding.EncodeToString(p.OutValue.PkScript)
	}

	return addresses[0].EncodeAddress()
}

func NewCompilingOutput(tx *TxOutputV2) *TxOutputV2 {
	return &TxOutputV2{
		TxOutput: *tx.Clone(),

		base: 0,
		offsetCursor: make(map[AssetName]int),
		satKeys: make(map[AssetName][]int),
		satCursor: make(map[AssetName]int),
	}
}

// 为编译数据增加两个新的函数，加快处理速度，小心内存碎片的处理
// Append 是编译期快路径：
// - 不做资产合并、不排序
// - 仅保证 offsets / sat 偏移正确
// - 结果必须通过 Cut + TxAssetsBuilder 才能成为合法 TxOutput
func (p *TxOutputV2) Append(another *TxOutput) error {

	if another == nil {
		return nil
	}

	base := p.OutValue.Value
	p.OutValue.Value += another.OutValue.Value

	// === Assets + Offsets ===
	for _, asset := range another.Assets {
		if invalid, ok := another.Invalids[asset.Name]; ok && invalid {
			continue
		}

		// 资产直接加（编译期不做去重 / rebuild）
		p.Assets = append(p.Assets, *asset.Clone())

		offsets, ok := another.Offsets[asset.Name]
		if !ok {
			continue
		}

		dst := p.Offsets[asset.Name]
		if dst == nil {
			dst = make(AssetOffsets, 0, len(offsets))
		}

		// 直接 append，做 offset 平移
		for _, r := range offsets {
			dst.Cat_NoSafe(&OffsetRange{
				Start: r.Start + base,
				End:   r.End + base,
			})
		}
		p.Offsets[asset.Name] = dst
	}

	// === SatBindingMap（brc20） ===
	if len(another.SatBindingMap) > 0 {
		for k, v := range another.SatBindingMap {
			if invalid, ok := another.Invalids[v.Name]; ok && invalid {
				continue
			}

			p.SatBindingMap[k+base] = v.Clone()
		}
	}

	p.UtxoId = INVALID_ID
	p.OutPointStr = ""

	return nil
}

// Cut 使用 offsetCursor / satCursor 实现 O(n) 线性切分
// 前提：
// - 对同一个 TxOutputV2，Cut 只能按 value 单调递增调用
// - builder.Build() 是保证 Assets 正确性的步骤
func (p *TxOutputV2) Cut(offset int64) (*TxOutput, error) {
	if p == nil {
		return nil, fmt.Errorf("TxOutput is nil")
	}
	if offset < 0 || offset > p.OutValue.Value {
		return nil, fmt.Errorf("invalid offset")
	}

	start := p.base
	end := p.base+offset
	if end > p.OutValue.Value {
		return nil, fmt.Errorf("invalid offset")
	}

	if start == 0 && end == p.OutValue.Value {
		builder1 := NewTxAssetsBuilder(len(p.Assets))
		for _, asset := range p.Assets {
			if invalid, ok := p.Invalids[asset.Name]; ok && invalid {
				continue
			}
			builder1.Add(&asset)
		}
		p.Assets = builder1.Build()
		p.base += offset
		return &p.TxOutput, nil
	}

	part1 := NewTxOutput(offset)
	builder1 := NewTxAssetsBuilder(len(p.Assets))

	// === Assets / Offsets ===
	for _, asset := range p.Assets {
		if invalid, ok := p.Invalids[asset.Name]; ok && invalid {
			continue
		}

		offsets := p.Offsets[asset.Name]
		if len(offsets) == 0 {
			// 非绑定资产直接进 part1
			builder1.AddClone(&asset)
			//part1.Assets = append(part1.Assets, *asset.Clone())
			continue
		}

		cur := p.offsetCursor[asset.Name]
		var off1 AssetOffsets

		for cur < len(offsets) {
			r := offsets[cur]

			if r.Start >= end {
				break
			}
			if r.End <= start {
				// 完全左边
				cur++
				continue
			} 
			// r.End > start && r.Start < end
			if r.Start >= start {
				// range starts within [start, end)
				if r.End <= end {
					off1 = append(off1, &OffsetRange{Start: r.Start - start, End: r.End - start}) // 中间
					cur++
				} else {
					off1 = append(off1, &OffsetRange{Start: r.Start - start, End: end - start}) // 最右
					break
				}
			} else {
				// r.Start < start, overlaps from the left
				if r.End <= end {
					off1 = append(off1, &OffsetRange{Start: 0, End: r.End - start}) // 最左
					cur++
				} else {
					off1 = append(off1, &OffsetRange{Start: 0, End: end - start}) // 内切
					break
				}
			}
		}
		p.offsetCursor[asset.Name] = cur

		if len(off1) > 0 {
			part1.Offsets[asset.Name] = off1
		}

		// === 资产数量 ===
		if asset.BindingSat > 0 {
			if len(off1) > 0 {
				amt := off1.Size() * int64(asset.BindingSat)
				n := AssetInfo{
					Name:       asset.Name,
					Amount:     *NewDefaultDecimal(amt),
					BindingSat: asset.BindingSat,
				}
				builder1.Add(&n)
				//part1.Assets = append(part1.Assets, n)
			}
		} else {
			// brc20
			if len(off1) > 0 {
				var amt *Decimal
				for _, r := range off1 {
					o, ok := p.SatBindingMap[r.Start+p.base]
					if !ok {
						Log.Panicf("can't find asset info for sat %d in %s", r.Start, p.OutPointStr)
					}
					amt = amt.Add(&o.Amount)
				}
				n := AssetInfo{
					Name:       asset.Name,
					Amount:     *amt,
					BindingSat: 0,
				}
				builder1.Add(&n)
				//part1.Assets = append(part1.Assets, n)
			}
		}
	}
	part1.Assets = builder1.Build()

	// === brc20 SatBindingMap ===
	if len(p.SatBindingMap) > 0 {
		for name := range p.Offsets {
			if p.satKeys[name] == nil { // 初始化
				keys := make([]int, 0)
				for k, v := range p.SatBindingMap {
					if v.Name == name {
						keys = append(keys, int(k))
					}
				}
				sort.Ints(keys)
				p.satKeys[name] = keys
			}

			keys := p.satKeys[name]
			cur := p.satCursor[name]

			for cur < len(keys) {
				k := int64(keys[cur])
				v := p.SatBindingMap[k]

				if k < offset {
					part1.SatBindingMap[k] = v.Clone()
				}
				cur++
			}
			p.satCursor[name] = cur
		}
	}

	p.base += offset
	return part1, nil
}



func GetPkScriptFromAddress(address string) ([]byte, error) {
	if address == "OP_RETURN" {
		return []byte{0x6a}, nil
	}
	// if address == "UNKNOWN" {
	// 	return []byte{0x51}, nil
	// }
	var chainParams *chaincfg.Params
	if IsMainnet() {
		chainParams = &chaincfg.MainNetParams
	} else {
		chainParams = &chaincfg.TestNet4Params
	}

	pkScript, err := AddrToPkScript(address, chainParams)
	if err != nil {
		// base64
		pkScript, err = base64.StdEncoding.DecodeString(address)
	}
	return pkScript, err
}

func GetAddressTypeFromAddress(address string) int {
	pkScript, err := GetPkScriptFromAddress(address)
	if err != nil {
		return int(txscript.NonStandardTy)
	}
	return GetAddressTypeFromPkScript(pkScript)
}

func GetAddressTypeFromPkScript(pkScript []byte) int {
	var chainParams *chaincfg.Params
	if IsMainnet() {
		chainParams = &chaincfg.MainNetParams
	} else {
		chainParams = &chaincfg.TestNet4Params
	}
	scriptClass, _, _, err := txscript.ExtractPkScriptAddrs(pkScript, chainParams)
	if err != nil {
		return int(txscript.NonStandardTy)
	}
	return int(scriptClass)
}

type Transaction struct {
	TxId    string        `json:"txid"`
	Inputs  []*TxInput    `json:"inputs"`
	Outputs []*TxOutputV2 `json:"outputs"`
}

type Block struct {
	Timestamp     time.Time      `json:"timestamp"`
	Height        int            `json:"height"`
	Hash          string         `json:"hash"`
	PrevBlockHash string         `json:"prevBlockHash"`
	Transactions  []*Transaction `json:"transactions"`
}

type UTXOIndex struct {
	Index map[string]*TxOutputV2
}

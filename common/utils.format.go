// ConvertTimestampToISO8601 将时间戳转换为 ISO 8601 格式的字符串
package common

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/gob"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

)

func ConvertTimestampToISO8601(timestamp int64) string {
	// 将时间戳转换为 time.Time 类型
	t := time.Unix(timestamp, 0).UTC()

	// 检查时间戳是否合法
	if t.IsZero() {
		Log.Error("invalid timestamp")
		return ""
	}

	// 将时间格式化为 ISO 8601 格式的字符串
	//iso8601 := t.Format("2006-01-02T15:04:05Z")
	iso8601 := t.Format(time.RFC3339)

	return iso8601
}

func ParseUtxo(utxo string) (txid string, vout int, err error) {
	parts := strings.Split(utxo, ":")
	if len(parts) != 2 {
		return txid, vout, fmt.Errorf("invalid utxo")
	}

	txid = parts[0]
	vout, err = strconv.Atoi(parts[1])
	if err != nil {
		return txid, vout, err
	}
	if vout < 0 {
		return txid, vout, fmt.Errorf("invalid vout")
	}
	return txid, vout, err
}

func ParseAddressIdKey(addresskey string) (addressId uint64, utxoId uint64, typ, index int64,  err error) {
	parts := strings.Split(addresskey, "-")
	if len(parts) < 4 {
		return INVALID_ID, INVALID_ID, 0, 0, fmt.Errorf("invalid address key %s", addresskey)
	}
	addressId, err = strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return INVALID_ID, INVALID_ID, 0, 0, err
	}
	utxoId, err = strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return INVALID_ID, INVALID_ID, 0, 0, err
	}
	typ, err = strconv.ParseInt(parts[3], 16, 32)
	if err != nil {
		return INVALID_ID, INVALID_ID, 0, 0, err
	}
	index = 0
	if len(parts) > 4 {
		index, err = strconv.ParseInt(parts[4], 16, 32)
		if err != nil {
			return INVALID_ID, INVALID_ID, 0, 0, err
		}
	}
	return addressId, utxoId, typ, index, err
}

func ConvertToUtxoId(height int, tx int, vout int) uint64 {
	// 极少情况下，vout会大于0xffff，目前在主网上没看到，只在测试网络看到，比如高度2578308
	if height > 0x7fffffff || tx > 0xffff || vout > 0x1ffff {
		Log.Panicf("parameters too big %x %x %x", height, tx, vout)
	}

	return (uint64(height)<<33 | uint64(tx)<<17 | uint64(vout))
}

func ConvertFromUtxoId(id uint64) (int, int, int) {
	return (int)(id >> 33), (int)(uint32((id >> 17) & 0xffff)), int((uint32(id)) & 0x1ffff)
}

func GetUtxoId(addrAndId *Output) uint64 {
	return ConvertToUtxoId(addrAndId.Height, addrAndId.TxId, int(addrAndId.N))
}

func ParseOrdInscriptionID(inscriptionID string) (txid string, index int, err error) {
	parts := strings.Split(inscriptionID, "i")
	if len(parts) != 2 {
		return "", 0, fmt.Errorf("invalid inscriptionID")
	}
	txid = parts[0]
	index, err = strconv.Atoi(parts[1])
	if err != nil {
		return txid, index, err
	}
	if index < 0 {
		return txid, index, fmt.Errorf("invalid index")
	}
	return txid, index, nil
}

func ParseOrdSatPoint(satPoint string) (txid string, outputIndex int, offset int64, err error) {
	parts := strings.Split(satPoint, ":")
	if len(parts) != 3 {
		return "", 0, 0, fmt.Errorf("invalid satPoint")
	}
	txid = parts[0]
	outputIndex, err = strconv.Atoi(parts[1])
	if err != nil {
		return txid, outputIndex, 0, err
	}
	if outputIndex < 0 {
		return txid, outputIndex, 0, fmt.Errorf("invalid index")
	}

	offset, err = strconv.ParseInt(parts[2], 10, 64)
	if err != nil {
		return txid, outputIndex, offset, err
	}
	return txid, outputIndex, offset, nil
}

func GenerateSeed(data interface{}) string {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(data)
	if err != nil {
		return "0"
	}

	hash := sha256.New()
	_, err = hash.Write(buf.Bytes())
	if err != nil {
		return "0"
	}
	// 获取哈希结果
	hashBytes := hash.Sum(nil)
	// 将哈希值转换为 uint64
	result := binary.LittleEndian.Uint64(hashBytes[:8])

	return fmt.Sprintf("%x", result)
}

func GenerateSeed2(ranges []*Range) string {
	bytes, err := json.Marshal(ranges)
	if err != nil {
		Log.Errorf("json.Marshal failed. %v", err)
		return "0"
	}

	//fmt.Printf("%s\n", string(bytes))

	hash := sha256.New()
	hash.Write(bytes)
	hashResult := hash.Sum(nil)
	return hex.EncodeToString(hashResult[:8])
}

// 二分查找函数，返回插入位置的索引
func binarySearch(arr []*UtxoIdInDB, utxoId uint64) int {
	left := 0
	right := len(arr)

	for left < right {
		mid := left + (right-left)/2

		if arr[mid].UtxoId == utxoId {
			return mid
		} else if arr[mid].UtxoId < utxoId {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left
}

// 快速插入函数
func InsertUtxo(arr []*UtxoIdInDB, utxo *UtxoIdInDB) []*UtxoIdInDB {
	index := binarySearch(arr, utxo.UtxoId)
	if index < len(arr) && arr[index].UtxoId == utxo.UtxoId {
		arr[index].Value = utxo.Value
		return arr
	}

	// 在 index 位置插入新元素
	arr = append(arr, nil)
	copy(arr[index+1:], arr[index:])
	arr[index] = utxo

	return arr
}

// 快速删除函数
func DeleteUtxo(arr []*UtxoIdInDB, utxoId uint64) []*UtxoIdInDB {
	index := binarySearch(arr, utxoId)

	// 如果找到匹配的 utxoId，则从数组中删除对应元素
	if index < len(arr) && arr[index].UtxoId == utxoId {
		copy(arr[index:], arr[index+1:])
		arr = arr[:len(arr)-1]
	}

	return arr
}

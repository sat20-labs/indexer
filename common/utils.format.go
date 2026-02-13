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

func TxIdFromUtxo(utxo string) string {
	parts := strings.Split(utxo, ":")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
}

func TxIdFromInscriptionId(id string) string {
	parts := strings.Split(id, "i")
	if len(parts) != 2 {
		return ""
	}
	return parts[0]
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

func ParseAddressIdKey(addresskey string) (addressId uint64, utxoId uint64, err error) {
	parts := strings.Split(addresskey, "-")
	if len(parts) < 2 {
		return INVALID_ID, INVALID_ID, fmt.Errorf("invalid address key %s", addresskey)
	}
	addressId, err = strconv.ParseUint(parts[1], 16, 64)
	if err != nil {
		return INVALID_ID, INVALID_ID, err
	}
	utxoId, err = strconv.ParseUint(parts[2], 16, 64)
	if err != nil {
		return INVALID_ID, INVALID_ID, err
	}
	return addressId, utxoId, err
}


func ToUtxo(txid string, vout int) string {
	return txid+":"+strconv.Itoa(vout)
}

/*
最小交易总大小 = 82 bytes
区块大小限制：4MB (4,000,000 bytes)
理论最大交易数 = 4,000,000 / 82 ≈ 48,780 笔交易
每个输入最小大小：41 bytes (前一个输出点36 + 序列号4 + varint 1)
理论最大输入数 = (4,000,000 - 10) / 41 ≈ 97,560个
每个输出最小大小：31 bytes (value 8 + varint 1 + 最小脚本22)
理论最大输出数 = (4,000,000 - 10) / 31 ≈ 129,032个

Height: 29bit 0x1fffffff  	< 536870911
tx: 	17bit 0x1ffff 		< 131071
vout:	18bit 0x3ffff 		< 262143
*/
func ToUtxoId(height int, tx int, vout int) uint64 {
	if height > 0x1fffffff || tx > 0x1ffff || vout > 0x3ffff {
		Log.Panicf("parameters too big %x %x %x", height, tx, vout)
	}

	return (uint64(height)<<35 | uint64(tx)<<18 | uint64(vout))
}

func FromUtxoId(id uint64) (int, int, int) {
	return (int)(id >> 35), (int)((id >> 18) & 0x1ffff), (int)((id) & 0x3ffff)
}

func GetUtxoId(addrAndId *Output) uint64 {
	return ToUtxoId(addrAndId.Height, addrAndId.TxId, int(addrAndId.N))
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

// sat为全局统一编码时的计算方式
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

// 通过切片操作删除元素（保持原有顺序）
func RemoveIndex[T any](slice []T, index int) []T {
    return append(slice[:index], slice[index+1:]...)
}

// 大端序下，高位字节先比较 → 字节序比较行为与整数比较行为一致。
// 如果采用pebble数据库，所有数据库的KEY，如果是键值是整数，都转换为这个格式
func Uint64ToBytes(value uint64) []byte {
	bytes := make([]byte, 8)
	binary.BigEndian.PutUint64(bytes, value)
	return bytes
}

func BytesToUint64(bytes []byte) uint64 {
	return binary.BigEndian.Uint64(bytes)
}

func Uint64ToString(value uint64) string {
	b := Uint64ToBytes(value)
	return hex.EncodeToString(b)
}

func StringToUint64(str string) (uint64, error) {
	b, err := hex.DecodeString(str)
	if err != nil {
		Log.Errorf("DecodeString %s failed, %v", str, err)
		return 0, err
	}
	return BytesToUint64(b), nil
} 

func Uint32ToBytes(value uint32) []byte {
	bytes := make([]byte, 4)
	binary.BigEndian.PutUint32(bytes, value)
	return bytes
}

func BytesToUint32(bytes []byte) uint32 {
	return binary.BigEndian.Uint32(bytes)
}

func Uint32ToString(value uint32) string {
	b := Uint32ToBytes(value)
	return hex.EncodeToString(b)
}

func StringToUint32(str string) (uint32, error) {
	b, err := hex.DecodeString(str)
	if err != nil {
		Log.Errorf("DecodeString %s failed, %v", str, err)
		return 0, err
	}
	return BytesToUint32(b), nil
}

func CheckUtxoFormat(utxo string) error {
	parts := strings.Split(utxo, ":")
	_, err := hex.DecodeString(parts[0])
	if err != nil {
		return fmt.Errorf("wrong utxo format %v", err)
	}
	return nil
}


// 二分查找函数，返回插入位置的索引
func binarySearch_uint64(arr []uint64, id uint64) int {
	left := 0
	right := len(arr)

	for left < right {
		mid := left + (right-left)/2

		if arr[mid] == id {
			return mid
		} else if arr[mid] < id {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left
}

// 快速插入函数
func InsertVector_uint64(arr []uint64, id uint64) []uint64 {
	index := binarySearch_uint64(arr, id)
	if index < len(arr) && arr[index] == id {
		// 有同样的存在
		return arr
	}

	// 在 index 位置插入新元素
	arr = append(arr, 0)
	copy(arr[index+1:], arr[index:])
	arr[index] = id

	return arr
}

// 快速删除函数
func DeleteFromVector_uint64(arr []uint64, id uint64) []uint64 {
	index := binarySearch_uint64(arr, id)

	// 如果找到匹配的 id， 则从数组中删除对应元素
	if index < len(arr) && arr[index] == id {
		copy(arr[index:], arr[index+1:])
		arr = arr[:len(arr)-1]
	}

	return arr
}


// 二分查找函数，返回插入位置的索引
func binarySearch_string(arr []string, id string) int {
	left := 0
	right := len(arr)

	for left < right {
		mid := left + (right-left)/2

		if arr[mid] == id {
			return mid
		} else if arr[mid] < id {
			left = mid + 1
		} else {
			right = mid
		}
	}

	return left
}

// 快速插入函数
func InsertVector_string(arr []string, id string) []string {
	index := binarySearch_string(arr, id)
	if index < len(arr) && arr[index] == id {
		// 有同样的存在
		return arr
	}

	// 在 index 位置插入新元素
	arr = append(arr, "")
	copy(arr[index+1:], arr[index:])
	arr[index] = id

	return arr
}

// 快速删除函数
func DeleteFromVector_string(arr []string, id string) []string {
	index := binarySearch_string(arr, id)

	// 如果找到匹配的 id， 则从数组中删除对应元素
	if index < len(arr) && arr[index] == id {
		copy(arr[index:], arr[index+1:])
		arr = arr[:len(arr)-1]
	}

	return arr
}

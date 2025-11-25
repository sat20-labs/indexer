package common

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/btcsuite/btcd/txscript"
	"github.com/fxamacker/cbor/v2"
	"lukechampine.com/uint128"
)

func readBytes(raw []byte, pointer *int, n int) []byte {
	value := raw[*pointer : *pointer+n]
	*pointer += n
	return value
}

func getBeginPosition(raw []byte) (int, int) {
	inscriptionMark := []byte{0, txscript.OP_IF, 3, 111, 114, 100}                         // "0063036f7264"
	inscriptionMark2 := []byte{0, txscript.OP_IF, txscript.OP_PUSHDATA1, 3, 111, 114, 100} // "00634c036f7264"

	position := bytes.Index(raw, inscriptionMark)
	if position >= 0 {
		return int(position + len(inscriptionMark)), len(inscriptionMark)
	}

	position = bytes.Index(raw, inscriptionMark2)
	if position >= 0 {
		return int(position + len(inscriptionMark2)), len(inscriptionMark2)
	}

	return -1, 0
}

// 从跳过信封头开始
func getEndPosition(raw []byte) int {
	length := len(raw)
	i := 0
	for i < length {
		opcode := raw[i]
		if opcode == txscript.OP_0 {
			i++
			getContentLength(raw, &i)
		} else if txscript.OP_DATA_1 <= opcode && opcode <= txscript.OP_DATA_75 {
			i++
			i += int(opcode)
		} else if opcode >= txscript.OP_PUSHDATA1 && opcode <= txscript.OP_PUSHDATA4 {
			getPushDataLength(raw, &i)
		} else if opcode >= txscript.OP_1 && opcode <= txscript.OP_16 {
			i++
		} else if opcode == txscript.OP_1NEGATE { // testnet: f8fc655ffe139d9952e673c53b7d15cb4b82de5ef036c7fc1211262bbd29bec8
			i++
		} else if opcode == txscript.OP_ENDIF {
			return i
		} else {
			Log.Warnf("unsupport op_code %d ", opcode)
			return -1
		}
	}

	Log.Warnf("not find OP_ENDIF")
	return -2
}

func readPushData(raw []byte, posPointer *int, opcode byte) []byte {
	if txscript.OP_DATA_1 <= opcode && opcode <= txscript.OP_DATA_75 {
		return readBytes(raw, posPointer, int(opcode))
	}

	if opcode >= txscript.OP_1 && opcode <= txscript.OP_16 {
		byt := raw[*posPointer-1] - txscript.OP_1 + 1
		return []byte{byt}
	}

	if opcode == txscript.OP_1NEGATE {
		return []byte{opcode}
	}

	var numBytes int = 0
	var size int = 0
	switch opcode {
	case txscript.OP_PUSHDATA1:
		numBytes = 1
	case txscript.OP_PUSHDATA2:
		numBytes = 2
	case txscript.OP_PUSHDATA4:
		numBytes = 4
	default:
		return nil
	}
	sizeBytes := readBytes(raw, posPointer, numBytes)
	switch opcode {
	case txscript.OP_PUSHDATA1:
		size = int(sizeBytes[0])
	case txscript.OP_PUSHDATA2:
		size = int(binary.LittleEndian.Uint16(sizeBytes))
	case txscript.OP_PUSHDATA4:
		size = int(binary.LittleEndian.Uint32(sizeBytes))
	}
	return readBytes(raw, posPointer, size)
}

func readContent(raw []byte, pos *int) (content []byte, err error) {
	data := []byte{}
	opcode := readBytes(raw, pos, 1)
	if opcode[0] == txscript.OP_ENDIF {
		*pos--
		return nil, nil
	}
	chunk := readPushData(raw, pos, opcode[0])
	for chunk != nil {
		data = append(data, chunk...)
		opcode = readBytes(raw, pos, 1)
		if opcode[0] == txscript.OP_ENDIF {
			*pos--
			break
		} else if opcode[0] == txscript.OP_0 {
			// 某些情况会用OP_0分割，跳过
			opcode = readBytes(raw, pos, 1)
		}
		chunk = readPushData(raw, pos, opcode[0])
		if chunk == nil {
			*pos--
		}
	}
	return data, nil
}

func getPushDataLength(raw []byte, posPointer *int) int {
	opcode := raw[*posPointer]
	if txscript.OP_DATA_1 <= opcode && opcode <= txscript.OP_DATA_75 {
		*posPointer++
		*posPointer += int(opcode)
		return int(opcode)
	}

	if opcode >= txscript.OP_1 && opcode <= txscript.OP_16 {
		*posPointer++
		return 1
	}

	var numBytes int = 0
	var size int = 0
	switch opcode {
	case txscript.OP_PUSHDATA1:
		numBytes = 1
	case txscript.OP_PUSHDATA2:
		numBytes = 2
	case txscript.OP_PUSHDATA4:
		numBytes = 4
	default:
		return 0
	}
	*posPointer++
	sizeBytes := readBytes(raw, posPointer, numBytes)
	switch opcode {
	case txscript.OP_PUSHDATA1:
		size = int(sizeBytes[0])
	case txscript.OP_PUSHDATA2:
		size = int(binary.LittleEndian.Uint16(sizeBytes))
	case txscript.OP_PUSHDATA4:
		size = int(binary.LittleEndian.Uint32(sizeBytes))
	}
	*posPointer += size
	return size
}

func getContentLength(raw []byte, pos *int) int {
	total := 0
	length := getPushDataLength(raw, pos)
	for length > 0 {
		total += length
		opcode := raw[*pos]
		if opcode == txscript.OP_ENDIF {
			break
		} else if opcode == txscript.OP_0 {
			// 某些情况会用OP_0分割，跳过
			*pos++
		}
		length = getPushDataLength(raw, pos)
	}
	return total
}

func ParseInscription(txWitness [][]byte) ([]map[int][]byte, [][]byte, error) {
	// 规则：一个信封，就是一次铭刻。
	// 无效情况：1. 存在不支持的指令；2.信封内部嵌套信封
	// 可能存在任何一个witness

	result := make([]map[int][]byte, 0)
	envelopes := make([][]byte, 0)
	for _, raw := range txWitness {

		pos := int(0)
		for pos < len(raw) {
			begin, skip := getBeginPosition(raw[pos:])
			if begin < 0 {
				break
			}
			begin += pos

			end := getEndPosition(raw[begin:])
			if end < 0 {
				break
			}
			end += begin
			pos = end + 1

			envelope := raw[begin:pos]
			envelopes = append(envelopes, raw[begin-skip:pos])

			fields := make(map[int][]byte)
			length := end - begin
			i := 0
			for i < length {
				opcode := envelope[i]
				if opcode == txscript.OP_0 {
					// body
					i++
					content, err := readContent(envelope, &i)
					if err != nil {
						break
					}
					fields[FIELD_CONTENT] = content
				} else if opcode == txscript.OP_ENDIF {
					i++
					break
				} else {
					// read tags
					i++
					tagType := readPushData(envelope, &i, opcode)
					if tagType == nil {
						continue
					} else {
						opcode = envelope[i]
						i++
						tagContent := readPushData(envelope, &i, opcode)

						if len(tagType) == 1 {
							fields[int(tagType[0])] = tagContent
						} else {
							if tagContent == nil {
								fields[FIELD_INVALID1] = tagType
							} else if len(tagContent) == 1 {
								fields[int(tagContent[0])] = tagType
							} else {
								fields[FIELD_INVALID1] = tagType
								fields[FIELD_INVALID2] = tagContent
							}
						}
					}
				}
			}
			result = append(result, fields)
		}

	}

	return result, envelopes, nil
}

func GetBasicContent(content string) *OrdxBaseContent {
	var ordxContent OrdxBaseContent
	err := json.Unmarshal([]byte(content), &ordxContent)
	if err != nil {
		return nil
	}

	return &ordxContent
}

func ParseDeployContent(content string) *OrdxDeployContent {
	var ret OrdxDeployContent
	err := json.Unmarshal([]byte(content), &ret)
	if err != nil {
		Log.Warnf("invalid json: %s, %v", content, err)
		return nil
	}
	ret.Ticker = PreprocessName(ret.Ticker)
	// if strings.Contains(ret.Ticker, " ") {
	// 	Log.Warnf("invalid ticker name: %s", ret.Ticker)
	// 	return nil
	// }
	return &ret
}

func ParseMintContent(content string) *OrdxMintContent {
	var ret OrdxMintContent
	err := json.Unmarshal([]byte(content), &ret)
	if err != nil {
		return nil
	}
	return &ret
}

func StrictlyUnmarshal(jsonString string, targetStructPtr interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(jsonString))
	decoder.DisallowUnknownFields()
	return decoder.Decode(targetStructPtr)
}

func ParseBrc20BaseContent(content string) *BRC20BaseContent {
	var ret BRC20BaseContent
	err := json.Unmarshal([]byte(content), &ret)
	if err != nil {
		Log.Warnf("invalid json: %s, %v", content, err)
		return nil
	}
	if err := ret.Validate(); err != nil {
		return nil
	}
	return &ret
}

func ParseBrc20DeployContent(content string) *BRC20DeployContent {
	var ret BRC20DeployContent
	err := StrictlyUnmarshal(content, &ret)
	if err != nil {
		return nil
	}
	if err := ret.Validate(); err != nil {
		return nil
	}
	return &ret
}

func ParseBBRC20AmtContent(content string) *BRC20AmtContent {
	var ret BRC20AmtContent
	err := StrictlyUnmarshal(content, &ret)
	if err != nil {
		return nil
	}
	if err := ret.Validate(); err != nil {
		return nil
	}
	return &ret
}

func ParseBrc20MintContent(content string) *BRC20MintContent {
	var ret BRC20MintContent
	err := StrictlyUnmarshal(content, &ret)
	if err != nil {
		return nil
	}
	if err := ret.Validate(); err != nil {
		return nil
	}
	return &ret
}

func ParseBrc20TransferContent(content string) *BRC20TransferContent {
	var ret BRC20TransferContent
	err := StrictlyUnmarshal(content, &ret)
	if err != nil {
		return nil
	}
	if err := ret.Validate(); err != nil {
		return nil
	}
	return &ret
}

func Cbor2json(cborData []byte) ([]byte, error) {
	if cborData == nil {
		return nil, fmt.Errorf("no data")
	}
	var decodedData map[string]string
	err := cbor.Unmarshal(cborData, &decodedData)
	if err != nil {
		return nil, err
	}
	jsonData, err := json.Marshal(decodedData)
	if err != nil {
		return nil, err
	}
	return (jsonData), nil
}

func Json2cbor(jsonData []byte) ([]byte, error) {
	if jsonData == nil {
		return nil, fmt.Errorf("no data")
	}
	var decodedData map[string]string
	err := json.Unmarshal(jsonData, &decodedData)
	if err != nil {
		return nil, err
	}

	cborData, err := cbor.Marshal(decodedData)
	if err != nil {
		return nil, err
	}
	return (cborData), nil
}

func GetSatpoint(spBytes []byte) int {
	// ab28fc85219361cd62d1302048e160d7632903b1bde4c6158c005f05ea46bd02
	l := len(spBytes)
	if l == 2 {
		return int(binary.LittleEndian.Uint16(spBytes))
	} else if l == 4 {
		return int(binary.LittleEndian.Uint32(spBytes))
	} else if l == 1 {
		return int(spBytes[0])
	} else if l == 3 {
		// cc bb aa -> 0xaabbcc
		// 4988a700aec5d1c14d7a55f96d97cea2afdff11d8e284d0bb388514e6a3d2958
		return int(spBytes[2])<<16 + int(spBytes[1])<<8 + int(spBytes[0])
	} else {
		return 0
	}
}

func IsOrdXProtocol(fields map[int][]byte) (string, bool) {
	var content string

	content = string((fields)[FIELD_CONTENT])
	protocol, ok := (fields)[FIELD_META_PROTOCOL]
	if ok {
		if string(protocol) == PROTOCOL_NAME_ORDX {
			jsonStr, err := Cbor2json((fields)[FIELD_META_DATA])
			if err != nil {
				return content, false
			} else {
				content = string(jsonStr)
			}
		}
	}

	var ordxContent OrdxBaseContent
	err := json.Unmarshal([]byte(content), &ordxContent)
	if err != nil {
		return content, false
	}

	return content, ordxContent.P == PROTOCOL_NAME_ORDX
}

func GetProtocol(fields map[int][]byte) (string, []byte) {
	content := (fields)[FIELD_CONTENT]
	protocol, ok := (fields)[FIELD_META_PROTOCOL]
	if ok {
		jsonStr, err := Cbor2json((fields)[FIELD_META_DATA])
		if err == nil {
			content = jsonStr
		}
		return string(protocol), content
	}

	var ordxContent OrdxBaseContent
	err := json.Unmarshal([]byte(content), &ordxContent)
	if err != nil {
		return "", nil
	}

	return ordxContent.P, content
}

func ParseInscriptionId(input []byte) string {
	/*
		010320b5cbc7526bf2619bc912e7584bb47d414e3f3bd2e209bfbc1edb162b5ddfb2fd
			OP_PUSHBYTES_1 03
			OP_PUSHBYTES_32 b5cbc7526bf2619bc912e7584bb47d414e3f3bd2e209bfbc1edb162b5ddfb2fd
		010b20a072a699867de6b7a87956d6d36d926d91d47a39e6e08bb7b848899135bf76ed
			OP_PUSHBYTES_1 0b
			OP_PUSHBYTES_32 a072a699867de6b7a87956d6d36d926d91d47a39e6e08bb7b848899135bf76ed
			实际的值：
			03-parent: fdb2df5d2b16db1ebcbf09e2d23b3f4e417db44b58e712c99b61f26b52c7cbb5i0
			0b-delegate：ed76bf35918948b8b78be0e6397ad4916d926dd3d65679a8b7e67d8699a672a0i0
			需要做转换: serialized as the 32-byte TXID, followed by the four-byte little-endian INDEX,
			with trailing zeroes omitted.
			0 被忽略
	*/

	if len(input) < 32 {
		return ""
	}

	// Reverse the byte slice
	reverseBytes := make([]byte, 32)
	for i := 0; i < 32; i++ {
		reverseBytes[i] = input[32-1-i]
	}

	// Convert reversed bytes to hex string
	txid := hex.EncodeToString(reverseBytes)
	index := 0
	if len(input) > 32 {
		indexBytes := make([]byte, 4)
		for i := 0; i+32 < len(input) && i < 4; i++ {
			indexBytes[i] = input[32+i]
		}
		index = int(binary.LittleEndian.Uint32(indexBytes))
	}

	return txid + "i" + strconv.Itoa(index)
}

// ordinals, tag 13
// 长度要确保小于16
func ParseRunesName(input []byte) uint128.Uint128 {
	/*
		To prevent front running an etching that has been broadcast but not mined,
		if a non-reserved rune name is being etched, the etching transaction must contain
		a valid commitment to the name being etched.
		A commitment consists of a data push of the rune name, encoded as a little-endian integer
		with trailing zero bytes elided, present in an input witness tapscript where the output
		being spent has at least six confirmations.
		If a valid commitment is not present, the etching is ignored.
	*/

	// Reverse the byte slice
	length := len(input)

	// runes name
	// reverseBytes := make([]byte, length)
	// for i := 0; i < length; i++ {
	// 	reverseBytes[i] = input[length-1-i]
	// }
	left := make([]byte, 16-length)
	le128 := append(input, left...)

	return uint128.FromBytes(le128)
}

func IsValidName(name string) bool {
	// 使用正则表达式匹配标点符号/空格/控制符
	if name == "" {
		return false
	}
	reg := regexp.MustCompile(`[\pP\pZ\pC]`)
	return !reg.MatchString(name)
}

func IsValidSat20Name(name string) bool {
	return IsValidName(name) && IsValidNameLen(name)
}

func IsValidNameLen(name string) bool {
	tickLen := len(name)
	return (tickLen >= MIN_NAME_LEN && tickLen <= MAX_NAME_LEN)
}

func PreprocessName(name string) string {
	return strings.TrimSpace(name)
}

func IsValidSNSName(name string) bool {
	// 参考标准：https://docs.btcname.id/docs/overview/chapter-4-thinking-about-.btc-domain-name/calibration-rules
	if len(name) > MAX_NAME_LEN {
		return false
	}

	if strings.Contains(name, "\n") || strings.Contains(name, " ") {
		return false
	}

	parts := strings.Split(name, ".")
	l := len(parts)
	bReg := false
	if l == 1 {
		bReg = IsValidName(parts[0]) // 无后缀名字
	} else if l == 2 {
		bReg = IsValidName(parts[0]) && IsValidName(parts[1])
	}
	return bReg
}

func CloneBaseContent(base *InscribeBaseContent) *InscribeBaseContent {
	return &InscribeBaseContent{
		InscriptionId:      base.InscriptionId,
		InscriptionAddress: base.InscriptionAddress,
		BlockHeight:        base.BlockHeight,
		BlockTime:          base.BlockTime,
		ContentType:        base.ContentType,
		ContentEncoding:    base.ContentEncoding,
		Content:            base.Content,
		MetaProtocol:       base.MetaProtocol,
		MetaData:           base.MetaData,
		Parent:             base.Parent,
		Delegate:           base.Delegate,
		Id:                 base.Id,
		BasePoint:          base.BasePoint,
		Sat:                base.Sat,
		CurseType:          base.CurseType,
		TypeName:           base.TypeName,
		UserData:           base.UserData,
	}
}

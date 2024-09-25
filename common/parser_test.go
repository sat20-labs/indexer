package common

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCborParser(t *testing.T) {
	// 示例 JSON 数据
	jsonData := []byte(`{"name": "Alice", "age": "30"}`)

	// 将 JSON 数据转换为 CBOR 数据
	cborData, err := Json2cbor(jsonData)
	if err != nil {
		fmt.Printf("Error converting JSON to CBOR: %v\n", err)
		return
	}
	fmt.Printf("CBOR Data: %x\n", cborData)

	// 将 CBOR 数据转换回 JSON 数据
	jsonDataConvertedBack, err := Cbor2json(cborData)
	if err != nil {
		fmt.Printf("Error converting CBOR to JSON: %v\n", err)
		return
	}
	fmt.Printf("JSON Data: %s\n", jsonDataConvertedBack)
}

func TestParser(t *testing.T) {
	// 1f8863156b8c53aeddcf912cbb02884e0b1379920cd698c8f9080e126ba98593 html testnet
	// 2e05e8f64955ecf31e2ba411af16cbb3d47cb225f2cd45039955c96282612006 png testnet
	// f542b9ba7637d50f5b27264ef7a24cc0b0bce2860f141cc8ef5e704ef59b9ead tradition testnet
	// 9d7b92da52b0d18ad9586cdad3b1c68c558cb816d516c7911d30ce95bf45d1e6 mainnet
	rawData, err := GetRawData("3cef7be93fa6a71caa40b861f4d1789bfcc743a583339655f599f4f10e8f7f6b", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	Log.Infof("ordxInfo: %s, contentType: %s, content: %s", string(fields[0][FIELD_META_DATA]), string(fields[0][FIELD_CONTENT_TYPE]), string(fields[0][FIELD_CONTENT]))
}

// satpoint指向同一个uxto中的不同sat
func TestParser_Satpoint(t *testing.T) {
	rawData, err := GetRawData("4cc11aed71720c8e14c757c903bbf7b7b0c9aa2be4daafe7f50ef70f86a7bcc7", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1000 {
		assert.True(t, false)
	}

	for _, field := range fields {
		spBytes := field[FIELD_POINT]
		satpoint := 0
		if len(spBytes) > 0 {
			satpoint = GetSatpoint(spBytes)
		}

		Log.Infof("sat point %s %d", hex.EncodeToString(field[FIELD_POINT]), satpoint)
	}

}

func TestParser_specialmetadata(t *testing.T) {
	// 5d2482d01100e2ab44906a676949ebcb62aa898e4b81aea6d7630edd4b00eb1c
	// rawData, err := GetRawData("31134bdf5018f0b4d0d634cc70dbd18bbb82fb0770ac308b3876a36cadc2eb0b", "testnet")
	rawData, err := GetRawData("5d2482d01100e2ab44906a676949ebcb62aa898e4b81aea6d7630edd4b00eb1c", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	for _, field := range fields {
		Log.Info(string(field[FIELD_CONTENT]))
	}

}

func TestParser_specialmedia(t *testing.T) {
	rawData, err := GetRawData("f38b2001c65b9d6b4b54203ec14f6c5497336ce725ed1f08fa918a187ef3ea1f", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	if len(fields[0][FIELD_CONTENT]) != 256997 {
		assert.True(t, false)
	}
}

func TestParser_specialprotocol(t *testing.T) {
	rawData, err := GetRawData("193f95863e0cedc66562ff29a1e7f6a5caabd2d755ab6456b8aa4e4ccadd6818", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	for _, field := range fields {
		Log.Info(string(field[FIELD_CONTENT]))
	}

}

func TestParser_specialcase1(t *testing.T) {
	rawData, err := GetRawData("2dc9a9d84565b096ba80f91620ee4a1d8dd924a695b63c32f38815f90cd121d1", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	for _, field := range fields {
		Log.Info(string(field[FIELD_CONTENT]))
	}

}

func TestParser_specialcase2(t *testing.T) {
	rawData, err := GetRawData("aa646942f39ca5f2eb0f2a442624a065271e54754b5152561c929071712ddc57", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	for _, field := range fields {
		Log.Info(string(field[FIELD_CONTENT]))
	}

}

func TestParser_metadata_deploy(t *testing.T) {
	rawData, err := GetRawData("2e05e8f64955ecf31e2ba411af16cbb3d47cb225f2cd45039955c96282612006", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	for _, field := range fields {
		for ty, data := range field {
			Log.Infof("%d:%s", ty, string(data))
		}
	}

	content, b := IsOrdXProtocol(fields[0])
	if !b {
		assert.True(t, false)
	}

	Log.Infof("%s", content)

}

func TestParser_metadata_mint(t *testing.T) {
	rawData, err := GetRawData("1f8863156b8c53aeddcf912cbb02884e0b1379920cd698c8f9080e126ba98593", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	for _, field := range fields {
		for ty, data := range field {
			Log.Infof("%d:%s", ty, string(data))
		}
	}

	content, b := IsOrdXProtocol(fields[0])
	if !b {
		assert.True(t, false)
	}

	Log.Infof("%s", content)
	ordxType := GetBasicContent(content)
	if ordxType.Op != "mint" {
		assert.True(t, false)
	}

	mintInfo := ParseMintContent(content)
	if mintInfo == nil {
		assert.True(t, false)
	}
}

func TestParser_ord1(t *testing.T) {
	// witness[0]
	rawData, err := GetRawData("2a0b461b76c182ef8d1ec457f84093bf5a2b925c8b8a6938b2775050be518255", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	for _, field := range fields {
		for ty, data := range field {
			Log.Infof("%d:%s", ty, string(data))
		}
	}
}

func TestParser_ord2(t *testing.T) {
	// don't recognite
	rawData, err := GetRawData("861c74973a4ab6be4f4c40690210d9095eab234f56173cfa82bfd07f4278febc", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	insc, _ := ParseInscription(rawData)

	if len(insc) > 0 {
		assert.True(t, false)
	}
}

func TestParser_ord3(t *testing.T) {
	// don't recognite
	rawData, err := GetRawData("2b6ded8c1a9fe1c017003a5783f3e42e0c903af80b2d49577a1c490815e671f1", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 3 {
		Log.Info(err)
		assert.True(t, false)
	}
}

func TestParser_ord4(t *testing.T) {
	// don't recognite
	rawData, err := GetRawData("dede288471de31da65f3cadd52b57094320a63f7faa8034a96a4a7f097856c88", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 5 {
		Log.Info(err)
		assert.True(t, false)
	}

}

func TestParseInscription(t *testing.T) {

	hexBytes := []byte{0x1f, 0x1e, 0x1d, 0x1c, 0x1b, 0x1a, 0x19, 0x18, 0x17, 0x16, 0x15, 0x14, 0x13, 0x12,
		0x11, 0x10, 0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00}
	result := ParseInscriptionId(hexBytes)
	if result != "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1fi0" {
		t.Fatal()
	}

	hexBytes = []byte{0x1f, 0x1e, 0x1d, 0x1c, 0x1b, 0x1a, 0x19, 0x18, 0x17, 0x16, 0x15, 0x14, 0x13, 0x12,
		0x11, 0x10, 0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00, 0xff}
	result = ParseInscriptionId(hexBytes)
	if result != "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1fi255" {
		t.Fatal()
	}

	hexBytes = []byte{0x1f, 0x1e, 0x1d, 0x1c, 0x1b, 0x1a, 0x19, 0x18, 0x17, 0x16, 0x15, 0x14, 0x13, 0x12,
		0x11, 0x10, 0x0f, 0x0e, 0x0d, 0x0c, 0x0b, 0x0a, 0x09, 0x08, 0x07, 0x06, 0x05, 0x04, 0x03, 0x02, 0x01, 0x00, 0x00, 0x01}
	result = ParseInscriptionId(hexBytes)
	if result != "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1fi256" {
		t.Fatal()
	}

}

func TestParser_ord5(t *testing.T) {
	// pointer
	rawData, err := GetRawData("a9e0ed50e0eb92274bbc78511ace1ba49ba993282b8c3d51da29efae8ce57bca", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	satpoint := GetSatpoint(fields[0][FIELD_POINT])

	if satpoint != 0x18d {
		Log.Info(err)
		assert.True(t, false)
	}
}

func TestParser_ns(t *testing.T) {
	// KVs
	update := OrdxUpdateContentV1{
		OrdxBaseContent: OrdxBaseContent{
			P:  "ordx",
			Op: "update",
		},
		Name: "12345",
		KVs:  nil,
	}

	kv1 := `key1=value1`
	kv2 := `key2=value2`

	update.KVs = append(update.KVs, kv1)
	update.KVs = append(update.KVs, kv2)

	bytes, err := json.Marshal(update)
	if err == nil {
		fmt.Printf("%s", string(bytes))
	} else {
		fmt.Printf("%v", err)
	}

}

func TestRegTest(t *testing.T) {
	inscriptionMark, _ := hex.DecodeString("0063036f7264")
	fmt.Printf("%v\n", inscriptionMark)
	str := "  \n Iou \n\n"
	str2 := PreprocessName(str)
	fmt.Printf("%s\n", str2)
	fmt.Printf("%v\n", IsValidSat20Name("12b1!"))
	fmt.Printf("%v\n", IsValidSat20Name("12b 1"))
	fmt.Printf("%v\n", IsValidSat20Name("12b11"))
	fmt.Printf("%v\n", IsValidSat20Name("12.b1"))
}

func TestParser_ord6(t *testing.T) {
	// invalid
	rawData, err := GetRawData("861c74973a4ab6be4f4c40690210d9095eab234f56173cfa82bfd07f4278febc", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	if len(fields) > 0 {
		assert.True(t, false)
	}
}

// 下面几个一起测试
func TestParser_nested(t *testing.T) {
	rawData, err := GetRawData("b484bd4e81aa74d5524e77278f70068d636e2c50e885dbfbb1f2591aad61e386", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if len(fields) != 0 {
		Log.Info(err)
		assert.True(t, false)
	}
}

func TestParser_ord7(t *testing.T) {
	// cursed
	rawData, err := GetRawData("2550cb512c61a03d87ce7a42f6e96b999a371ff9d8929d9be772ba20744d1de9", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	Log.Info(fields[0][FIELD_CONTENT_TYPE])
}

func TestParser_ord8(t *testing.T) {
	// invalid
	rawData, err := GetRawData("861c74973a4ab6be4f4c40690210d9095eab234f56173cfa82bfd07f4278febc", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) > 0 {
		assert.True(t, false)
	}
}

func TestParser_ord9(t *testing.T) {
	// invalid
	rawData, err := GetRawData("6f937fc7e60ea66cbccf584f22922e4c756e574d30ce9b8ed5aa8526eabf988d", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) > 0 {
		assert.True(t, false)
	}
}

func TestParser_ord10(t *testing.T) {
	// cursed
	rawData, err := GetRawData("75919b8b7e49d091e4c5dc3d61e2fa9dcfcc10f7232def081d852fe700bc25b9", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}
}

func TestParser_ord11(t *testing.T) {
	// cursed
	rawData, err := GetRawData("b9c4c69c0160c4a30a438ccd976d0c96a0a8296812121931c46ee370af729de2", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

func TestParser_ord12(t *testing.T) {
	// empty envelope
	rawData, err := GetRawData("ce7291326d22e1f2dffc12c118dfa464ad55d07ff9ada5a91b8ec9d9301a6f05", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, _ := ParseInscription(rawData)

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

func TestParser_ord13(t *testing.T) {
	// cursed
	rawData, err := GetRawData("c52311d91d666ddf9e27caffc84f9a3cd967b58aab97e44e828c90842d53dd79", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

func TestParser_ord14(t *testing.T) {
	// cursed
	rawData, err := GetRawData("5f7f8779ead5d786f13310b5e9239ee3cc1f4697ccd368ea5b3a37f785fca200", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

func TestParser_ord15(t *testing.T) {
	// invalid. why?
	rawData, err := GetRawData("c769750df54ee38fe2bae876dbf1632c779c3af780958a19cee1ca0497c78e80", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 0 {
		assert.True(t, false)
	}
}

func TestParser_ord16(t *testing.T) {
	// invalid. why?
	rawData, err := GetRawData("e8104e50ac9b0539a6bbb6f8e60436b086991a7e62920e9dc0f39053707a37a9", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 0 {
		assert.True(t, false)
	}
}

func TestParser_ord17(t *testing.T) {

	rawData, err := GetRawData("f8fc655ffe139d9952e673c53b7d15cb4b82de5ef036c7fc1211262bbd29bec8", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1000 {
		assert.True(t, false)
	}
}

func TestParser_ord18(t *testing.T) {
	// cursed
	rawData, err := GetRawData("fd1f01dc91580ebeb75fd8ecb3ee4efa9f9d3e94c726139cbd706192ad0edb03", "mainnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

func TestParser_ord19(t *testing.T) {
	// reinscription
	rawData, err := GetRawData("4c6479280e27c6b99c8de404921b6c813fc323df77e06c7f8f6aa08e6a81f148", "testnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 2 {
		assert.True(t, false)
	}
}

func TestParser_ord20(t *testing.T) {
	// input 0, output 0
	rawData, err := GetRawData("c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151", "mainnet")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

func TestParser_ord21(t *testing.T) {
	// input 0, output 0
	rawData, err := GetRawData("4e73e226998b37ea6eee0d904a17321e3c0f75abfd9c3b534845ea5ff345a9e3", "testnet4")
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}
	fields, err := ParseInscription(rawData)
	if err != nil {
		Log.Info(err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

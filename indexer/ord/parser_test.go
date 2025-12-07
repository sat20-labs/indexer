package ord

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/ord/ord0_14_1"
)


func GetTxRawData(txID string, network string) (string, error) {
	url := ""
	switch network {
	case "testnet":
		url = fmt.Sprintf("https://mempool.space/testnet/api/tx/%s/hex", txID)
	case "testnet4":
		url = fmt.Sprintf("https://mempool.space/testnet4/api/tx/%s/hex", txID)
	case "mainnet":
		url = fmt.Sprintf("https://mempool.space/api/tx/%s/hex", txID)
	default:
		return "", fmt.Errorf("unsupported network: %s", network)
	}

	response, err := http.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)

	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)
	}

	respBytes, err := io.ReadAll(response.Body)
	response.Body.Close()
	if err != nil {
		err = fmt.Errorf("error reading json reply: %v", err)
		return "", err
	}

	return string(respBytes), nil
}

func GetRawData(txID string, network string) ([][]byte, error) {
	url := ""
	switch network {
	case "testnet":
		url = fmt.Sprintf("https://mempool.space/testnet/api/tx/%s", txID)
	case "testnet4":
		url = fmt.Sprintf("https://mempool.space/testnet4/api/tx/%s", txID)
	case "mainnet":
		url = fmt.Sprintf("https://mempool.space/api/tx/%s", txID)
	default:
		return nil, fmt.Errorf("unsupported network: %s", network)
	}

	response, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)

	}
	defer response.Body.Close()

	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to retrieve transaction data for %s from the API, error: %v", txID, err)
	}

	var data map[string]interface{}
	err = json.NewDecoder(response.Body).Decode(&data)
	if err != nil {
		return nil, fmt.Errorf("failed to decode JSON response for %s, error: %v", txID, err)
	}
	txWitness := data["vin"].([]interface{})[0].(map[string]interface{})["witness"].([]interface{})

	if len(txWitness) < 2 {
		return nil, fmt.Errorf("failed to retrieve witness for %s", txID)
	}

	var rawData [][]byte = make([][]byte, len(txWitness))
	for i, v := range txWitness {
		rawData[i], err = hex.DecodeString(v.(string))
		if err != nil {
			return nil, fmt.Errorf("failed to decode hex string to byte array for %s, error: %v", txID, err)
		}
	}
	if err != nil {
		return nil, fmt.Errorf("failed to decode hex string to byte array for %s, error: %v", txID, err)
	}
	return rawData, nil
}

func ParseInscription(data [][]byte) ([]*ord0_14_1.InscriptionResult, []byte, error) {
	return ord0_14_1.GetInscriptionsInTxInput(data, 0, 0), nil, nil
}

func TestParser(t *testing.T) {
	// 1f8863156b8c53aeddcf912cbb02884e0b1379920cd698c8f9080e126ba98593 html testnet
	// 2e05e8f64955ecf31e2ba411af16cbb3d47cb225f2cd45039955c96282612006 png testnet
	// f542b9ba7637d50f5b27264ef7a24cc0b0bce2860f141cc8ef5e704ef59b9ead tradition testnet
	// 9d7b92da52b0d18ad9586cdad3b1c68c558cb816d516c7911d30ce95bf45d1e6 mainnet
	rawData, err := GetRawData("3cef7be93fa6a71caa40b861f4d1789bfcc743a583339655f599f4f10e8f7f6b", "testnet")
	if err != nil {
		fmt.Printf("%s", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fmt.Printf("%v", fields)
}

// satpoint指向同一个uxto中的不同sat
func TestParser_Satpoint(t *testing.T) {
	rawData, err := GetRawData("4cc11aed71720c8e14c757c903bbf7b7b0c9aa2be4daafe7f50ef70f86a7bcc7", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 1000 {
		assert.True(t, false)
	}

	for _, field := range fields {
		spBytes := field.Inscription.Pointer
		satpoint := 0
		if len(spBytes) > 0 {
			satpoint = common.GetSatpoint(spBytes)
		}

		fmt.Printf("sat point %s %d\n", hex.EncodeToString(field.Inscription.Pointer), satpoint)
	}

}

func TestParser_specialmetadata(t *testing.T) {
	// 5d2482d01100e2ab44906a676949ebcb62aa898e4b81aea6d7630edd4b00eb1c
	// rawData, err := GetRawData("31134bdf5018f0b4d0d634cc70dbd18bbb82fb0770ac308b3876a36cadc2eb0b", "testnet")
	rawData, err := GetRawData("5d2482d01100e2ab44906a676949ebcb62aa898e4b81aea6d7630edd4b00eb1c", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	// TODO 无法解析出body
	for _, field := range fields {
		fmt.Printf(string(field.Inscription.Body))
	}

}

func TestParser_specialmedia(t *testing.T) {
	rawData, err := GetRawData("f38b2001c65b9d6b4b54203ec14f6c5497336ce725ed1f08fa918a187ef3ea1f", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	if len(fields[0].Inscription.Body) != 256997 {
		assert.True(t, false)
	}
}

func TestParser_specialprotocol(t *testing.T) {
	rawData, err := GetRawData("193f95863e0cedc66562ff29a1e7f6a5caabd2d755ab6456b8aa4e4ccadd6818", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	for _, field := range fields {
		assert.True(t, field.IsCursed)
	}

}

func TestParser_specialcase1(t *testing.T) {
	rawData, err := GetRawData("2dc9a9d84565b096ba80f91620ee4a1d8dd924a695b63c32f38815f90cd121d1", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	// for _, field := range fields {
	// 	assert.True(t, field.IsCursed)
	// }

}

func TestParser_specialcase2(t *testing.T) {
	rawData, err := GetRawData("aa646942f39ca5f2eb0f2a442624a065271e54754b5152561c929071712ddc57", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}

	for _, field := range fields {
		fmt.Printf(string(field.Inscription.Body))
	}

}

func TestParser_ord1(t *testing.T) {
	// witness[0]
	rawData, err := GetRawData("2a0b461b76c182ef8d1ec457f84093bf5a2b925c8b8a6938b2775050be518255", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	for i, field := range fields {
		
			fmt.Printf("%d: %v", i, field)
		
	}
}

func TestParser_ord2(t *testing.T) {
	// don't recognite
	rawData, err := GetRawData("861c74973a4ab6be4f4c40690210d9095eab234f56173cfa82bfd07f4278febc", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	insc, _, _ := ParseInscription(rawData)

	if len(insc) > 0 {
		assert.True(t, false)
	}
}

func TestParser_ord3(t *testing.T) {
	// don't recognite
	rawData, err := GetRawData("2b6ded8c1a9fe1c017003a5783f3e42e0c903af80b2d49577a1c490815e671f1", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 3 {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
}

func TestParser_ord4(t *testing.T) {
	// don't recognite
	rawData, err := GetRawData("dede288471de31da65f3cadd52b57094320a63f7faa8034a96a4a7f097856c88", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 5 {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

}

func TestParser_ord5(t *testing.T) {
	// pointer
	rawData, err := GetRawData("a9e0ed50e0eb92274bbc78511ace1ba49ba993282b8c3d51da29efae8ce57bca", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	satpoint := common.GetSatpoint(fields[0].Inscription.Pointer)

	if satpoint != 0x18d {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
}

func TestParser_ord6(t *testing.T) {
	// invalid
	rawData, err := GetRawData("861c74973a4ab6be4f4c40690210d9095eab234f56173cfa82bfd07f4278febc", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
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
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if len(fields) != 0 {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
}

func TestParser_ord7(t *testing.T) {
	// cursed
	rawData, err := GetRawData("2550cb512c61a03d87ce7a42f6e96b999a371ff9d8929d9be772ba20744d1de9", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	for _, field := range fields {
		assert.True(t, field.IsCursed)
	}

	fmt.Printf(string(fields[0].Inscription.ContentType))
}

func TestParser_ord8(t *testing.T) {
	// invalid
	rawData, err := GetRawData("861c74973a4ab6be4f4c40690210d9095eab234f56173cfa82bfd07f4278febc", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
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
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
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
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	for _, field := range fields {
		assert.True(t, field.IsCursed)
	}

	if len(fields) == 0 {
		assert.True(t, false)
	}
}

func TestParser_ord11(t *testing.T) {
	// cursed
	rawData, err := GetRawData("b9c4c69c0160c4a30a438ccd976d0c96a0a8296812121931c46ee370af729de2", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}

	for _, field := range fields {
		assert.True(t, field.IsCursed)
	}
}

func TestParser_ord12(t *testing.T) {
	// empty envelope
	rawData, err := GetRawData("ce7291326d22e1f2dffc12c118dfa464ad55d07ff9ada5a91b8ec9d9301a6f05", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, _ := ParseInscription(rawData)

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

func TestParser_ord13(t *testing.T) {
	// cursed
	rawData, err := GetRawData("c52311d91d666ddf9e27caffc84f9a3cd967b58aab97e44e828c90842d53dd79", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}

	for _, field := range fields {
		assert.True(t, field.IsCursed)
	}
}

func TestParser_ord14(t *testing.T) {
	// cursed
	rawData, err := GetRawData("5f7f8779ead5d786f13310b5e9239ee3cc1f4697ccd368ea5b3a37f785fca200", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}

	for _, field := range fields {
		assert.True(t, field.IsCursed)
	}
}

func TestParser_ord15(t *testing.T) {
	// invalid. why?
	rawData, err := GetRawData("c769750df54ee38fe2bae876dbf1632c779c3af780958a19cee1ca0497c78e80", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
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
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 0 {
		assert.True(t, false)
	}
}

func TestParser_ord17(t *testing.T) {

	rawData, err := GetRawData("f8fc655ffe139d9952e673c53b7d15cb4b82de5ef036c7fc1211262bbd29bec8", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
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
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
	for _, field := range fields {
		assert.True(t, field.IsCursed)
	}
}

func TestParser_ord19(t *testing.T) {
	// reinscription
	rawData, err := GetRawData("4c6479280e27c6b99c8de404921b6c813fc323df77e06c7f8f6aa08e6a81f148", "testnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 2 {
		assert.True(t, false)
	}

	
	assert.True(t, !fields[0].IsCursed)
	assert.True(t, fields[1].IsCursed)
}

func TestParser_ord20(t *testing.T) {
	// input 0, output 0
	rawData, err := GetRawData("c1e0db6368a43f5589352ed44aa1ff9af33410e4a9fd9be0f6ac42d9e4117151", "mainnet")
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
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
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}
	fields, _, err := ParseInscription(rawData)
	if err != nil {
		fmt.Printf("%v\n", err)
		assert.True(t, false)
	}

	if len(fields) != 1 {
		assert.True(t, false)
	}
}

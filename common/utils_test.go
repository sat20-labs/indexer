package common

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"testing"
	"time"

	"github.com/vmihailenco/msgpack/v5"

	"google.golang.org/protobuf/proto"
)

func TestBinarySearch(t *testing.T) {

	utxos := make([]*UtxoIdInDB, 0)
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:10, Value:1})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:20, Value:2})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:15, Value:3})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:25, Value:4})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:5, Value:5})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:35, Value:6})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:24, Value:7})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:26, Value:8})
	printUtxos(utxos)

	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:25, Value:9})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:5, Value:11})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:35, Value:10})

	printUtxos(utxos)

	utxos = DeleteUtxo(utxos, 5)
	utxos = DeleteUtxo(utxos, 6)
	utxos = DeleteUtxo(utxos, 35)
	utxos = DeleteUtxo(utxos, 25)

	printUtxos(utxos)

}

func printUtxos(utxos []*UtxoIdInDB) {
	for _, utxo := range utxos {
		fmt.Printf("%d-%d\n", utxo.UtxoId, utxo.Value)
	}
	fmt.Printf("\n")
}

func TestSliceCopy(t *testing.T) {

	utxos := make([]*UtxoIdInDB, 0)
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:10, Value:1})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:20, Value:2})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:15, Value:3})
	utxos = InsertUtxo(utxos, &UtxoIdInDB{UtxoId:25, Value:4})
	destination := make([]*UtxoIdInDB, len(utxos))

	copy(destination, utxos)

	fmt.Printf("%v\n", utxos)
	fmt.Printf("%v\n", destination)

	utxos = DeleteUtxo(utxos, 20)
	destination = InsertUtxo(destination, &UtxoIdInDB{UtxoId:35, Value:5})

	fmt.Printf("%v\n", utxos)
	fmt.Printf("%v\n", destination)

	printUtxos(utxos)
	printUtxos(destination)

}

type UtxoValueInDBv2 struct {
	UtxoId      uint64
	AddressType uint32
	AddressId   []uint64
	Ordinals    []Range
}

func TestDecode(t *testing.T) {
	rngs := make([]*Range, 0)
	for i := int64(0); i < 10; i++ {
		rngs = append(rngs, &Range{Start: i, Size: i})
	}

	rngs2 := make([]Range, 0)
	for i := int64(0); i < 10; i++ {
		rngs2 = append(rngs2, Range{Start: i, Size: i})
	}

	value1 := UtxoValueInDB{UtxoId: 1, Value: 1000, AddressId: 0}
	value2 := UtxoValueInDBv2{UtxoId: 3, AddressId: []uint64{4}, Ordinals: rngs2}

	fmt.Printf("gob...\n")
	start := time.Now()
	var encodeBytes []byte
	for i := 0; i < 1000; i++ {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(&value1); err != nil {
			t.Fatal(err)
		}
		encodeBytes = buf.Bytes()
	}
	fmt.Printf("encode time: %v\n", time.Since(start)) // 52ms
	fmt.Printf("%d\n", len(encodeBytes))               // 709

	start = time.Now()
	result1 := &UtxoValueInDB{}
	for i := 0; i < 1000; i++ {
		buf := bytes.NewBuffer(encodeBytes)
		dec := gob.NewDecoder(buf)
		err := dec.Decode(result1)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("decode time: %v\n", time.Since(start)) // 84ms

	fmt.Printf("\ngob...improve struct\n")
	start = time.Now()
	for i := 0; i < 1000; i++ {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(&value2); err != nil {
			t.Fatal(err)
		}
		encodeBytes = buf.Bytes()
	}
	fmt.Printf("encode time: %v\n", time.Since(start)) // 53ms
	fmt.Printf("%d\n", len(encodeBytes))               // 709

	start = time.Now()
	result2 := &UtxoValueInDBv2{}
	for i := 0; i < 1000; i++ {
		buf := bytes.NewBuffer(encodeBytes)
		dec := gob.NewDecoder(buf)
		err := dec.Decode(result2)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("decode time: %v\n", time.Since(start)) // 86ms

	fmt.Printf("\nproto buffer...\n")
	start = time.Now()
	var err error
	for i := 0; i < 1000; i++ {
		encodeBytes, err = proto.Marshal(&value1)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("encode time: %v\n", time.Since(start)) // 23ms
	fmt.Printf("%d\n", len(encodeBytes))               // 600

	start = time.Now()
	for i := 0; i < 1000; i++ {
		err = proto.Unmarshal(encodeBytes, result1)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("decode time: %v\n", time.Since(start)) // 32ms
	//fmt.Printf("%v\n", result1)

	/////////////////////////

	fmt.Printf("\nmsgpack...\n")
	start = time.Now()
	for i := 0; i < 1000; i++ {
		encodeBytes, err = Serialize(&value2)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("encode time: %v\n", time.Since(start)) // 73ms
	fmt.Printf("%d\n", len(encodeBytes))               // 3048

	start = time.Now()
	for i := 0; i < 1000; i++ {
		result2, err = Deserialize(encodeBytes)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("decode time: %v\n", time.Since(start)) // 94ms
	fmt.Printf("utxo %d\n", result2.UtxoId)
	//////////////////////////////
}

func TestDecode2(t *testing.T) {
	rngs := make([]*Range, 0)
	for i := int64(0); i < 10; i++ {
		rngs = append(rngs, &Range{Start: i, Size: i})
	}

	value1 := UtxoValueInDB{UtxoId: 1, Value: 1000, AddressId: 0}

	//fmt.Printf("%v\n", value1)
	start := time.Now()
	var encodeBytes []byte
	var err error
	for i := 0; i < 1000; i++ {
		encodeBytes, err = proto.Marshal(&value1)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("encode time: %vs\n", time.Since(start).Seconds()) // 3.1ms
	fmt.Printf("%d\n", len(encodeBytes))                          // 82

	start = time.Now()
	result1 := &UtxoValueInDB{}
	for i := 0; i < 1000; i++ {
		err = proto.Unmarshal(encodeBytes, result1)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("decode time: %vs\n", time.Since(start).Seconds()) // 4.6ms
	//fmt.Printf("%v\n", result1)

	start = time.Now()
	for i := 0; i < 1000; i++ {
		var buf bytes.Buffer
		enc := gob.NewEncoder(&buf)
		if err := enc.Encode(&value1); err != nil {
			t.Fatal(err)
		}
		encodeBytes = buf.Bytes()
	}
	fmt.Printf("encode time: %vs\n", time.Since(start).Seconds()) // 22ms
	fmt.Printf("%d\n", len(encodeBytes))                          // 226

	start = time.Now()
	result2 := &UtxoValueInDB{}
	for i := 0; i < 1000; i++ {
		buf := bytes.NewBuffer(encodeBytes)
		dec := gob.NewDecoder(buf)
		err := dec.Decode(result2)
		if err != nil {
			t.Fatal(err)
		}
	}
	fmt.Printf("decode time: %vs\n", time.Since(start).Seconds()) // 55ms
}

// 序列化函数
func Serialize(obj *UtxoValueInDBv2) ([]byte, error) {
	data, err := msgpack.Marshal(obj)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// 反序列化函数
func Deserialize(data []byte) (*UtxoValueInDBv2, error) {
	var obj UtxoValueInDBv2
	err := msgpack.Unmarshal(data, &obj)
	if err != nil {
		return nil, err
	}
	return &obj, nil
}

func TestDecode3(t *testing.T) {

	value := int64(1)

	bytes := Uint64ToBytes(uint64(value))
	fmt.Printf("%v", bytes)

	value2 := int64(BytesToUint64(bytes))
	fmt.Printf("%d", value2)

	utxoid := ToUtxoId(0x7ffffffe, 0xeffe, 0x1effe)
	fmt.Printf("%x\n", utxoid)
	v1, v2, v3 := FromUtxoId(utxoid)
	fmt.Printf("%x %x %x\n", v1, v2, v3)
}

func TestGenerateSeed2(t *testing.T) {

	range1 := []*Range{{Start: 1234567890123456, Size: 1000}}

	seed := GenerateSeed2(range1)
	fmt.Printf("%s\n", seed) // 780e6f5c7a7e95b

	range2 := []*Range{{Start: 1234567890123456, Size: 1000},
		{Start: 100000000, Size: 1}}

	seed = GenerateSeed2(range2)
	fmt.Printf("%s\n", seed) // 8869c43df2f8a6d7

}


func TestGenerateSeed(t *testing.T) {

	assetVector := []*AssetInfo{
		{
			Name: AssetName{
				Protocol: "ord",
				Type: "o",
				Ticker: "12345678",
			},
			Amount: *NewDecimal(1, 0),
			BindingSat: 1,
		},
		{
			Name: AssetName{
				Protocol: "ordx",
				Type: "f",
				Ticker: "rarepizza",
			},
			Amount: *NewDecimal(1000, 0),
			BindingSat: 1,
		},
	}

	seed := GenerateSeed(assetVector)
	fmt.Printf("%s\n", seed) // 482f0fd575b0d16

	assetVector[0].Name.Ticker = "12345679"
	seed = GenerateSeed(assetVector)
	fmt.Printf("%s\n", seed) // d5d7f609a69c9bae
}

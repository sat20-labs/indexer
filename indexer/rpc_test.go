package indexer

import (
	"fmt"
	"math"
	"strconv"
	"testing"
	"time"

	"github.com/sat20-labs/ordx/common"
	"github.com/sat20-labs/ordx/share/bitcoin_rpc"
	"github.com/stretchr/testify/assert"
)

func TestGetBatchUnspendTxOutput(t *testing.T) {
	err := bitcoin_rpc.InitBitconRpc(
		"192.168.1.102",
		8332,
		"jacky",
		"_RZekaGRgKQJSIOYi6vq0_CkJtjoCootamy81J2cDn0",
		false,
	)
	if err != nil {
		common.Log.Fatalln(err)
	}

	utxoListTemplate := []string{
		"90808ef396f76e433ec2fd40890e6c6de8a331688b1feb941c10b3fdbe0052a4:0",
		"90808ef396f76e433ec2fd40890e6c6de8a331688b1feb941c10b3fdbe0052a4:3",
		"9ef03ffde4b91744798cde0eb1f8ffe6c0e7607220dd9a87fb522578248cb081:0",
		"dd31b57963d7a6289af83d47a59c78d437671158e6613bf1e5bc0f2241e8d0e1:0",
		"84bdb0b407acde53c45eae02b86485c5a83de26af711aee3a4d421dea93c5d4a:0",
		"84bdb0b407acde53c45eae02b86485c5a83de26af711aee3a4d421dea93c5d4a:1",
		"84bdb0b407acde53c45eae02b86485c5a83de26af711aee3a4d421dea93c5d4a:2",
		"84bdb0b407acde53c45eae02b86485c5a83de26af711aee3a4d421dea93c5d4a:3",
		"84bdb0b407acde53c45eae02b86485c5a83de26af711aee3a4d421dea93c5d4a:4",
		"84bdb0b407acde53c45eae02b86485c5a83de26af711aee3a4d421dea93c5d4a:5",
	}
	utxoList := []string{}
	for i := 0; i < 100; i++ {
		utxoList = append(utxoList, utxoListTemplate...)
	}
	start1Time := time.Now()
	unspendTxOutputListChan := bitcoin_rpc.GetBatchUnspendTxOutput(utxoList, true, 100, nil)
	unspendTxOutputList := <-unspendTxOutputListChan
	endTime := time.Now()
	t.Logf("GetBatchUnspendTxOutput count: %d, elapsed: %v", len(unspendTxOutputList), endTime.Sub(start1Time))
	for utxo, unspendTxOutput := range unspendTxOutputList {
		if unspendTxOutput.BestBlock == "" {
			t.Logf("utxo:%s isn't exist in mempool", utxo)
		}
	}
	assert.NotEmpty(t, unspendTxOutputList)
}

func TestParsePercentage(t *testing.T) {

	fstr := "0.999"
	f, _ := strconv.ParseFloat(fstr, 32)
	fmt.Printf("%s -> %f\n", fstr, f)
	r := (math.Floor(f))
	fmt.Printf("%f -> %f\n", f, r)
	r = (math.Round(f))
	fmt.Printf("%f -> %f\n", f, r)
	r = (math.Trunc(f))
	fmt.Printf("%f -> %f\n", f, r)

	str := "0.991"
	p, err := getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.999"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.9999"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.99"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "1.99"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.90"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "0.990"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "00.990"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "0.09%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.1%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "0.99%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "1.99%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "90%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "990%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
	} else {
		fmt.Printf("%s -> %d\n", str, p)
		t.Fatal()
	}

	str = "99.0%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}

	str = "9.0%"
	p, err = getPercentage(str)
	if err != nil {
		fmt.Printf("%v\n", err)
		t.Fatal()
	} else {
		fmt.Printf("%s -> %d\n", str, p)
	}
}

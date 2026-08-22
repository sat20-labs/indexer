package common

import (
	"encoding/hex"
	"encoding/json"
	"reflect"
	"testing"
)

func TestCBORJSONRoundTrip(t *testing.T) {
	input := []byte(`{"name":"Alice","age":"30"}`)
	encoded, err := Json2cbor(input)
	if err != nil {
		t.Fatalf("Json2cbor: %v", err)
	}
	decoded, err := Cbor2json(encoded)
	if err != nil {
		t.Fatalf("Cbor2json: %v", err)
	}
	var got, want map[string]any
	if err := json.Unmarshal(decoded, &got); err != nil {
		t.Fatalf("decode result: %v", err)
	}
	if err := json.Unmarshal(input, &want); err != nil {
		t.Fatalf("decode input: %v", err)
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("round trip=%v, want %v", got, want)
	}
}

func TestParseInscriptionID(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{
			raw:  "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100",
			want: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1fi0",
		},
		{
			raw:  "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a09080706050403020100ff",
			want: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1fi255",
		},
		{
			raw:  "1f1e1d1c1b1a191817161514131211100f0e0d0c0b0a090807060504030201000001",
			want: "000102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1fi256",
		},
	}
	for _, test := range tests {
		raw, err := hex.DecodeString(test.raw)
		if err != nil {
			t.Fatal(err)
		}
		if got := ParseInscriptionId(raw); got != test.want {
			t.Fatalf("ParseInscriptionId=%s, want %s", got, test.want)
		}
	}
}

func TestOrdxUpdateJSON(t *testing.T) {
	update := OrdxUpdateContentV1{
		OrdxBaseContent: OrdxBaseContent{P: "ordx", Op: "update"},
		Name:            "12345",
		KVs:             []string{"key1=value1", "key2=value2"},
	}
	encoded, err := json.Marshal(update)
	if err != nil {
		t.Fatal(err)
	}
	var decoded OrdxUpdateContentV1
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded.Name != update.Name || !reflect.DeepEqual(decoded.KVs, update.KVs) {
		t.Fatalf("decoded=%+v, want %+v", decoded, update)
	}
}

func TestNamePreprocessingAndValidation(t *testing.T) {
	if got := PreprocessName("  \n Iou \n\n"); got != "Iou" {
		t.Fatalf("PreprocessName=%q, want Iou", got)
	}
	invalid := []string{"12b1!", "12b 1", "12.b1"}
	for _, name := range invalid {
		if IsValidSat20Name(name) {
			t.Fatalf("%q should be invalid", name)
		}
	}
	valid := []string{"12b11"}
	for _, name := range valid {
		if !IsValidSat20Name(name) {
			t.Fatalf("%q should be valid", name)
		}
	}
}

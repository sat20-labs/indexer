package common

import (
	"encoding/json"
	"math/big"
	"strconv"
	"strings"
	"testing"
)

func TestDecimalJSONPreservesWireBytesAndValidation(t *testing.T) {
	for _, test := range []struct {
		name      string
		precision int
		value     string
	}{
		{"zero", 0, "0"},
		{"positive", 8, "123456789"},
		{"negative", 18, "-123456789"},
		{"max_precision", MAX_PRECISION, "1"},
		{"large_integer", 0, strings.Repeat("9", MaxProtocolDecimalIntegerDigits)},
	} {
		t.Run(test.name, func(t *testing.T) {
			value, ok := new(big.Int).SetString(test.value, 10)
			if !ok {
				t.Fatal("invalid test value")
			}
			d := &Decimal{Precision: test.precision, Value: value}
			// The existing wire representation uses sorted map keys, integer
			// Precision, and the unscaled integer as a JSON string.
			want, err := json.Marshal(map[string]interface{}{"Precision": test.precision, "Value": test.value})
			if err != nil {
				t.Fatal(err)
			}
			got, err := d.MarshalJSON()
			if err != nil || string(got) != string(want) {
				t.Fatalf("direct encoding: got %s, %v; want %s", got, err, want)
			}
			got, err = json.Marshal(d)
			if err != nil || string(got) != string(want) {
				t.Fatalf("JSON encoding: got %s, %v; want %s", got, err, want)
			}
			var decoded Decimal
			if err := json.Unmarshal(got, &decoded); err != nil {
				t.Fatal(err)
			}
			if decoded.Precision != d.Precision || decoded.Value.Cmp(d.Value) != 0 {
				t.Fatalf("roundtrip mismatch: %+v", decoded)
			}
			nested, err := json.Marshal(struct {
				Amount *Decimal `json:"amount"`
			}{d})
			if err != nil || string(nested) != `{"amount":`+string(want)+`}` {
				t.Fatalf("nested encoding: %s, %v", nested, err)
			}
		})
	}
	for _, test := range []struct {
		name  string
		value *Decimal
	}{
		{"nil_receiver", nil},
		{"nil_integer", &Decimal{Precision: 0}},
		{"negative_precision", &Decimal{Precision: -1, Value: big.NewInt(1)}},
		{"overflow_precision", &Decimal{Precision: MAX_PRECISION + 1, Value: big.NewInt(1)}},
	} {
		t.Run(test.name, func(t *testing.T) {
			wantErr := test.value.Validate()
			got, err := test.value.MarshalJSON()
			if wantErr == nil || err == nil || err.Error() != wantErr.Error() || got != nil {
				t.Fatalf("validation changed: bytes=%s, err=%v; want %v", got, err, wantErr)
			}
		})
	}
	var nilAmount *Decimal
	got, err := json.Marshal(nilAmount)
	if err != nil || string(got) != "null" {
		t.Fatalf("nil JSON encoding: %s, %v", got, err)
	}
}

func TestDecimalJSONDirectEncodingAllPrecisions(t *testing.T) {
	for precision := 0; precision <= MAX_PRECISION; precision++ {
		t.Run(strconv.Itoa(precision), func(t *testing.T) {
			for _, text := range []string{"0", "1", "-1", "18446744073709551616", "-18446744073709551616", strings.Repeat("9", 512)} {
				value, ok := new(big.Int).SetString(text, 10)
				if !ok {
					t.Fatal("invalid test value")
				}
				amount := &Decimal{Precision: precision, Value: value}
				want, err := json.Marshal(struct {
					Precision int    `json:"Precision"`
					Value     string `json:"Value"`
				}{Precision: precision, Value: value.String()})
				if err != nil {
					t.Fatal(err)
				}
				got, err := amount.MarshalJSON()
				if err != nil || string(got) != string(want) {
					t.Fatalf("direct encoding of %s: got %s, %v; want %s", text, got, err, want)
				}
				got, err = json.Marshal(amount)
				if err != nil || string(got) != string(want) {
					t.Fatalf("nested encoding of %s: got %s, %v; want %s", text, got, err, want)
				}
			}
		})
	}
}

func BenchmarkDecimalJSONEncoding(b *testing.B) {
	for _, precision := range []int{0, 8, MAX_PRECISION} {
		amount := &Decimal{Precision: precision, Value: big.NewInt(-123456789)}
		b.Run(strconv.Itoa(precision)+"/direct", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if _, err := amount.MarshalJSON(); err != nil {
					b.Fatal(err)
				}
			}
		})
		b.Run(strconv.Itoa(precision)+"/previous", func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				if err := amount.Validate(); err != nil {
					b.Fatal(err)
				}
				if _, err := json.Marshal(struct {
					Precision int    `json:"Precision"`
					Value     string `json:"Value"`
				}{Precision: amount.Precision, Value: amount.Value.String()}); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

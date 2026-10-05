package db_test

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"testing"

	"github.com/sat20-labs/indexer/common"
	"google.golang.org/protobuf/proto"
)

type logicalUtxoValue struct {
	UtxoID    uint64
	Value     int64
	AddressID uint64
}

var codecSample = logicalUtxoValue{
	UtxoID:    4_294_967_321_001,
	Value:     5_000_000_000,
	AddressID: 108_033_293,
}

var (
	codecBytesSink []byte
	codecValueSink logicalUtxoValue
)

func encodeProtoValue(value logicalUtxoValue) ([]byte, error) {
	return proto.Marshal(&common.UtxoValueInDB{
		UtxoId:    value.UtxoID,
		Value:     value.Value,
		AddressId: value.AddressID,
	})
}

func decodeProtoValue(data []byte) (logicalUtxoValue, error) {
	var value common.UtxoValueInDB
	if err := proto.Unmarshal(data, &value); err != nil {
		return logicalUtxoValue{}, err
	}
	return logicalUtxoValue{
		UtxoID:    value.UtxoId,
		Value:     value.Value,
		AddressID: value.AddressId,
	}, nil
}

func encodeGobValue(value logicalUtxoValue) ([]byte, error) {
	var buffer bytes.Buffer
	if err := gob.NewEncoder(&buffer).Encode(value); err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func decodeGobValue(data []byte) (logicalUtxoValue, error) {
	var value logicalUtxoValue
	err := gob.NewDecoder(bytes.NewReader(data)).Decode(&value)
	return value, err
}

func encodeRawValue(value logicalUtxoValue) []byte {
	result := make([]byte, 24)
	binary.BigEndian.PutUint64(result[0:8], value.UtxoID)
	binary.BigEndian.PutUint64(result[8:16], uint64(value.Value))
	binary.BigEndian.PutUint64(result[16:24], value.AddressID)
	return result
}

func decodeRawValue(data []byte) (logicalUtxoValue, error) {
	if len(data) != 24 {
		return logicalUtxoValue{}, fmt.Errorf("raw value length %d, want 24", len(data))
	}
	return logicalUtxoValue{
		UtxoID:    binary.BigEndian.Uint64(data[0:8]),
		Value:     int64(binary.BigEndian.Uint64(data[8:16])),
		AddressID: binary.BigEndian.Uint64(data[16:24]),
	}, nil
}

// encodeGenericTLV intentionally models a general-purpose TLV rather than a
// Protobuf-like field codec. Each field carries tag, type, length and payload.
func encodeGenericTLV(value logicalUtxoValue) []byte {
	result := make([]byte, 0, 33)
	appendUint64 := func(tag byte, number uint64) {
		result = append(result, tag, 1, 8) // tag, uint64 type, payload length
		var encoded [8]byte
		binary.BigEndian.PutUint64(encoded[:], number)
		result = append(result, encoded[:]...)
	}
	appendUint64(1, value.UtxoID)
	appendUint64(2, uint64(value.Value))
	appendUint64(3, value.AddressID)
	return result
}

func decodeGenericTLV(data []byte) (logicalUtxoValue, error) {
	var result logicalUtxoValue
	for len(data) > 0 {
		if len(data) < 3 {
			return logicalUtxoValue{}, fmt.Errorf("truncated TLV header")
		}
		tag, typ, length := data[0], data[1], int(data[2])
		data = data[3:]
		if typ != 1 || length != 8 || len(data) < length {
			return logicalUtxoValue{}, fmt.Errorf("invalid TLV field tag=%d type=%d length=%d", tag, typ, length)
		}
		value := binary.BigEndian.Uint64(data[:length])
		data = data[length:]
		switch tag {
		case 1:
			result.UtxoID = value
		case 2:
			result.Value = int64(value)
		case 3:
			result.AddressID = value
		}
	}
	return result, nil
}

func TestRepresentativeCodecSizesAndRoundTrips(t *testing.T) {
	encoders := []struct {
		name   string
		encode func(logicalUtxoValue) ([]byte, error)
		decode func([]byte) (logicalUtxoValue, error)
	}{
		{"protobuf", encodeProtoValue, decodeProtoValue},
		{"gob", encodeGobValue, decodeGobValue},
		{"raw-fixed", func(v logicalUtxoValue) ([]byte, error) { return encodeRawValue(v), nil }, decodeRawValue},
		{"generic-tlv", func(v logicalUtxoValue) ([]byte, error) { return encodeGenericTLV(v), nil }, decodeGenericTLV},
	}

	sizes := make(map[string]int, len(encoders))
	for _, codec := range encoders {
		encoded, err := codec.encode(codecSample)
		if err != nil {
			t.Fatalf("%s encode: %v", codec.name, err)
		}
		decoded, err := codec.decode(encoded)
		if err != nil {
			t.Fatalf("%s decode: %v", codec.name, err)
		}
		if decoded != codecSample {
			t.Fatalf("%s round trip=%+v, want %+v", codec.name, decoded, codecSample)
		}
		sizes[codec.name] = len(encoded)
		t.Logf("%s encoded size: %d bytes", codec.name, len(encoded))
	}
	if sizes["protobuf"] >= sizes["gob"] {
		t.Fatalf("protobuf size %d should be smaller than per-value Gob size %d", sizes["protobuf"], sizes["gob"])
	}
	if sizes["raw-fixed"] >= sizes["gob"] {
		t.Fatalf("raw size %d should be smaller than per-value Gob size %d", sizes["raw-fixed"], sizes["gob"])
	}
}

func BenchmarkRepresentativeCodecEncode(b *testing.B) {
	benchmarks := []struct {
		name   string
		encode func(logicalUtxoValue) ([]byte, error)
	}{
		{"protobuf", encodeProtoValue},
		{"gob", encodeGobValue},
		{"raw-fixed", func(v logicalUtxoValue) ([]byte, error) { return encodeRawValue(v), nil }},
		{"generic-tlv", func(v logicalUtxoValue) ([]byte, error) { return encodeGenericTLV(v), nil }},
	}
	for _, benchmark := range benchmarks {
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				encoded, err := benchmark.encode(codecSample)
				if err != nil {
					b.Fatal(err)
				}
				codecBytesSink = encoded
			}
		})
	}
}

func BenchmarkRepresentativeCodecDecode(b *testing.B) {
	benchmarks := []struct {
		name   string
		encode func(logicalUtxoValue) ([]byte, error)
		decode func([]byte) (logicalUtxoValue, error)
	}{
		{"protobuf", encodeProtoValue, decodeProtoValue},
		{"gob", encodeGobValue, decodeGobValue},
		{"raw-fixed", func(v logicalUtxoValue) ([]byte, error) { return encodeRawValue(v), nil }, decodeRawValue},
		{"generic-tlv", func(v logicalUtxoValue) ([]byte, error) { return encodeGenericTLV(v), nil }, decodeGenericTLV},
	}
	for _, benchmark := range benchmarks {
		encoded, err := benchmark.encode(codecSample)
		if err != nil {
			b.Fatal(err)
		}
		b.Run(benchmark.name, func(b *testing.B) {
			b.ReportAllocs()
			b.SetBytes(int64(len(encoded)))
			for i := 0; i < b.N; i++ {
				decoded, err := benchmark.decode(encoded)
				if err != nil {
					b.Fatal(err)
				}
				codecValueSink = decoded
			}
		})
	}
}

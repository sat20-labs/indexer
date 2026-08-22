# Indexer persistent-value encoding policy

## Decision

Indexer does **not** force every key and value through one physical encoding.
The persistent format is selected by data shape:

| Data shape | Encoding |
|---|---|
| Ordered identifiers embedded in keys | fixed-width, big-endian binary |
| Small scalar values with a fixed schema | raw fixed-width binary or varint |
| Structured records that evolve over time | Protocol Buffers |
| Protocol-owned payloads | their original protocol format, such as CBOR, JSON, or opaque bytes |
| Legacy export/backup streams | Gob may remain until the export format is separately revised |

New online tables must not introduce Gob values. Existing Gob tables are migrated
when their module schema is rebuilt; a cross-repository, one-shot conversion is
intentionally avoided because it would combine unrelated protocol migrations in
one change.

A generic custom TLV format is not adopted. Protobuf is already a tagged,
wire-typed encoding, provides schema tooling and unknown-field compatibility,
and was materially smaller than the representative generic TLV in the local
benchmark.

## Reproducible benchmark

Run:

```bash
go test ./indexer/db \
  -run TestRepresentativeCodecSizesAndRoundTrips \
  -bench 'BenchmarkRepresentativeCodec(Encode|Decode)$' \
  -benchmem -count=3
```

Reference run on 2026-08-22:

```text
goos: darwin
goarch: amd64
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz
```

Representative logical record:

```text
utxo_id    = 4,294,967,321,001
value      = 5,000,000,000
address_id = 108,033,293
```

### Encoded size

| Encoding | Bytes |
|---|---:|
| Protobuf | 18 |
| Gob, new encoder per value | 90 |
| Raw fixed-width binary | 24 |
| Generic tag/type/length/value | 33 |

### Encode

| Encoding | Time/op | Bytes allocated/op | Allocations/op |
|---|---:|---:|---:|
| Protobuf | 172–175 ns | 88 | 2 |
| Gob | 2.23–2.31 µs | 1,528 | 20 |
| Raw fixed-width binary | 26.4–26.7 ns | 24 | 1 |
| Generic TLV | 33.8–34.3 ns | 48 | 1 |

### Decode

| Encoding | Time/op | Bytes allocated/op | Allocations/op |
|---|---:|---:|---:|
| Protobuf | 161–165 ns | 64 | 1 |
| Gob | 15.9–16.6 µs | 7,136 | 159 |
| Raw fixed-width binary | 3.19–3.28 ns | 0 | 0 |
| Generic TLV | 9.61–10.65 ns | 0 | 0 |

The exact timings are machine-dependent. The relative result is the relevant
one: the project's current per-value Gob usage is substantially slower and
larger than Protobuf, while raw binary is appropriate for truly fixed scalar
records. Generic TLV was fast in this deliberately simple benchmark, but it was
larger than Protobuf and would require a second hand-written schema system.

## Schema rules

1. Keys that must sort numerically encode integers as fixed-width big-endian
   values. Decimal and hexadecimal strings are not used for new hot indexes.
2. A table owns exactly one codec. Call sites do not choose between `SetDB` and
   `SetDBWithProto3` ad hoc.
3. Protobuf messages should use packed repeated numeric fields and `sint64` for
   frequently negative values.
4. Large logical collections are split into multiple keys rather than stored as
   one very large message.
5. A module format change increments its DB version. Historical index databases
   are rebuilt rather than carrying permanent dual-read compatibility.
6. Benchmarks must include encoded bytes, ns/op, B/op, allocs/op, Badger point
   reads, prefix scans, batch writes, peak RSS, and post-GC disk usage before a
   custom codec is introduced into production.

## Status in this branch

- Base address UTXO amounts use compact raw 8-byte values under binary
  `address_id/utxo_id` keys.
- Base address metadata and NFT state use Protobuf.
- Runes continues to use Protobuf typed tables.
- The new Exotic ticker-holder aggregate uses compact raw 8-byte values.
- Atom and remaining legacy protocol tables still contain Gob values. Their
  in-memory full-state problem is removed in this branch; their structured
  value migration should be performed module-by-module with DB-version bumps
  and full historical comparison tests.

# Indexer persistent encoding policy

## Decision

Indexer does not force every key and value through one physical encoding. The production rule is:

- Structured persistent values: Protocol Buffers.
- Small fixed-schema scalar values: compact binary (fixed-width big-endian or varint where appropriate).
- Ordered integer components in DB keys: fixed-width big-endian binary.
- Protocol-owned payloads: retain the protocol's native encoding.
- Gob: no new online index tables; keep only where legacy compatibility or offline backup/export still requires it, then migrate module-by-module when the DB version is rebuilt.

A new generic TLV format is not adopted. Protobuf already provides a tagged wire format, schema tooling and compatibility semantics. A custom binary codec is justified only for very small, stable, hot records where the schema is fixed and benchmarked end-to-end.

## Local benchmark

Environment:

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

Encoded sizes:

| Encoding | Bytes |
|---|---:|
| Protobuf | 18 |
| Gob (new encoder per value) | 90 |
| Raw fixed-width | 24 |
| Generic TLV sample | 33 |

Representative encode results across three runs:

| Encoding | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| Protobuf | 286-303 | 88 | 2 |
| Gob | 3744-3881 | 1528 | 20 |
| Raw fixed-width | 40-42 | 24 | 1 |
| Generic TLV sample | 50-51 | 48 | 1 |

Representative decode results across three runs:

| Encoding | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| Protobuf | 262-330 | 64 | 1 |
| Gob | 25,916-27,292 | 7136 | 159 |
| Raw fixed-width | 5.35-5.53 | 0 | 0 |
| Generic TLV sample | 16.24-16.37 | 0 | 0 |

The exact timings are machine-dependent. The important conclusions are stable for the current code path: per-value Gob is much larger and slower than Protobuf, while raw binary is appropriate for simple fixed records. The generic TLV sample is fast but larger than Protobuf and would introduce another hand-maintained schema system.

## Schema rules

1. Numeric key components that require ordered scans use fixed-width big-endian binary.
2. Large logical collections are split into prefix-key records instead of one growing value.
3. Each table has one declared codec; call sites should not choose Gob vs Protobuf ad hoc.
4. Protobuf schemas should use `sint64` for frequently negative integers and packed repeated numeric fields where applicable.
5. Incompatible module schema changes bump the module DB version and require an index rebuild rather than permanent dual-read compatibility.
6. A new custom codec requires both micro-benchmark and Badger end-to-end evidence: write throughput, point reads, prefix scans, allocations/RSS, disk size after GC, and maintenance cost.

## Current branch status

- Base address UTXOs: compact per-UTXO key/value records; address metadata remains separate.
- NFT: primary structured records use Protobuf; the old Gob BuckStore has been removed.
- Runes: Protobuf typed tables; read cache is separated from pending writes.
- Exotic: growing indexes are DB-backed/pending rather than fully resident; remaining legacy Gob structures should migrate module-by-module.
- Atom: growing state is queried from DB on demand while block processing remains intentionally disabled; legacy Gob structures can be migrated before Atomicals is enabled.
- FT: unchanged in this optimization round.

Reproduce with:

```bash
go test ./indexer/db -run TestRepresentativeCodecSizesAndRoundTrips -v -count=1
go test ./indexer/db -run '^$' -bench 'BenchmarkRepresentativeCodec(Encode|Decode)$' -benchmem -count=3
```

# Indexer persistent-value encoding policy

This document records the encoding policy. The measured results and complete benchmark output are in:

- `storage-encoding.md`
- `storage-encoding-size-results.txt`
- `storage-encoding-benchmark.txt`

## Policy

Indexer does **not** force every key and value through one physical encoding. Encoding is selected by data shape:

| Data shape | Encoding |
|---|---|
| Ordered identifiers embedded in keys | fixed-width big-endian binary |
| Small scalar values with a fixed schema | raw fixed-width binary or varint |
| Structured records that evolve over time | Protocol Buffers |
| Protocol-owned payloads | native protocol encoding (CBOR, JSON, opaque bytes, etc.) |
| Legacy backup/export streams | Gob may remain until separately revised |

New online index tables must not introduce Gob values. Existing Gob tables are migrated module-by-module when their storage schema is rebuilt.

A generic custom TLV format is not adopted. Protobuf already provides a tagged wire format, schema tooling and compatibility semantics. A hand-written binary codec is reserved for small, fixed and heavily accessed records where an end-to-end Badger benchmark justifies it.

## Verified local benchmark — 2026-08-22

Environment:

```text
goos: darwin
goarch: amd64
cpu: Intel(R) Core(TM) i7-1068NG7 CPU @ 2.30GHz
```

Representative value encoded sizes:

| Encoding | Bytes |
|---|---:|
| Protobuf | 18 |
| Gob (new encoder per value) | 90 |
| Raw fixed-width | 24 |
| Generic TLV sample | 33 |

Measured encode ranges across three runs:

| Encoding | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| Protobuf | 286-303 | 88 | 2 |
| Gob | 3744-3881 | 1528 | 20 |
| Raw fixed-width | 40-42 | 24 | 1 |
| Generic TLV sample | 50-51 | 48 | 1 |

Measured decode ranges:

| Encoding | ns/op | B/op | allocs/op |
|---|---:|---:|---:|
| Protobuf | 262-330 | 64 | 1 |
| Gob | 25,916-27,292 | 7136 | 159 |
| Raw fixed-width | 5.35-5.53 | 0 | 0 |
| Generic TLV sample | 16.24-16.37 | 0 | 0 |

The previous draft values in this file were not produced by a successfully executed final benchmark and have been replaced by the measurements above.

## Schema rules

1. Numeric key components that require ordered scans use fixed-width big-endian binary.
2. A table owns one persistent codec; callers do not choose Gob or Protobuf ad hoc.
3. Large logical collections are split into prefix-key records instead of one ever-growing value.
4. Protobuf should use `sint64` for frequently negative values and packed repeated numeric fields where appropriate.
5. Incompatible module schema changes bump that module's DB version and are handled by rebuilding the index, not permanent dual-read compatibility.
6. A custom codec requires both micro-benchmark and Badger end-to-end evidence: write throughput, point reads, prefix scans, allocations/RSS and disk size after GC.

## Current branch

- Base address UTXOs use compact per-UTXO records rather than one full address UTXO value.
- NFT primary structured records use Protobuf; the redundant Gob BuckStore has been removed.
- Runes keeps Protobuf typed tables and separates pending writes from bounded read cache.
- Exotic no longer keeps the durable ticker/UTXO state fully resident; holder aggregate values use compact scalar encoding.
- Atom growing state is queried from DB on demand while Atomicals block processing remains intentionally disabled.
- FT is intentionally unchanged in this optimization round.

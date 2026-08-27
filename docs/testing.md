# Indexer test suites

## Default offline suite

The default test contract is offline: it must not contact Bitcoin RPC, public HTTP services, or depend on an existing index database or external validation CSV files.

```bash
go test ./... -count=1
go vet ./...
go build ./...
```

Badger is the default backend. Pebble is retained only as an explicit legacy build:

```bash
go test -tags pebble ./...
go build -tags pebble ./...
```

## Core race suite

Storage/snapshot/cache changes should additionally run:

```bash
go test -race \
  ./indexer/db \
  ./indexer/base \
  ./indexer/exotic \
  ./indexer/nft \
  ./indexer/runes/store \
  ./indexer/atom \
  ./indexer \
  -count=1
```

## Live suite

Tests that require bitcoind, mempool.space, a running local index service, or other real network resources use the `live` build tag:

```bash
go test -tags live ./...
```

`indexer/block_test.go` reads RPC configuration only from environment variables:

```text
INDEXER_LIVE_MAINNET_RPC_HOST
INDEXER_LIVE_MAINNET_RPC_PORT
INDEXER_LIVE_MAINNET_RPC_USER
INDEXER_LIVE_MAINNET_RPC_PASSWORD

INDEXER_LIVE_TESTNET4_RPC_HOST
INDEXER_LIVE_TESTNET4_RPC_PORT
INDEXER_LIVE_TESTNET4_RPC_USER
INDEXER_LIVE_TESTNET4_RPC_PASSWORD
```

No fixed private-network RPC credentials belong in live-test source code.

## Validation-data suite

Large CSV comparison/transformation tests use the `validation` build tag:

```bash
go test -tags validation ./indexer/brc20
```

Their fixtures are explicitly provisioned and are not part of the default test contract.

## Testnet4 replay

Storage changes that affect Base or protocol indexes must be validated against a new isolated Badger directory. Do not reuse a production database for a schema experiment.

The Badger optimization branch was verified on 2026-08-22 with a fresh testnet4 replay from height 0 through 32000 using `period_flush_to_db=10`, followed by the full `IndexerMgr.checkSelf`. The same database was then reopened with the current code and checked a second time. See `testnet4-badger-validation.md` for the measured result.

Height 32000 covers the active testnet4 paths relevant to this optimization, including FT activation at 28883 and Runes activation at 30562.

The replay should exercise at least:

- Base metadata and per-address UTXO records;
- Ordinals/NFT persistence;
- Exotic DB-backed UTXO/holder aggregates;
- FT and BRC-20 processing;
- Runes typed-table/write-cache behavior after activation;
- final Badger GC and full self-check;
- reopen and self-check of the completed DB.

All development changes remain unstaged unless explicitly requested otherwise.


## Badger cache configuration

The production default process-wide Badger block-cache budget is 32768 MiB.
Configure it in YAML under `db`:

```yaml
db:
  path: ./db/mainnet
  badger_block_cache_total_mb: 32768
```

`INDEXER_BADGER_BLOCK_CACHE_TOTAL_MB` (legacy alias
`INDEXER_DB_CACHE_TOTAL_MB`) has higher priority than YAML and is intended for
runtime experiments. Setting the environment variable to `0` disables the
Badger block cache explicitly.

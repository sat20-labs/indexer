# Indexer test suites

## Default offline suite

The default suite must not contact Bitcoin RPC, public HTTP services, or depend
on an existing local index database or external CSV files.

```bash
go test ./...
go vet ./...
```

Badger is the default backend. Pebble is legacy-only and can be selected
explicitly:

```bash
go test -tags pebble ./...
```

## Live suite

Tests that require bitcoind, mempool.space, local index databases, or running
HTTP services use the `live` build tag:

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

No RPC credentials or fixed private-network endpoints belong in source code.

## Validation-data suite

Large CSV comparison and transformation tests use the `validation` build tag:

```bash
go test -tags validation ./indexer/brc20
```

The required validation files are intentionally not part of the default test
contract. They must be provisioned explicitly before this suite is run.

## Testnet4 replay

Schema versions changed for Base, Exotic, and Atom in the Badger optimization
branch. A replay must use a new, isolated DB directory; an existing database is
expected to fail its version check rather than be migrated in place.

A complete replay should exercise at least:

- Base address metadata and per-address UTXO prefix records;
- Ordinals/NFT persistence and NFT primary-table pagination;
- Exotic ticker-UTXO and ticker-holder aggregates;
- FT and BRC-20 processing;
- Runes activation and typed-table write/read-cache separation;
- delayed DB snapshots and final self-check.

The code remains in the working tree. Test commands must not stage or commit
changes.

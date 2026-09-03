# Indexer Badger optimization validation

## Baseline

Repository:

```text
/Users/yingfeng/github/indexer_dev
```

Branch baseline:

```text
feature/indexer-badger-optimization
HEAD b37293e
```

The branch was already committed before this re-validation. All follow-up corrections in this session remain as unstaged worktree changes.

## Verified optimization state

The committed source already contains the main structural changes:

- Badger source backend is the default (`!pebble`); Pebble is explicit `pebble` build-tag legacy mode.
- Badger cache allocation uses a process-level total budget rather than large independent per-DB defaults.
- Badger has the canonical forward/reverse prefix Scan contract and tests.
- Base address metadata is separated from per-address UTXO records; pending additions/deletions are kept as deltas.
- NFT no longer maintains the redundant Gob BuckStore as the write path, and zero-sat serialization is filtered safely.
- Exotic durable ticker/UTXO state is DB-authoritative with pending changes rather than a full startup-resident copy; ticker/address aggregates support holder queries.
- Runes read cache is separated from the non-evictable pending write set.
- Atom growing balance/history state is queried from DB on demand; Atomicals block processing remains intentionally disabled.
- RPC state admission uses an RWMutex barrier rather than the previous atomic-counter admission window.
- Block fetch send is cancellable.
- live-network and validation-data tests are separated from the default offline suite.

## Corrections found during re-validation

The committed `build.sh` still defaulted to Pebble even though the Go backend tags already defaulted to Badger. The worktree now corrects `build.sh` so:

```text
INDEXER_DB_BACKEND unset  -> Badger
INDEXER_DB_BACKEND=badger -> no build tag
INDEXER_DB_BACKEND=pebble -> -tags pebble
```

One Base unit test seeded a durable DB UTXO but constructed its synthetic `TxOutputV2` as non-persisted. The test was corrected to call `MarkPersisted()`; the production durability flag logic was not changed.

A new delayed-buffer lifecycle test was added to verify the important snapshot invariant:

```text
pending output
 -> Clone(true) snapshot
 -> live output spent before snapshot flush
 -> snapshot flush marks shared TxOutputV2 persisted
 -> live subtract retains the durable deletion
 -> next flush decrements UtxoCount and removes address/UTXO record
```

## Actual test gates

The following were executed successfully against `indexer_dev`:

```bash
go test ./... -count=1
go vet ./...
go build ./...
```

Core race suite:

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

All passed.

## Storage encoding benchmark

Actual measured representative encoded sizes:

```text
protobuf:   18 bytes
gob:        90 bytes
raw-fixed:  24 bytes
generic TLV sample: 33 bytes
```

Gob was also substantially slower and more allocation-heavy than Protobuf in the current per-value usage pattern. The policy is therefore:

- Protobuf for structured evolving values;
- compact binary for simple fixed scalars and ordered numeric key components;
- no new Gob online index tables;
- no generic custom TLV unless an end-to-end Badger benchmark proves a material advantage.

See `storage-encoding.md` and the two raw result files for details.

## Testnet4 result

A fresh isolated Badger replay from height 0 through 32000 passed, including FT and Runes activation ranges, and ended with:

```text
IndexerMgr.checkSelf succeed. 9.160479961s
process exit code: 0
peak observed RSS: ~2.50 GiB
DB size: 304 MiB
```

The same database was then reopened with the current code and passed a second complete self-check:

```text
IndexerMgr.checkSelf succeed. 8.896906437s
reopen exit code: 0
```

See `testnet4-badger-validation.md` for full details and the malformed FT input observation retained from the run log.

## Scope deliberately not changed

- No cross-database crash-consistency/recovery protocol was added. An indexing failure still invalidates the build database and requires a rebuild.
- Atomicals block updates remain disabled by design.
- FT full-resident state is not optimized in this round because its current data volume is small.
- Pebble receives no performance investment; it remains a legacy comparison backend only.
- This testnet4 replay validates correctness and bounded cache behavior, but late-mainnet hardware sizing should still be measured separately on a long mainnet range.

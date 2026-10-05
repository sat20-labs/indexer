# Badger testnet4 validation — 2026-08-22

## Result

**PASS**

A fresh Badger index was built from testnet4 height 0 through height 32000, then the completed database was reopened with the current code and checked again.

Validation configuration:

```text
chain: testnet4
start height: 0
target height: 32000
period_flush_to_db: 10
Badger block-cache total: 2048 MiB
Runes read cache: 64 MiB
DB directory: isolated temporary directory
```

The repository's tracked `testnet.env` and existing index databases were not modified. The MCP execution sandbox could not directly route the Go process to the LAN bitcoind endpoint, so the validation used a temporary loopback JSON-RPC bridge. The bridge affected transport only; it did not alter block data or indexer logic and was not added to the repository.

## Fresh replay

Observed final state:

```text
height: 32000
sync height: 32000
process exit code: 0
peak RSS observed: 2,626,096 KiB (~2.50 GiB)
Badger DB size after run: 304 MiB (du -sh)
IndexerMgr.checkSelf: PASS
IndexerMgr.checkSelf duration: 9.160479961s
```

Final markers:

```text
ns DB checked successfully
nft DB checked successfully
runes checked.
FTIndexer CheckSelf took 3.598031ms.
ExoticIndexer CheckSelf took 2.643511011s.
DB checked successfully, 5.068513038s
IndexerMgr.checkSelf succeed. 9.160479961s
reach expected height, set exit flag
```

The replay crossed the testnet4 protocol activation ranges relevant to this optimization:

- NFT/Ordinals path was active before the target range;
- Atomicals activation height is 27000, but Atomicals `UpdateTransfer` remains intentionally disabled by design;
- FT activation: 28883;
- Runes activation: 30562.

After Runes activation the observed RSS continued to remain in the low single-digit GiB range. The largest observed RSS before process exit was about 2.50 GiB with a configured 2 GiB total Badger block-cache budget, which also includes Go heap and Badger/index overhead outside that block-cache budget.

## Reopen verification

The completed height-32000 database was reopened with a newly built binary from the current worktree and checked again.

```text
bitcoind block count during reopen probe: 149461
reopen process exit code: 0
already sync to block 32000-32000
IndexerMgr.checkSelf succeed. 8.896906437s
reach expected height, set exit flag
```

This verifies the startup/decode path in addition to the fresh-build path.

## Protocol-level self-check observations

The final self-check reported, among other invariants:

- BRC-20: 14 holders and CheckSelf PASS;
- Runes: `runes checked.`;
- FT: all indexed ticker holder totals matched minted totals and CheckSelf PASS;
- NFT: primary/sat/UTXO tables and Base address references passed;
- Exotic: ticker holder totals matched ticker UTXO totals and holder details; examples included `vintage`, `black`, `uncommon`, `1sttx`, `rare`, `mythic` and others;
- Base: final DB check PASS.

At height 31989 one malformed FT-related inscription was rejected by four parsing paths because a numeric field contained 54 bytes while the allowed numeric width is 8 bytes. This was logged as an input parse error; the transaction did not corrupt indexed state, and the subsequent FT and global self-checks passed. This observation is retained here rather than treating the log as entirely warning-free.

## Memory observations

Selected observed RSS samples from the same run:

| Approx. height | RSS |
|---:|---:|
| 100 | ~57 MiB |
| 4,000 | ~957 MiB |
| 8,600 | ~1.31 GiB |
| 15,600 | ~1.32 GiB |
| 22,000 | ~1.37 GiB |
| 25,400 | ~1.52 GiB |
| 29,000 | ~1.86 GiB |
| 31,000 | ~1.97 GiB |
| 32,000 / self-check | peak ~2.50 GiB |

This testnet4 range is not a substitute for a late-mainnet memory benchmark, but it confirms that the new global Badger cache budget is enforced and that the optimized state paths do not show the prior tens-of-GiB baseline behavior in this replay.


## Extended replay to height 100000

The same validation was subsequently extended from a fresh height-0 database to testnet4 height 100000. The database was kept across controlled process restarts while runtime-only cache settings and the prefetched-block reuse optimization were validated; it was never rebuilt or replaced during the extension.

Final state:

```text
height: 100000
sync height: 100000
first final CheckSelf: PASS, 4m19.9471778s
first process exit: 0
reopen CheckSelf: PASS, 4m36.321919337s
reopen process exit: 0
persisted DB size: ~2.9 GiB
persisted DB path: db/testnet-100000
```

The final Base DB verification in the first check took about 1m39s; in the reopen verification it took about 1m48s. Runes, NS, NFT, FT, BRC-20, Exotic and Base all completed their self-check paths before the global `IndexerMgr.checkSelf succeed` marker.

### Runtime cache stages

The long replay intentionally exercised different process-wide Badger block-cache budgets:

```text
initial range:  2048 MiB
mid-range:      8192 MiB
later range:   16384 MiB
```

These stages are performance experiments and must not be combined into a single memory-vs-height curve. The production/YAML default was subsequently raised to 32768 MiB and made configurable as `db.badger_block_cache_total_mb`. The local 32-GiB Mac did not run a 32768-MiB block cache because Badger index cache, Go heap and the operating system also need memory.

During the 16-GiB stage, observed RSS remained well below physical memory; selected high-load samples were roughly 4-8 GiB while processing blocks containing hundreds to several thousand transactions. The largest sampled RSS before the final run completed was about 8 GiB. This supports using a substantially larger cache in production while retaining a process-wide cap.

### Prefetched-block reuse optimization discovered during replay

Around height 51k-54k, `processOrdProtocol` slowed from sub-second processing to several seconds per block. Sampling showed that `prepareFreezeLookahead` was calling `FetchBlock(h+1/h+2)` even though the Base fetcher had already downloaded and parsed those future blocks inside its prefetch window.

The Base indexer now exposes a bounded, read-only view of already parsed prefetched blocks. `prepareFreezeLookahead` reuses those pointers first and falls back to `FetchBlock` only when the future block is not yet prefetched. The cache contains only pointers to blocks already owned by the fetch window, is cleared on drain/reorg paths, and removes consumed heights. Targeted tests and race tests passed before the height-100000 replay resumed.

After this change, representative `processOrdProtocol` times in the same region dropped from roughly 3-5 seconds per block to roughly 0.15-0.17 seconds per block, with subsequent performance dominated by Base/Badger point reads and the actual transaction density of testnet4 blocks.

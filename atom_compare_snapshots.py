#!/usr/bin/env python3
import os
import sys


OFFICIAL_SNAPSHOT = os.environ.get("ATOM_OFFICIAL_SNAPSHOT", "/data1/github/atomicals-electrumx/atom-official-snapshot.txt")
INDEXER_SNAPSHOT = os.environ.get("ATOM_INDEXER_SNAPSHOT", "/data1/github/indexer_debug/atom-indexer-snapshot.txt")


def parse_snapshot(path):
    status = ""
    height = -1
    tickers = {}
    utxos = {}
    with open(path, "r", encoding="utf-8") as f:
        for line in f:
            line = line.strip()
            if not line:
                continue
            parts = line.split("|")
            fields = {}
            for part in parts[1:]:
                if "=" in part:
                    key, value = part.split("=", 1)
                    fields[key] = value
            if parts[0] == "status":
                status = line
                height = int(fields.get("height", "-1"))
            elif parts[0] == "ticker":
                tickers[fields["name"]] = fields["atomical"]
            elif parts[0] == "utxo":
                utxos[(fields["ticker"], fields["outpoint"])] = int(fields["amount"])
    return status, height, tickers, utxos


def summarize(utxos):
    rows = {}
    totals = {}
    for ticker, outpoint in utxos:
        rows[ticker] = rows.get(ticker, 0) + 1
        totals[ticker] = totals.get(ticker, 0) + utxos[(ticker, outpoint)]
    return {ticker: (rows[ticker], totals[ticker]) for ticker in sorted(totals)}


def main():
    official_status, official_height, official_tickers, official = parse_snapshot(OFFICIAL_SNAPSHOT)
    indexer_status, indexer_height, indexer_tickers, indexer = parse_snapshot(INDEXER_SNAPSHOT)
    print(f"official_{official_status}")
    print(f"indexer_{indexer_status}")

    ok = True
    if official_height != indexer_height:
        ok = False
        print(f"height_mismatch|official={official_height}|indexer={indexer_height}")

    if official_tickers != indexer_tickers:
        ok = False
        print(f"ticker_mismatch|official={len(official_tickers)}|indexer={len(indexer_tickers)}")
        for ticker in sorted(set(official_tickers) | set(indexer_tickers)):
            if official_tickers.get(ticker) != indexer_tickers.get(ticker):
                print(f"ticker_diff|{ticker}|official={official_tickers.get(ticker)}|indexer={indexer_tickers.get(ticker)}")

    indexer_scoped = {key: value for key, value in indexer.items() if key[0] in official_tickers}
    missing = sorted((key, value, indexer_scoped.get(key)) for key, value in official.items() if indexer_scoped.get(key) != value)
    extra = sorted((key, value, official.get(key)) for key, value in indexer_scoped.items() if official.get(key) != value)
    if missing or extra:
        ok = False
    print(
        "compare|official_tickers=%d|indexer_tickers=%d|official_utxos=%d|indexer_utxos=%d|missing_or_mismatch=%d|extra_or_mismatch=%d"
        % (len(official_tickers), len(indexer_tickers), len(official), len(indexer_scoped), len(missing), len(extra))
    )
    official_summary = summarize(official)
    indexer_summary = summarize(indexer_scoped)
    for ticker, (rows, total) in official_summary.items():
        idx_rows, idx_total = indexer_summary.get(ticker, (0, 0))
        print(f"summary|ticker={ticker}|official_rows={rows}|official_total={total}|indexer_rows={idx_rows}|indexer_total={idx_total}")
    for key, official_value, indexer_value in missing[:50]:
        print(f"official_only_or_diff|ticker={key[0]}|outpoint={key[1]}|official={official_value}|indexer={indexer_value}")
    for key, indexer_value, official_value in extra[:50]:
        print(f"indexer_only_or_diff|ticker={key[0]}|outpoint={key[1]}|indexer={indexer_value}|official={official_value}")
    return 0 if ok else 1


if __name__ == "__main__":
    raise SystemExit(main())

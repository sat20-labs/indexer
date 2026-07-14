# Atomicals ARC-20 indexer notes

This document records the local implementation scope and validation path for
the `indexer/atom` package.

## Scope

- ARC-20 FT direct mint (`ft`) and decentralized FT deploy/mint (`dft` / `dmt`).
- Regular, split (`y`), and custom coloring (`z`) transfer handling. Mainnet
  uses the official custom coloring activation height `848484`; testnet4 uses
  the official `BitcoinTestnet4` constants where Atomicals, DMINT, COMMITZ,
  DENSITY, DFT bitwork rollover, and custom coloring are all activated at
  height `27000`.
- Per-ticker, per-holder, per-UTXO, mint-history, and action-history persistence.
- Query integration through the existing v3 asset interfaces using protocol
  name `atom`.

NFT, realm, container, event, data, and sealed-location operations are only
recognized by the witness parser as framework placeholders. They are not
indexed into state yet.

## Validation

Local verification:

```bash
go test ./indexer/atom -count=1
go test ./indexer -run '^$' -count=1
go test ./... -run '^$' -vet=off
```

Plain `go test ./... -run '^$'` currently fails before this package because of
pre-existing vet format issues in `indexer/mpn` and `indexer/nft`.

External reconciliation should compare our ticker and UTXO state against an
Atomicals ElectrumX proxy. The mainnet proxy provided during implementation is:

```text
https://ep.atomicals.tech/proxy
```

Useful RPC methods exposed by the Atomicals ElectrumX server include:

- `blockchain.atomicals.get_global`
- `blockchain.atomicals.get_ft_info`
- `blockchain.atomicals.get_location`
- `blockchain.atomicals.get_ft_balances_scripthash`

Known testnet4 ARC-20 samples exist after official testnet4 activation height
`27000`.
Examples used during implementation:

- `wizz` DFT deploy at height `31089`, tx
  `88888ccdae6737ebc576e2635c1185fcc81dd0fc4b91aaaedb49af78ebcdfde0`.
- `atom` DFT deploy at height `31730`, tx
  `56a88fd9bf0d5c6077abb47ec4ed16013e05d7bff9dc938b9d56a17fc9b0494d`.

These samples include early burned-asset behavior and custom-coloring
transactions.

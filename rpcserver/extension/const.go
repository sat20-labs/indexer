package extension

const (
	OrdTestnetPreviewUrl  = "https://ord-testnet.ordx.space/preview/"
	OrdTestnetContentUrl  = "https://ord-testnet.ordx.space/content/"
	OrdTestnet4PreviewUrl = "https://ord-testnet4.ordx.space/preview/"
	OrdTestnet4ContentUrl = "https://ord-testnet4.ordx.space/content/"
	OrdMainnetPreviewUrl  = "https://ord-mainnet.ordx.space/preview/"
	OrdMainnetContentUrl  = "https://ord-mainnet.ordx.space/content/"
)

const BtcBitLen = 100000000

type AddressType int

const (
	P2PKH AddressType = iota
	P2WPKH
	P2TR
	P2SH_P2WPKH
	M44_P2WPKH
	M44_P2TR
)

type RiskType int

const (
	SIGHASH_NONE RiskType = iota
	SCAMMER_ADDRESS
	UNCONFIRMED_UTXO
	INSCRIPTION_BURNING
	ATOMICALS_DISABLE
	ATOMICALS_NFT_BURNING
	ATOMICALS_FT_BURNING
	MULTIPLE_ASSETS
	LOW_FEE_RATE
	HIGH_FEE_RATE
	SPLITTING_INSCRIPTIONS
	MERGING_INSCRIPTIONS
	CHANGING_INSCRIPTION
	RUNES_BURNING
)

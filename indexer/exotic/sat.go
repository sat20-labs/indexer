package exotic

import (
	"github.com/sat20-labs/indexer/common"
)

const DificultyAdjustmentInterval = 2016
const HalvingInterval = 210000
const CycleInterval = HalvingInterval * 6

const LastSat = common.MaxSupply - 1
const FirstSat = 0
const MAX_SUBSIDY_HEIGHT = 6929999 // subsidy = 0 when exceeds this height

var PizzaRanges = ReadRangesFromOrdResponse(PIZZA_RANGES)
var NakamotoBlocks = []int{9, 286, 688, 877, 1760, 2459, 2485, 3479, 5326, 9443, 9925, 10645, 14450, 15625, 15817, 19093, 23014, 28593, 29097}
var FirstTxOutPoint = "f4184fc596403b9d638783cf57adfe4c75c605f6356fbc91338530e9831e9e16:0"
var FirstTransactionRanges = []*common.Range{
	{
		Start: 45000000000,
		Size:  1000000000,
	},
}

const FirstTxValue = 1000000000

var FirstTxHeight = 170

var HitmanRanges = SatingRangesToOrdinalsRanges(hitmanSatingRanges)
var JpegRanges = SatingRangesToOrdinalsRanges(jpegSatingRanges)

// TODO rare sat test code，正式版本需要去掉
var CustomizedRange = []*common.Range{
	////
	{
		Start: 1527497194694598,
		Size:  23696,
	},

	////
	{
		Start: 1528464274906263,
		Size:  600,
	},
	{
		Start: 1532527329996942,
		Size:  755,
	},
	{
		Start: 1426523961437350,
		Size:  10000,
	},
}

type Sat int64

func (s Sat) Epoch() Epoch {
	return EpochFromSat(s)
}

func (s Sat) Cycle() int64 {
	return s.Height() / CycleInterval
}

func (s Sat) Period() int64 {
	return s.Height() / DificultyAdjustmentInterval
}

func (s Sat) EpochPosition() int64 {
	r := s - s.Epoch().GetStartingSat()
	return int64(r)
}

func (s Sat) Height() int64 {
	if s < FirstSat || s > LastSat {
		return -1
	}
	epoch := s.Epoch()
	subsidy := epoch.GetSubsidy()
	if subsidy <= 0 {
		return -1
	}
	return int64(epoch)*HalvingInterval + s.EpochPosition()/subsidy
}

func (s Sat) IsFirstSatInBlock() bool {
	if s < FirstSat || s > LastSat {
		return false
	}
	subsidy := s.Epoch().GetSubsidy()
	if subsidy <= 0 {
		return false
	}
	return s.EpochPosition()%subsidy == 0
}

func (s Sat) GetRodarmorRarity() string {
	isFirstSatInBlock := s.IsFirstSatInBlock()
	h := s.Height()

	if s == 0 {
		return Mythic
	}

	if isFirstSatInBlock && h%CycleInterval == 0 {
		return Legendary
	}

	if isFirstSatInBlock && h%HalvingInterval == 0 {
		return Epic
	}

	if isFirstSatInBlock && h%DificultyAdjustmentInterval == 0 {
		return Rare
	}

	if isFirstSatInBlock {
		return Uncommon
	}

	return Common
}

func (s Sat) IsBlack() bool {
	return Sat(s + 1).IsFirstSatInBlock()
}

func (s Sat) IsAlpha() bool {
	return s%1e8 == 0
}

func (s Sat) IsOmega() bool {
	return Sat(s + 1).IsAlpha()
}

func (s Sat) IsFibonacci() bool {
	a := int64(0)
	b := int64(1)
	next := a + b
	for next < int64(s) {
		next = a + b
		a = b
		b = next
	}

	if s == 0 || s == 1 {
		return true
	}

	return int64(s) == next
}

func (s Sat) Satributes() []string {
	exotics := make([]string, 0)

	rarity := s.GetRodarmorRarity()
	if rarity != Common {
		exotics = append(exotics, rarity)
	}

	if s.IsBlack() {
		exotics = append(exotics, Black)
	}

	// TODO
	// if s.IsAlpha() {
	// 	exotics = append(exotics, Alpha)
	// }

	// if s.IsOmega() {
	// 	exotics = append(exotics, Omega)
	// }

	// if s.IsFibonacci() {
	// 	exotics = append(exotics, Fibonacci)
	// }

	if IsSatInRange(PizzaRanges, s) {
		exotics = append(exotics, Pizza)
	}

	if IsSatInRange(FirstTransactionRanges, s) {
		exotics = append(exotics, FirstTransaction)
	}

	// if IsSatInRange(HitmanRanges, s) {
	// 	exotics = append(exotics, Hitman)
	// }

	// if IsSatInRange(JpegRanges, s) {
	// 	exotics = append(exotics, Jpeg)
	// }

	h := s.Height()
	if h == 9 {
		exotics = append(exotics, Block9)
	}

	if h == 78 {
		exotics = append(exotics, Block78)
	}

	if h <= 1000 {
		exotics = append(exotics, Vintage)
	}

	if IsInBlocks(NakamotoBlocks, int(h)) {
		exotics = append(exotics, Nakamoto)
	}

	if IsTestNet {
		if IsSatInRange(CustomizedRange, s) {
			exotics = append(exotics, Customized)
		}
	}

	return exotics
}

func CheckSatribute(sat int64, rarity string) bool {
	satributeList := Sat(sat).Satributes()
	for _, ty := range satributeList {
		if string(ty) == rarity {
			return true
		}
	}
	return false
}

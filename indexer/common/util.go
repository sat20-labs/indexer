package common

import (
	"fmt"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/sat20-labs/indexer/common"
)

func GetInscriptionId(mintutxo string, id int) string {
	parts := strings.Split(mintutxo, ":")
	idstr := strconv.Itoa(int(id))

	return parts[0] + "i" + idstr
}

func ParseInscriptionId(inscId string) (string, int, error) {
	parts := strings.Split(inscId, "i")
	if len(parts) != 2 {
		return inscId, 0, fmt.Errorf("wrong format %s", inscId)
	}

	i, err := strconv.Atoi(parts[1])
	if err != nil {
		return inscId, 0, err
	}

	return parts[0], (i), nil
}

func ParseSatPoint(satpoint string) (string, error) {
	parts := strings.Split(satpoint, ":")
	if len(parts) != 3 {
		return satpoint, fmt.Errorf("wrong format %s", satpoint)
	}

	return parts[0] + parts[1], nil
}

func IsRaritySatRequired(attr *common.SatAttr) bool {
	return attr.Rarity != "" || attr.TrailingZero > 0 ||
		attr.RegularExp != "" || attr.Template != ""
}

func EndsWithNZeroes(num int, n int64) bool {
	dividend := int64(math.Pow10(num))
	return n%dividend == 0
}

func GetInscriptionNumber(utxo string, inscriptionId int) int64 {
	return common.INVALID_INSCRIPTION_NUM
}


func CompareDecimal(amt *common.Decimal, str string) bool {

	if strings.Contains(str, "E") {
		// f, ok := new(big.Float).SetString("1.21906E+11")
		// if !ok {
		// 	common.Log.Panicf("SetString %s failed", str)
		// }
		// str = f.Text('f', -1)
		parts := strings.Split(str, "E")
		parts = strings.Split(parts[0], ".")
		d := 0
		if len(parts) == 2 {
			d = len(parts[1])
		}

		f, ok := new(big.Float).SetString(amt.String())
		if !ok {
			common.Log.Panicf("SetString %s failed", amt.String())
		}
		n := f.Text('E', d)
		return n == str
	}

	parts := strings.Split(str, ".")
	d := 8
	if len(parts) == 2 {
		d = len(parts[1])
	} else {
		d = 0
	}
	if amt != nil && amt.Precision > d {
		amt := amt.SetPrecisionWithRound(d)
		return amt.String() == str
	}
	return false
}
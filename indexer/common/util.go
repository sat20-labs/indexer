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

// 校验数据不规范，需要做一些处理，非通用接口
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
		n := f.Text('E', d) // 没有做四舍五入，而是直接切断了
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
		amt1 := amt.SetPrecisionWithRound(d)
		if amt1.String() == str {
			return true
		}
		amt2 := amt.SetPrecision(d)
		return amt2.String() == str
	}
	return amt.String() == str
}

func IntToSciHalfUp10(n *big.Int, decimals int) string {
    s := n.String()
    if len(s) == 0 {
        return "0E+0"
    }

    // 10 进制指数
    exp := len(s) - 1

    // 需要的有效数字数 = 1 + decimals
    need := 1 + decimals

    // 确保有一位用于判断进位
    cut := need + 1

    // 不足补零
    if len(s) < cut {
        s = s + strings.Repeat("0", cut-len(s))
    }

    main := s[:need]      // 主体
    roundDigit := s[need] // 判断位

    // half-up 舍入
    if roundDigit >= '5' {
        i := new(big.Int)
        i.SetString(main, 10)
        i.Add(i, big.NewInt(1))
        main = i.String()
    }

    // 处理进位导致长度变化（如 9.9999 → 10.000）
    if len(main) > need {
        exp++
        main = main[:need]
    }

    // 组装 mantissa
    if decimals > 0 {
        return fmt.Sprintf(
            "%s.%sE+%d",
            main[:1],
            main[1:],
            exp,
        )
    }

    return fmt.Sprintf("%sE+%d", main[:1], exp)
}



func CompareForRunes(amt *big.Int, str string) bool {

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

		n := IntToSciHalfUp10(amt, d)
		return n == str
	}

	if amt != nil {
		return amt.String() == str
	}
	return false
}

package indexer

import (
	"fmt"
	"math"
	"regexp"
	"runtime"
	"strconv"
	"strings"

	"github.com/sat20-labs/ordx/common"
	"github.com/sat20-labs/ordx/indexer/exotic"
)

// memory util
func PrintMemUsage() {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// For info on each, see: https://golang.org/pkg/runtime/#MemStats
	fmt.Printf("Alloc = %v MiB", bToMb(m.Alloc))
	fmt.Printf("\tTotalAlloc = %v MiB", bToMb(m.TotalAlloc))
	fmt.Printf("\tSys = %v MiB", bToMb(m.Sys))
	fmt.Printf("\tNumGC = %v\n", m.NumGC)
}

func GetSysMb() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return bToMb(m.Sys)
}

func GetAlloc() uint64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return bToMb(m.Alloc)
}

func bToMb(b uint64) uint64 {
	return b / 1024 / 1024
}

func isValidExoticType(ty string) bool {
	for _, s := range exotic.SatributeList {
		if string(s) == ty {
			return true
		}
	}
	return false
}

func parseSatAttrString(s string) (common.SatAttr, error) {
	attr := common.SatAttr{}
	attributes := strings.Split(s, ";")
	for _, attribute := range attributes {
		pair := strings.SplitN(attribute, "=", 2)
		if len(pair) != 2 {
			return attr, fmt.Errorf("invalid attribute format: %s", attribute)
		}
		key := pair[0]
		value := pair[1]

		switch key {
		case "rar":
			if isValidExoticType(value) {
				attr.Rarity = value
			} else {
				return attr, fmt.Errorf("invalid exotic type value: %s", value)
			}
		case "trz":
			trailingZero, err := strconv.Atoi(value)
			if err != nil {
				return attr, fmt.Errorf("invalid trailing zero value: %s", value)
			}
			if trailingZero <= 0 {
				return attr, fmt.Errorf("invalid trailing zero value: %s", value)
			}
			attr.TrailingZero = trailingZero
		case "tmpl":
			attr.Template = value
		case "reg":
			attr.RegularExp = value
		}
	}

	return attr, nil
}

func skipOffsetRange(ord []*common.Range, satpoint int) []*common.Range {
	if satpoint == 0 {
		return ord
	}

	result := make([]*common.Range, 0)
	for _, rng := range ord {
		// skip the offset
		if satpoint > 0 {
			if int64(satpoint) >= (rng.Size) {
				satpoint -= int(rng.Size)
			} else {
				newRange := common.Range{Start: rng.Start + int64(satpoint), Size: rng.Size - int64(satpoint)}
				result = append(result, &newRange)
				satpoint = 0
			}
			continue
		}

		result = append(result, rng)
	}
	return result
}

func reSizeRange(ord []*common.Range, amt int64) []*common.Range {
	result := make([]*common.Range, 0)
	size := int64(0)
	for _, rng := range ord {
		if size+(rng.Size) <= amt {
			result = append(result, rng)
			size += (rng.Size)
		} else {
			newRng := common.Range{Start: rng.Start, Size: (amt - size)}
			result = append(result, &newRng)
			size += (newRng.Size)
		}

		if size == amt {
			break
		}
	}
	return result
}

func reAlignRange(ord []*common.Range, satpoint int, amt int64) []*common.Range {
	ret := skipOffsetRange(ord, satpoint)
	return reSizeRange(ret, amt)
}

func getPercentage(str string) (int, error) {
	// 只接受两位小数，或者100%
	str2 := strings.TrimSpace(str)

	var f float64
	var err error
	if strings.Contains(str2, "%") {
		str2 = strings.TrimRight(str2, "%") // 去掉百分号
		if strings.Contains(str2, ".") {
			parts := strings.Split(str2, ".")
			str3 := strings.Trim(parts[1], "0")
			if str3 != "" {
				return 0, fmt.Errorf("invalid format %s", str)
			}
		}
		f, err = strconv.ParseFloat(str2, 32)
	} else {
		regex := `^\d+(\.\d{0,2})?$`
		str2 = strings.TrimRight(str2, "0")
		var math bool
		math, err = regexp.MatchString(regex, str2)
		if err != nil || !math {
			return 0, fmt.Errorf("invalid format %s", str)
		}

		f, err = strconv.ParseFloat(str2, 32)
		f = f * 100
	}

	r := int(math.Round(f))
	if r > 100 {
		return 0, fmt.Errorf("invalid format %s", str)
	}

	return r, err
}

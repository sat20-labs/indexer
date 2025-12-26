package indexer

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emirpasic/gods/maps/treemap"
	"github.com/emirpasic/gods/utils"
	"github.com/sat20-labs/indexer/common"
	"github.com/sat20-labs/indexer/indexer/db"
	"github.com/sat20-labs/indexer/indexer/exotic"
	"github.com/sat20-labs/indexer/indexer/nft"
	"github.com/sat20-labs/indexer/indexer/ns"
)

type Inscription struct {
	ID string `json:"id"`
}

func (p *IndexerMgr) FilterRarepizza() bool {
	rarepizzaOldFilepath := "./rarepizza_old.json"
	rarepizzaNewFilepath := "./rarepizza_new.json"
	if _, err := os.Stat(rarepizzaNewFilepath); err == nil {
		os.Remove(rarepizzaOldFilepath)
		err := os.Rename(rarepizzaNewFilepath, rarepizzaOldFilepath)
		if err != nil {
			common.Log.Infof("Rename failed: %v\n", err)
			return true
		}
	}

	rarepizzaInFile, err := os.Open(rarepizzaOldFilepath)
	if err != nil {
		common.Log.Infof("Error opening file: %v", err)
		return true
	}
	defer rarepizzaInFile.Close()

	rarepizzaInData, err := io.ReadAll(rarepizzaInFile)
	if err != nil {
		common.Log.Infof("Error reading file: %v", err)
		return true
	}
	var oldInscriptionList []Inscription
	err = json.Unmarshal(rarepizzaInData, &oldInscriptionList)
	if err != nil {
		common.Log.Infof("Error unmarshaling JSON: %v", err)
		return true
	}
	common.Log.Infof("old count %d", len(oldInscriptionList))

	var newInscriptionList []Inscription
	mintInfos := p.ftIndexer.GetMintHistory("rarepizza", 0, 200000)
	for _, info := range mintInfos {
		if info.Amount.Int64() != 1000 {
			continue
		}
		nft := p.GetNftInfoWithInscriptionId(info.InscriptionId)
		if nft == nil {
			continue
		}

		amount := p.ftIndexer.GetAssetsWithUtxoV3(nft.UtxoId, "rarepizza")
		if amount != 1000 {
			continue
		}

		newInscriptionList = append(newInscriptionList, Inscription{ID: info.InscriptionId})
	}
	common.Log.Infof("new count %d", len(newInscriptionList))

	// save current result
	rarepizzaOutData, err := json.MarshalIndent(newInscriptionList, "", "  ")
	if err != nil {
		common.Log.Infof("Error marshaling JSON: %v", err)
		return true
	}
	err = os.WriteFile(rarepizzaNewFilepath, rarepizzaOutData, 0644)
	if err != nil {
		common.Log.Infof("Error writing file: %v", err)
		return true
	}

	// find the different
	diff := findDiff(oldInscriptionList, newInscriptionList)
	common.Log.Infof("diff count %d", len(diff))
	diffOutData, err := json.MarshalIndent(diff, "", "  ")
	if err != nil {
		common.Log.Infof("Error marshaling JSON: %v", err)
		return true
	}
	err = os.WriteFile("./rarepizza_diff.json", diffOutData, 0644)
	if err != nil {
		common.Log.Infof("Error writing file: %v", err)
		return true
	}

	return true
}

func findDiff(oldv, newv []Inscription) []Inscription {
	var result []Inscription
	nmap := make(map[string]Inscription)
	for _, it := range newv {
		nmap[it.ID] = it
	}

	for _, it := range oldv {
		_, ok := nmap[it.ID]
		if !ok {
			result = append(result, it)
		}
	}

	return result
}

func (p *IndexerMgr) listMintHistory() bool {
	filepath := "./rarepizza_mint.json"
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Log.Infof("OpenFile failed %v", err)
		return true
	}

	file.WriteString("[\n")
	mintInfos := p.ftIndexer.GetMintHistory("rarepizza", 0, 200000)
	for i := 0; i < len(mintInfos)-1; i++ {
		info := mintInfos[i]
		file.WriteString(fmt.Sprintf("{\"id\":\"%s\"},\n", info.InscriptionId))
	}
	info := mintInfos[len(mintInfos)-1]
	file.WriteString(fmt.Sprintf("{\"id\":\"%s\"}\n", info.InscriptionId))
	file.WriteString("]")

	file.Close()

	return true
}

func (p *IndexerMgr) listValidPearl() bool {
	filepath := "./pearl.json"
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Log.Infof("OpenFile failed %v", err)
		return true
	}

	file.WriteString("[\n")
	mintInfos := p.ftIndexer.GetMintHistory("pearl", 0, 200000)
	splitted := p.ftIndexer.GetSplittedInscriptionsWithTick("pearl")
	splittedMap := make(map[string]bool)
	for _, v := range splitted {
		splittedMap[v] = true
	}

	for i := 0; i < len(mintInfos)-1; i++ {
		info := mintInfos[i]
		_, ok := splittedMap[info.InscriptionId]
		if ok {
			continue
		}
		file.WriteString(fmt.Sprintf("{\"id\":\"%s\"},\n", info.InscriptionId))
	}
	info := mintInfos[len(mintInfos)-1]
	file.WriteString(fmt.Sprintf("{\"id\":\"%s\"}\n", info.InscriptionId))
	file.WriteString("]")

	file.Close()

	return true
}

func (p *IndexerMgr) searchName() bool {
	//	p.searchName3("3letters.txt", "")
	//	p.searchName3("3letters-btc.txt", ".btc")

	// p.searchZ3("3b-btc.txt", ".btc") // completed

	p.searchL4("4l.txt", "") // some
	//p.searchL4("4l-btc.txt", ".btc") // completed
	//p.searchcvcv(".btc")
	//p.searchcvvc(".btc")

	p.searchD5("5d.txt", "") // some
	//p.searchL5("5l-btc.txt", ".btc")

	//p.searchL6("6l.txt", "")
	//p.searchD6("6d-btc.txt", ".btc") // completed
	p.searchD7("7d-btc.txt", ".btc")

	//	p.searchValue5D("")
	//	p.searchValue6D(".btc")
	//p.searchValue8D(".btc")

	//	p.searcBIP39()

	p.searcD8()

	//p.subNameStatistic(".btc")
	//p.subNameStatistic("")

	return true
}

func (p *IndexerMgr) searchcvcv(suffix string) {
	vowels := []rune{'a', 'e', 'i', 'o', 'u'}
	consonants := []rune{'b', 'c', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'y', 'z'}

	var names []string

	for _, c1 := range consonants {
		for _, v1 := range vowels {
			for _, c2 := range consonants {
				for _, v2 := range vowels {
					combination := string([]rune{c1, v1, c2, v2})
					if !p.ns.IsNameExist(combination + suffix) {
						names = append(names, combination+suffix)
					}
				}
			}
		}
	}

	common.Log.Infof("get records %d", len(names))
	p.writeToFile("cvcv"+suffix, names)
}

func (p *IndexerMgr) searchcvvc(suffix string) {
	vowels := []rune{'a', 'e', 'i', 'o', 'u'}
	consonants := []rune{'b', 'c', 'd', 'f', 'g', 'h', 'j', 'k', 'l', 'm', 'n', 'p', 'q', 'r', 's', 't', 'v', 'w', 'x', 'y', 'z'}

	var names []string

	for _, c1 := range consonants {
		for _, v1 := range vowels {
			for _, v2 := range vowels {
				for _, c2 := range consonants {
					combination := string([]rune{c1, v1, v2, c2})
					if !p.ns.IsNameExist(combination + suffix) {
						names = append(names, combination+suffix)
					}
				}
			}
		}
	}

	common.Log.Infof("get records %d", len(names))
	p.writeToFile("cvvc"+suffix, names)
}

func (p *IndexerMgr) searchValue5D(suffix string) {
	names := make([]string, 0)
	size := 5
	for i := 0; i <= 99999; i++ {
		numStr := strconv.Itoa(i)
		if len(numStr) < size {
			numStr = strings.Repeat("0", size-len(numStr)) + numStr
		}
		if isValuable(numStr, size) {
			if !p.ns.IsNameExist(numStr + suffix) {
				names = append(names, numStr+suffix)
			}
		}
	}
	common.Log.Infof("get records %d", len(names))
	p.writeToFile("5d-value-"+suffix+".txt", names)
}

func (p *IndexerMgr) searchValue6D(suffix string) {
	names := make([]string, 0)
	size := 6
	for i := 0; i <= 999999; i++ {
		numStr := strconv.Itoa(i)
		if len(numStr) < size {
			numStr = strings.Repeat("0", size-len(numStr)) + numStr
		}
		if isValuable(numStr, size) {
			if !p.ns.IsNameExist(numStr + suffix) {
				names = append(names, numStr+suffix)
			}
		}
	}
	common.Log.Infof("get records %d", len(names))
	p.writeToFile("6d-value-"+suffix+".txt", names)
}

func (p *IndexerMgr) searchValue8D(suffix string) {
	names := make([]string, 0)
	size := 8
	for i := 0; i <= 99999999; i++ {
		numStr := strconv.Itoa(i)
		if len(numStr) < size {
			numStr = strings.Repeat("0", size-len(numStr)) + numStr
		}
		if isValuable(numStr, size) {
			if !p.ns.IsNameExist(numStr + suffix) {
				names = append(names, numStr+suffix)
			}
		}
	}
	common.Log.Infof("get records %d", len(names))
	p.writeToFile("8d-value-"+suffix+".txt", names)
}

func isValuable(num string, size int) bool {
	// 不包含4
	if strings.Contains(num, "0") ||
		strings.Contains(num, "1") ||
		strings.Contains(num, "2") ||
		strings.Contains(num, "3") ||
		strings.Contains(num, "5") ||
		strings.Contains(num, "6") ||
		strings.Contains(num, "7") ||
		//strings.Contains(num, "8") ||
		strings.Contains(num, "9") ||
		strings.Contains(num, "4") {
		return false
	}

	// 5个相同数字
	if countSame(num) >= size-2 {
		return true
	}

	// 8出现4次或以上
	if strings.Count(num, "8") >= size-2 {
		return true
	}

	// 连号(至少4个)
	if hasConsecutive(num, size) {
		return true
	}

	// 对称
	if isSymmetric(num) {
		return true
	}

	//顺序递增或递减
	if isSequential(num) {
		return true
	}

	if size == 6 {
		// //AABBCC模式
		if isAABBCC(num) {
			return true
		}
		// //ABCABC模式
		if isABCABC(num) {
			return true
		}
	}

	if size == 5 {
		// //AABBCC模式
		if isAABCC(num) {
			return true
		}
		// //ABCABC模式
		if isABCAB(num) {
			return true
		}
	}

	// 全相同数字
	if allSame(num) {
		return true
	}

	return false
}

func hasConsecutive(num string, n int) bool {
	for i := 0; i <= len(num)-n; i++ {
		if isConsecutive(num[i : i+n]) {
			return true
		}
	}
	return false
}

func isConsecutive(s string) bool {
	increasing := true
	decreasing := true
	for i := 1; i < len(s); i++ {
		if int(s[i]) != int(s[i-1])+1 {
			increasing = false
		}
		if int(s[i]) != int(s[i-1])-1 {
			decreasing = false
		}
	}
	return increasing || decreasing
}

func isSymmetric(num string) bool {
	size := len(num)
	if size == 6 || size == 7 {
		return num[0] == num[size-1] && num[1] == num[size-2] && num[2] == num[size-3]
	} else if size == 8 || size == 9 {
		return num[0] == num[size-1] && num[1] == num[size-2] && num[2] == num[size-3] && num[3] == num[size-4]
	} else if size == 10 {
		return num[0] == num[size-1] && num[1] == num[size-2] && num[2] == num[size-3] && num[3] == num[size-4] && num[4] == num[size-5]
	} else if size == 4 || size == 5 {
		return num[0] == num[size-1] && num[1] == num[size-2]
	}
	return false
}

func isSequential(num string) bool {
	increasing := true
	decreasing := true
	for i := 1; i < len(num); i++ {
		if int(num[i]) != int(num[i-1])+1 {
			increasing = false
		}
		if int(num[i]) != int(num[i-1])-1 {
			decreasing = false
		}
	}
	return increasing || decreasing
}

func isAABBCC(num string) bool {
	return num[0] == num[1] && num[2] == num[3] && num[4] == num[5]
}

func isABCABC(num string) bool {
	return num[0] == num[3] && num[1] == num[4] && num[2] == num[5]
}

func isAABCC(num string) bool {
	return (num[0] == num[1] && num[3] == num[4]) ||
		(num[1] == num[2] && num[3] == num[4]) ||
		(num[0] == num[1] && num[2] == num[3])
}

func isABCAB(num string) bool {
	return num[0] == num[3] && num[1] == num[4]
}

func allSame(num string) bool {
	for i := 1; i < len(num); i++ {
		if num[i] != num[0] {
			return false
		}
	}
	return true
}

func countSame(num string) int {
	count := make(map[rune]int)
	for _, r := range num {
		count[r]++
	}
	max := 0
	for _, v := range count {
		if v > max {
			max = v
		}
	}
	return max
}

func (p *IndexerMgr) searcBIP39() bool {
	startTime := time.Now()

	names := make([]string, 0)
	words := listMnemonicWords()
	for _, word := range words {
		if !p.ns.IsNameExist(word + ".btc") {
			names = append(names, word+".btc")
		}
	}

	common.Log.Infof("search bip39 words takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile("bip39.txt", names)

	return true
}

func (p *IndexerMgr) searcD8() bool {
	startTime := time.Now()
	// _names = make([]string, 0)
	// p.generate88Strings("",0,0)
	// common.Log.Infof("search 88 digits takes %v", time.Since(startTime))
	// common.Log.Infof("get records %d", len(_names))
	// p.writeToFile("8d_88", _names)

	// startTime = time.Now()
	// _names = make([]string, 0)
	// p.generate98Strings(8)
	// common.Log.Infof("search 98 digits takes %v", time.Since(startTime))
	// common.Log.Infof("get records %d", len(_names))
	// p.writeToFile("8d_98", _names)

	startTime = time.Now()
	_names = make([]string, 0)
	p.generate520Strings(".btc", 8)
	common.Log.Infof("search 520 digits takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(_names))
	p.writeToFile("8d_520", _names)

	startTime = time.Now()
	_names = make([]string, 0)
	p.generateC1C2Strings("0", "8", ".btc", 8)
	p.generateC1C2Strings("6", "8", ".btc", 8)
	p.generateC1C2Strings("6", "9", ".btc", 8)
	common.Log.Infof("search 68 digits takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(_names))
	p.writeToFile("8d_68", _names)

	startTime = time.Now()
	_names = make([]string, 0)
	p.generateIncreasingStrings("0123456789", 8)
	p.generateIncreasingStrings("9876543210", 8)
	common.Log.Infof("search 520 digits takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(_names))
	p.writeToFile("8d_123", _names)

	startTime = time.Now()
	_names = make([]string, 0)
	p.generate8BDateStrings(1960, 2030)
	common.Log.Infof("search date takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(_names))
	p.writeToFile("8d_date", _names)

	return true
}

func (p *IndexerMgr) searcD10() bool {
	//startTime := time.Now()
	// _names = make([]string, 0)
	// p.generate88Strings("",0,0)
	// common.Log.Infof("search 88 digits takes %v", time.Since(startTime))
	// common.Log.Infof("get records %d", len(_names))
	// p.writeToFile("10d_88", _names)

	// startTime = time.Now()
	// _names = make([]string, 0)
	// p.generate98Strings()
	// common.Log.Infof("search 98 digits takes %v", time.Since(startTime))
	// common.Log.Infof("get records %d", len(_names))
	// p.writeToFile("10d_98", _names)

	// startTime = time.Now()
	// _names = make([]string, 0)
	// p.generate520Strings(10)
	// common.Log.Infof("search 520 digits takes %v", time.Since(startTime))
	// common.Log.Infof("get records %d", len(_names))
	// p.writeToFile("10d_520", _names)

	// startTime = time.Now()
	// _names = make([]string, 0)
	// p.generateC1C2Strings("6", "8")
	// p.generateC1C2Strings("8", "6")
	// p.generateC1C2Strings("6", "9")
	// common.Log.Infof("search 68 digits takes %v", time.Since(startTime))
	// common.Log.Infof("get records %d", len(_names))
	// p.writeToFile("10d_68", _names)

	// startTime = time.Now()
	// _names = make([]string, 0)
	// p.generateIncreasingStrings("0123456789")
	// p.generateIncreasingStrings("9876543210")
	// common.Log.Infof("search 520 digits takes %v", time.Since(startTime))
	// common.Log.Infof("get records %d", len(_names))
	// p.writeToFile("10d_123", _names)

	return true
}

func writeMapToFile(filepath string, contents map[string]bool) {
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Log.Infof("OpenFile failed %v", err)
		return
	}

	file.WriteString(fmt.Sprintf("the total records: %d\n", len(contents)))
	for v := range contents {
		_, err = file.WriteString(v + "\n")
		if err != nil {
			common.Log.Infof("WriteString failed %v", err)
			break
		}
	}

	file.Close()
}

func (p *IndexerMgr) SearchPredefinedName() bool {

	phoneNumbers := []string{
		"12345", "54321", "11111", "99999", "10101",
	}

	names := make(map[string]bool, 0)
	for _, pn := range phoneNumbers {
		if !p.ns.IsNameExist(pn) {
			names[pn] = true
		}
	}

	common.Log.Infof("get records %d", len(names))
	writeMapToFile("5d_01.txt", names)

	return true
}

// 分析pizza聪的分布
func (p *IndexerMgr) pizzaStatistic(bRegenerat bool) bool {

	pizzaAddrMap := make(map[uint64]int64)
	ldb := p.base.GetBaseDB()
	key := []byte("pizza-holder")

	err := db.GetValueFromDB(key, &pizzaAddrMap, ldb)
	if err != nil || bRegenerat {
		pizzaAddrMap = make(map[uint64]int64)
		pizzaAddrMap[0] = int64(p.GetSyncHeight())
		startTime2 := time.Now()
		common.Log.Infof("calculating in %s table ...", common.DB_KEY_UTXO)
		total := int64(0)
		ldb.BatchRead([]byte(common.DB_KEY_UTXO), false, func(k, v []byte) error {
			var value common.UtxoValueInDB
			err := db.DecodeBytesWithProto3(v, &value)
			if err != nil {
				common.Log.Panicf("item.Value error: %v", err)
			}

			er := p.exotic.GetExoticsWithType(value.UtxoId, exotic.Pizza)
			if len(er) > 0 {
				num := er.Size()
				total += num
				pizzaAddrMap[(value.AddressId)] += num
			}

			return nil
		})
		common.Log.Infof("%s table takes %v", common.DB_KEY_UTXO, time.Since(startTime2))
		common.Log.Infof("network %s, block %d, %d addresses hold pizza %d",
			p.chaincfgParam.Name, p.GetSyncHeight(), len(pizzaAddrMap), total)

		db.GobSetDB(key, pizzaAddrMap, ldb)
	}

	syncHeight := pizzaAddrMap[0]
	delete(pizzaAddrMap, 0)

	addrTreeMap := treemap.NewWith(utils.Int64Comparator)
	total := int64(0)
	for k, v := range pizzaAddrMap {
		addrTreeMap.Put(v, k)
		total += v
	}
	common.Log.Infof("block height %d, total %d addresses hold pizza %d", syncHeight, len(pizzaAddrMap), total)

	type AddressPizza struct {
		Address string
		Pizza   int64
	}

	theBigAddresses := make([]*AddressPizza, 0)
	num1 := 1000
	total1 := int64(0)
	theTaprootAddresses := make([]*AddressPizza, 0)
	total2 := int64(0)

	ldb.View(func(txn common.ReadBatch) error {
		it := addrTreeMap.Iterator()
		it.End()

		for it.Prev() {
			pizza := it.Key().(int64)
			addressId := it.Value().(uint64)
			address, err := db.GetAddressByID(txn, addressId)
			if err != nil {
				common.Log.Warnf("GetAddressByID %d failed. %v", addressId, err)
			} else {
				if len(theBigAddresses) < num1 {
					total1 += pizza
					theBigAddresses = append(theBigAddresses, &AddressPizza{Address: address, Pizza: pizza})
				}
				if strings.HasPrefix(address, "bc1p") {
					total2 += pizza
					theTaprootAddresses = append(theTaprootAddresses, &AddressPizza{Address: address, Pizza: pizza})
				}
			}
		}
		return nil
	})

	common.Log.Infof("the top %d addresses hold pizza %d", len(theBigAddresses), total1)
	common.Log.Infof("the total %d taproot addresses hold pizza %d", len(theTaprootAddresses), total2)

	os.Remove("./pizza.txt")
	file, err := os.OpenFile("./pizza.txt", os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Log.Infof("OpenFile failed %v", err)
		return true
	}

	file.WriteString(fmt.Sprintf("block height %d, total %d addresses hold pizza %d\n", syncHeight, len(pizzaAddrMap), total))
	file.WriteString(fmt.Sprintf("the top %d addresses hold pizza %d\n", len(theBigAddresses), total1))
	file.WriteString(fmt.Sprintf("the total %d taproot addresses hold pizza %d\n", len(theTaprootAddresses), total2))

	file.WriteString("\n\nthe top pizza holder:\n")
	for _, v := range theBigAddresses {
		line := v.Address + " " + strconv.FormatInt(v.Pizza, 10) + "\n"
		_, err = file.WriteString(line)
		if err != nil {
			common.Log.Infof("WriteString failed %v", err)
			break
		}
	}

	file.WriteString("\n\ntaproot address:\n")
	for _, v := range theTaprootAddresses {
		line := v.Address + " " + strconv.FormatInt(v.Pizza, 10) + "\n"
		_, err = file.WriteString(line)
		if err != nil {
			common.Log.Infof("WriteString failed %v", err)
			break
		}
	}

	file.Close()

	return true
}

func (p *IndexerMgr) writeToFile(filepath string, contents []string) {
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Log.Infof("OpenFile failed %v", err)
		return
	}

	file.WriteString(fmt.Sprintf("the total records: %d (height %d)\n", len(contents), p.base.GetHeight()))
	for _, v := range contents {
		_, err = file.WriteString(v + "\n")
		if err != nil {
			common.Log.Infof("WriteString failed %v", err)
			break
		}
	}

	file.Close()
}

func (p *IndexerMgr) searchName3(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			for k := 0; k < 10; k++ {
				name := fmt.Sprintf("%d%d%d%s", i, j, k, suffix)
				if !p.ns.IsNameExist(name) {
					names = append(names, name)
				}
			}
		}
	}

	letters := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(letters); i++ {
		for j := 0; j < len(letters); j++ {
			for k := 0; k < len(letters); k++ {
				name := string(letters[i]) + string(letters[j]) + string(letters[k]) + suffix
				if !p.ns.IsNameExist(name) {
					names = append(names, name)
				}
			}
		}
	}
	common.Log.Infof("search 3 letters takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)

	return true
}

func (p *IndexerMgr) searchD4(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			for k := 0; k < 10; k++ {
				for l := 0; l < 10; l++ {
					name := fmt.Sprintf("%d%d%d%d%s", i, j, k, l, suffix)
					if !p.ns.IsNameExist(name) {
						names = append(names, name)
					}
				}
			}
		}
	}

	common.Log.Infof("search 4 digits takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)

	return true
}

func (p *IndexerMgr) searchZ3(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)

	letters := "abcdefghijklmnopqrstuvwxyz0123456789"
	for i := 0; i < len(letters); i++ {
		for j := 0; j < len(letters); j++ {
			for k := 0; k < len(letters); k++ {
				name := string(letters[i]) + string(letters[j]) + string(letters[k]) + suffix
				if !p.ns.IsNameExist(name) {
					names = append(names, name)
				}
			}
		}
	}
	common.Log.Infof("search 3 bytes takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)

	return true
}

func (p *IndexerMgr) searchL4(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)

	letters := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(letters); i++ {
		for j := 0; j < len(letters); j++ {
			for k := 0; k < len(letters); k++ {
				for l := 0; l < len(letters); l++ {
					name := string(letters[i]) + string(letters[j]) + string(letters[k]) + string(letters[l]) + suffix
					if !p.ns.IsNameExist(name) {
						names = append(names, name)
					}
				}
			}
		}
	}
	common.Log.Infof("search 4 letters takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)

	return true
}

func (p *IndexerMgr) searchD5(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			for k := 0; k < 10; k++ {
				for l := 0; l < 10; l++ {
					for m := 0; m < 10; m++ {
						name := fmt.Sprintf("%d%d%d%d%d%s", i, j, k, l, m, suffix)
						if !p.ns.IsNameExist(name) {
							names = append(names, name)
						}
					}
				}
			}
		}
	}

	common.Log.Infof("search 5 digits takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)

	return true
}

func (p *IndexerMgr) searchL5(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)

	letters := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(letters); i++ {
		for j := 0; j < len(letters); j++ {
			for k := 0; k < len(letters); k++ {
				for l := 0; l < len(letters); l++ {
					for m := 0; m < len(letters); m++ {
						name := string(letters[i]) + string(letters[j]) + string(letters[k]) + string(letters[l]) + string(letters[m]) + suffix
						if !p.ns.IsNameExist(name) {
							names = append(names, name)
						}
					}
				}
			}
		}
	}
	common.Log.Infof("search 5 letters takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)

	return true
}

func (p *IndexerMgr) searchD6(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			for k := 0; k < 10; k++ {
				for l := 0; l < 10; l++ {
					for m := 0; m < 10; m++ {
						for n := 0; n < 10; n++ {
							name := fmt.Sprintf("%d%d%d%d%d%d%s", i, j, k, l, m, n, suffix)
							if !p.ns.IsNameExist(name) {
								names = append(names, name)
							}
						}
					}
				}
			}
		}
	}

	common.Log.Infof("search 6 digits takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)
	return true
}

func (p *IndexerMgr) searchD7(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)
	for i := 0; i < 10; i++ {
		for j := 0; j < 10; j++ {
			for k := 0; k < 10; k++ {
				for l := 0; l < 10; l++ {
					for m := 0; m < 10; m++ {
						for n := 0; n < 10; n++ {
							for o := 0; o < 10; o++ {
								name := fmt.Sprintf("%d%d%d%d%d%d%d%s", i, j, k, l, m, n, o, suffix)
								if !p.ns.IsNameExist(name) {
									names = append(names, name)
								}
							}
						}
					}
				}
			}
		}
	}

	common.Log.Infof("search 7 digits takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)
	return true
}

func (p *IndexerMgr) searchL6(filepath, suffix string) bool {
	startTime := time.Now()
	names := make([]string, 0)

	letters := "abcdefghijklmnopqrstuvwxyz"
	for i := 0; i < len(letters); i++ {
		for j := 0; j < len(letters); j++ {
			for k := 0; k < len(letters); k++ {
				for l := 0; l < len(letters); l++ {
					for m := 0; m < len(letters); m++ {
						name := string(letters[i]) + string(letters[j]) + string(letters[k]) + string(letters[l]) + string(letters[m]) + suffix
						if !p.ns.IsNameExist(name) {
							names = append(names, name)
						}
					}
				}
			}
		}
	}
	common.Log.Infof("search 6 letters takes %v", time.Since(startTime))
	common.Log.Infof("get records %d", len(names))
	p.writeToFile(filepath, names)

	return true
}

var _names []string

// generateSpecialStrings 生成所有长度为 10 且至少包含 8 个 '8' 的字符串，并且其他字符不包含 '4'
func (p *IndexerMgr) generate88Strings(prefix string, length int, count8 int) {
	// 如果字符串长度达到 10 且符合条件，输出字符串
	if length == 10 {
		if count8 >= 8 {
			if !p.ns.IsNameExist(prefix) {
				_names = append(_names, prefix)
			}
		}
		return
	}

	// 添加 '8' 并递归调用
	p.generate88Strings(prefix+"8", length+1, count8+1)

	// 添加其他数字（0-9，但不包括 4 和 8）并递归调用
	for i := 0; i < 10; i++ {
		if i != 4 && i != 8 {
			p.generate88Strings(prefix+string(rune('0'+i)), length+1, count8)
		}
	}
}

// generateSpecialStrings 生成所有长度为 10 且包含 9 个 '8' 的字符串，并且另一个字符不包含 '4'
func (p *IndexerMgr) generate98Strings(size int) {
	// 定义允许的单个字符集合（不包括 '4'）
	allowedChars := []byte{'0', '1', '2', '3', '5', '6', '7', '9'}

	// 基本字符串由 9 个 '8' 组成
	baseString := strings.Repeat("8", 9)

	// 遍历字符串的每一个位置，将非 '4' 的字符插入其中
	for _, char := range allowedChars {
		for i := 0; i < size; i++ {
			// 构建新字符串
			newString := baseString[:i] + string(char) + baseString[i:]
			if !p.ns.IsNameExist(newString) {
				_names = append(_names, newString)
			}
		}
	}
}

// generateSpecialStrings 生成包含固定子串 5201314，且其他数字都是同一个数（不包括 4）的字符串
func (p *IndexerMgr) generate520Strings(suffix string, size int) {
	// 定义允许的单个字符集合（不包括 '4'）
	allowedChars := []byte{'0', '1', '2', '3', '4', '5', '6', '7', '8', '9'}

	// 固定子串
	fixedSubstring := "5201314"

	// 计算固定子串的长度
	fixedLength := len(fixedSubstring)

	// 遍历允许的其他数字
	for _, char := range allowedChars {
		// 生成剩余的字符
		remainingChar := string(char)
		remainingStr := strings.Repeat(remainingChar, size-fixedLength)

		// 在可能的位置插入固定子串
		for i := 0; i <= size-fixedLength; i++ {
			// 构建新字符串
			newString := remainingStr[:i] + fixedSubstring + remainingStr[i:]
			if !p.ns.IsNameExist(newString + suffix) {
				_names = append(_names, newString+suffix)
			}
		}
	}
}

// generateSpecialStrings 生成所有长度为 10 且只能包含 6 和 8 的字符串，并且 6 和 8 分别连接在一起
func (p *IndexerMgr) generateC1C2Strings(c1, c2, suffix string, length int) {
	// 长度为 10 的字符串
	//length := 10

	// 遍历可能的 6 的数量，从 0 到 10
	for count6 := 0; count6 <= length; count6++ {
		count8 := length - count6
		// 构建新字符串
		newString := strings.Repeat(c1, count6) + strings.Repeat(c2, count8)
		if !p.ns.IsNameExist(newString + suffix) {
			_names = append(_names, newString+suffix)
		}
		newString = strings.Repeat(c2, count8) + strings.Repeat(c1, count6)
		if !p.ns.IsNameExist(newString + suffix) {
			_names = append(_names, newString+suffix)
		}
	}
}

// generateIncreasingStrings 生成所有长度为 10 的循环递增字符串
func (p *IndexerMgr) generateIncreasingStrings(base string, size int) {
	//base := "0123456789"
	// 循环生成从 0 到 9 开始的字符串
	for i := 0; i < size; i++ {
		incrementalString := base[i:] + base[:i]
		if !p.ns.IsNameExist(incrementalString) {
			_names = append(_names, incrementalString)
		}
	}
}

// generateDateStrings 生成从开始年份到结束年份的所有日期字符串，格式为 yyyymmdd
func (p *IndexerMgr) generate8BDateStrings(startYear, endYear int) {
	// 定义日期格式
	dateFormat := "20060102"

	// 遍历年份
	for year := startYear; year <= endYear; year++ {
		// 遍历月份
		for month := 1; month <= 12; month++ {
			// 遍历天数
			for day := 1; day <= 31; day++ {
				// 创建日期对象
				date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
				// 检查日期是否合法
				if date.Year() == year && int(date.Month()) == month && date.Day() == day {
					// 输出格式化日期字符串
					str := date.Format(dateFormat)
					if !p.ns.IsNameExist(str + ".btc") {
						_names = append(_names, str+".btc")
					}
				}
			}
		}
	}
}

// generateDateStrings 生成从开始年份到结束年份的所有日期字符串，格式为 yyyymmdd
func (p *IndexerMgr) generate6BDateStrings(startYear, endYear int) {
	// 定义日期格式
	dateFormat := "060102"

	// 遍历年份
	for year := startYear; year <= endYear; year++ {
		// 遍历月份
		for month := 1; month <= 12; month++ {
			// 遍历天数
			for day := 1; day <= 31; day++ {
				// 创建日期对象
				date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)
				// 检查日期是否合法
				if date.Year() == year && int(date.Month()) == month && date.Day() == day {
					// 输出格式化日期字符串
					str := date.Format(dateFormat)
					if !p.ns.IsNameExist(str + ".btc") {
						_names = append(_names, str+".btc")
					}
				}
			}
		}
	}
}

// 统计名字
func (p *IndexerMgr) nameDBStatistic() bool {

	common.Log.Infof("stats: %v", p.ns.GetStatus())

	satsInT1 := make(map[int64]bool, 0)
	namesInT1 := make(map[string]map[string]bool, 0)

	startTime2 := time.Now()
	common.Log.Infof("calculating in %s table ...", ns.DB_PREFIX_NAME)
	p.nsDB.BatchRead([]byte(ns.DB_PREFIX_NAME), false, func(k, v []byte) error {

		var nameValue ns.NameValueInDB
		err := db.DecodeBytesWithProto3(v, &nameValue)
		if err != nil {
			common.Log.Panicf("item.Value error: %v", err)
		}

		satsInT1[nameValue.Sat] = true
		parts := strings.Split(nameValue.Name, ".")
		var key, val string
		if len(parts) == 2 {
			key = parts[1]
			val = parts[0]
		} else if len(parts) == 1 {
			key = "."
			val = nameValue.Name
		} else {
			common.Log.Infof("wrong format %s", nameValue.Name)
		}
		m, ok := namesInT1[key]
		if ok {
			m[val] = true
		} else {
			m := make(map[string]bool)
			m[val] = true
			namesInT1[key] = m
		}

		return nil
	})
	common.Log.Infof("%s table takes %v", ns.DB_PREFIX_NAME, time.Since(startTime2))

	startTime2 = time.Now()
	addrInT1 := make(map[uint64]int, 0)
	p.nftDB.View(func(txn common.ReadBatch) error {
		for k := range satsInT1 {
			var value common.NftsInSat
			key := nft.GetSatKey(k)
			err := db.GetValueFromTxnWithProto3([]byte(key), txn, &value)
			if err == nil {
				addrInT1[value.OwnerAddressId] += len(value.Nfts)
			}
		}
		return nil
	})

	common.Log.Infof("get address %d takes %v", len(addrInT1), time.Since(startTime2))

	addrTreeMap := treemap.NewWith(utils.IntComparator)
	for k, v := range addrInT1 {
		addrTreeMap.Put(v, k)
	}

	suffixTreeMap := treemap.NewWith(utils.IntComparator)
	for k, v := range namesInT1 {
		suffixTreeMap.Put(len(v), k)
	}

	filepath := "./names.txt"
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Log.Infof("OpenFile failed %v", err)
		return true
	}

	file.WriteString(fmt.Sprintf("status: %v\n", p.ns.GetStatus()))
	file.WriteString(fmt.Sprintf("total address: %d\n", len(addrInT1)))
	file.WriteString(fmt.Sprintf("total suffix: %d\n\n", len(namesInT1)))

	count := 100
	total := int(0)
	it := addrTreeMap.Iterator()
	it.End()
	for it.Prev() && count > 0 {
		addressId := it.Value().(uint64)
		amount := it.Key().(int)
		total += (amount)
		file.WriteString(fmt.Sprintf("%s: %d\n", p.GetAddressById(addressId), amount))
		count--
	}
	file.WriteString(fmt.Sprintf("Top 100 addresses have %d names\n\n", total))

	count = 100
	total = int(0)
	it = suffixTreeMap.Iterator()
	it.End()
	for it.Prev() && count > 0 {
		suffix := it.Value().(string)
		amount := it.Key().(int)
		total += int(amount)
		file.WriteString(fmt.Sprintf("%s: %d\n", suffix, amount))
		count--
	}
	file.WriteString(fmt.Sprintf("Top 100 suffix have %d names\n", total))

	file.Close()

	return true
}

func listMnemonicWords() []string {
	// BIP39 英文词表
	wordList := `abandon ability able about above absent absorb abstract absurd abuse access accident account accuse achieve acid acoustic acquire across act action actor actress actual adapt add addict address adjust admit adult advance advice aerobic affair afford afraid again age agent agree ahead aim air airport aisle alarm album alcohol alert alien all alley allow almost alone alpha already also alter always amateur amazing among amount amused analyst anchor ancient anger angle angry animal ankle announce annual another answer antenna antique anxiety any apart apology appear apple approve april arch arctic area arena argue arm armed armor army around arrange arrest arrive arrow art artefact artist artwork ask aspect assault asset assist assume asthma athlete atom attack attend attitude attract auction audit august aunt author auto autumn average avocado avoid awake aware away awesome awful awkward axis baby bachelor bacon badge bag balance balcony ball bamboo banana banner bar barely bargain barrel base basic basket battle beach bean beauty because become beef before begin behave behind believe below belt bench benefit best betray better between beyond bicycle bid bike bind biology bird birth bitter black blade blame blanket blast bleak bless blind blood blossom blouse blue blur blush board boat body boil bomb bone bonus book boost border boring borrow boss bottom bounce box boy bracket brain brand brass brave bread breeze brick bridge brief bright bring brisk broccoli broken bronze broom brother brown brush bubble buddy budget buffalo build bulb bulk bullet bundle bunker burden burger burst bus business busy butter buyer buzz cabbage cabin cable cactus cage cake call calm camera camp can canal cancel candy cannon canoe canvas canyon capable capital captain car carbon card cargo carpet carry cart case cash casino castle casual cat catalog catch category cattle caught cause caution cave ceiling celery cement census century cereal certain chair chalk champion change chaos chapter charge chase chat cheap check cheese chef cherry chest chicken chief child chimney choice choose chronic chuckle chunk churn cigar cinnamon circle citizen city civil claim clap clarify claw clay clean clerk clever click client cliff climb clinic clip clock clog close cloth cloud clown club clump cluster clutch coach coast coconut code coffee coil coin collect color column combine come comfort comic common company concert conduct confirm congress connect consider control convince cook cool copper copy coral core corn correct cost cotton couch country couple course cousin cover coyote crack cradle craft cram crane crash crater crawl crazy cream credit creek crew cricket crime crisp critic crop cross crouch crowd crucial cruel cruise crumble crunch crush cry crystal cube culture cup cupboard curious current curtain curve cushion custom cute cycle dad damage damp dance danger daring dash daughter dawn day deal debate debris decade december decide decline decorate decrease deer defense define defy degree delay deliver demand demise denial dentist deny depart depend deposit depth deputy derive describe desert design desk despair destroy detail detect develop device devote diagram dial diamond diary dice diesel diet differ digital dignity dilemma dinner dinosaur direct dirt disagree discover disease dish dismiss disorder display distance divert divide divorce dizzy doctor document dog doll dolphin domain donate donkey donor door dose double dove draft dragon drama drastic draw dream dress drift drill drink drip drive drop drum dry duck dumb dune during dust dutch duty dwarf dynamic eager eagle early earn earth easily east easy echo ecology economy edge edit educate effort egg eight either elbow elder electric elegant element elephant elevator elite else embark embody embrace emerge emotion employ empower empty enable enact end endless endorse enemy energy enforce engage engine enhance enjoy enlist enough enrich enroll ensure enter entire entry envelope episode equal equip era erase erode erosion error erupt escape essay essence estate eternal ethics evidence evil evoke evolve exact example excess exchange excite exclude excuse execute exercise exhaust exhibit exile exist exit exotic expand expect expire explain expose express extend extra eye eyebrow fabric face faculty fade faint faith fall false fame family famous fan fancy fantasy farm fashion fat fatal father fatigue fault favorite feature february federal fee feed feel female fence festival fetch fever few fiber fiction field figure file film filter final find fine finger finish fire firm first fiscal fish fit fitness fix flag flame flash flat flavor flee flight flip float flock floor flower fluid flush fly foam focus fog foil fold follow food foot force forest forget fork fortune forum forward fossil foster found fox fragile frame frequent fresh friend fringe frog front frost frown frozen fruit fuel fun funny furnace fury future gadget gain galaxy gallery game gap garage garbage garden garlic garment gas gasp gate gather gauge gaze general genius genre gentle genuine gesture ghost giant gift giggle ginger giraffe girl give glad glance glare glass glide glimpse globe gloom glory glove glow glue goat goddess gold good goose gorilla gospel gossip govern gown grab grace grain grant grape grass gravity great green grid grief grit grocery group grow grunt guard guess guide guilt guitar gun gym habit hair half hammer hamster hand happy harbor hard harsh harvest hat have hawk hazard head health heart heavy hedgehog height hello helmet help hen hero hidden high hill hint hip hire history hobby hockey hold hole holiday hollow home honey hood hope horn horror horse hospital host hotel hour hover hub huge human humble humor hundred hungry hunt hurdle hurry hurt husband hybrid ice icon idea identify idle ignore ill illegal illness image imitate immense immune impact impose improve impulse inch include income increase index indicate indoor industry infant inflict inform inhale inherit initial inject injury inmate inner innocent input inquiry insane insect inside inspire install intact interest into invest invite involve iron island isolate issue item ivory jacket jaguar jar jazz jealous jeans jelly jewel job join joke journey joy judge juice jump jungle junior junk just kangaroo keen keep ketchup key kick kid kidney kind kingdom kiss kit kitchen kite kitten kiwi knee knife knock know lab label labor ladder lady lake lamp language laptop large later latin laugh laundry lava law lawn lawsuit layer lazy leader leaf learn leave lecture left leg legal legend leisure lemon lend length lens leopard lesson letter level liar liberty library license life lift light like limb limit link lion liquid list little live lizard load loan lobster local lock logic lonely long loop lottery loud lounge love loyal lucky luggage lumber lunar lunch luxury lyrics machine mad magic magnet maid mail main major make mammal man manage mandate mango mansion manual maple marble march margin marine market marriage mask mass master match material math matrix matter maximum maze meadow mean measure meat mechanic medal media melody melt member memory mention menu mercy merge merit merry mesh message metal method middle midnight milk million mimic mind minimum minor minute miracle mirror misery miss mistake mix mixed mixture mobile model modify mom moment monitor monkey monster month moon moral more morning mosquito mother motion motor mountain mouse move movie much muffin mule multiply muscle museum mushroom music must mutual myself mystery myth naive name napkin narrow nasty nation nature near neck need negative neglect neither nephew nerve nest net network neutral never news next nice night noble noise nominee noodle normal north nose notable note nothing notice novel now nuclear number nurse nut oak obey object oblige obscure observe obtain obvious occur ocean october odor off offer office often oil okay old olive olympic omit once one onion online only open opera opinion oppose option orange orbit orchard order ordinary organ orient original orphan ostrich other outdoor outer output outside oval oven over own owner oxygen oyster ozone pact paddle page pair palace palm panda panel panic panther paper parade parent park parrot party pass patch path patient patrol pattern pause pave payment peace peanut pear peasant pelican pen penalty pencil people pepper perfect permit person pet phone photo phrase physical piano picnic picture piece pig pigeon pill pilot pink pioneer pipe pistol pitch pizza place planet plastic plate play please pledge pluck plug plunge poem poet point polar pole police pond pony pool popular portion position possible post potato pottery poverty powder power practice praise predict prefer prepare present pretty prevent price pride primary print priority prison private prize problem process produce profit program project promote proof property prosper protect proud provide public pudding pull pulp pulse pumpkin punch pupil puppy purchase purity purpose purse push put puzzle pyramid quality quantum quarter question quick quit quiz quote rabbit raccoon race rack radar radio rail rain raise rally ramp ranch random range rapid rare rate rather raven raw razor ready real reason rebel rebuild recall receive recipe record recycle reduce reflect reform refuse region regret regular reject relax release relief rely remain remember remind remove render renew rent reopen repair repeat replace report require rescue resemble resist resource response result retire retreat return reunion reveal review reward rhythm rib ribbon rice rich ride ridge rifle right rigid ring riot ripple risk ritual rival river road roast robot robust rocket romance roof rookie room rose rotate rough round route royal rubber rude rug rule run runway rural sad saddle safari safe salad salmon salon salt salute same sample sand satisfy satoshi sauce sausage save say scale scan scare scatter scene scheme school science scissors scorpion scout scrap screen script scrub sea search season seat second secret section security seed seek segment select sell seminar senior sense sentence series service session settle setup seven shadow shaft shallow share shed shell sheriff shield shift shine ship shiver shock shoe shoot shop short shoulder shove shrimp shrug shuffle shy sibling sick side siege sight sign silent silk silly silver similar simple since sing siren sister situate six size skate sketch ski skill skin skirt skull slab slam sleep slender slice slide slight slim slogan slot slow slush small smart smile smoke smooth snack snake snap sniff snow soap soccer social sock soda soft solar soldier solid solution solve someone song soon sorry sort soul sound soup source south space spare spatial spawn speak special speed spell spend sphere spice spider spike spin spirit split spoil sponsor spoon sport spot spray spread spring spy square squeeze squirrel stable stadium staff stage stairs stamp stand start state stay steak steel stem step stereo stick still sting stock stomach stone stool story stove strategy street strike strong struggle student stuff stumble style subject submit subway success such sudden suffer sugar suggest suit summer sun sunny sunset super supply supreme sure surface surge surprise surround survey suspect sustain swallow swamp swap swarm swear sweet swift swim swing switch sword symbol symptom syrup system table tackle tag tail talent talk tank tape target task taste tattoo taxi teach team tell ten tenant tennis tent term test text thank that theme then theory there they thing this thought three thrive throw thumb thunder ticket tide tiger tilt timber time tiny tip tired tissue title toast tobacco today toddler toe together toilet token tomato tomorrow tone tongue tonight tool tooth top topic topple torch tornado tortoise toss total tourist toward tower town toy track trade traffic tragic train transfer trap trash travel tray treat tree trend trial tribe trick trigger trim trip trophy trouble truck true truly trumpet trust truth try tube tuition tumble tuna tunnel turkey turn turtle twelve twenty twice twin twist type typical ugly umbrella unable unaware uncle uncover under undo unfair unfold unhappy uniform unique unit universe unknown unlock until unusual unveil update upgrade uphold upon upper upset urban urge usage use used useful useless usual utility vacant vacuum vague valid valley valve van vanish vapor various vast vault vehicle velvet vendor venture venue verb verify version very vessel veteran viable vibrant vicious victory video view village vintage violin virtual virus visa visit visual vital vivid vocal voice void volcano volume vote voyage wage wagon wait walk wall walnut want warfare warm warrior wash wasp waste water wave way wealth weapon wear weasel weather web wedding weekend weird welcome west wet whale what wheat wheel when where whip whisper wide width wife wild will win window wine wing wink winner winter wire wisdom wise wish witness wolf woman wonder wood wool word work world worry worth wrap wreck wrestle wrist write wrong yard year yellow you young youth zebra zero zone zoo`

	// 将字符串分割成单词列表
	words := strings.Fields(wordList)

	return words
}

func (p *IndexerMgr) subNameStatistic(sub string) bool {
	startTime2 := time.Now()
	common.Log.Infof("calculating in %s table ...", ns.DB_PREFIX_NAME)
	nameMap := make(map[string]int64)
	p.nsDB.BatchRead([]byte(ns.DB_PREFIX_NAME), false, func(k, v []byte) error {

		var value ns.NameValueInDB
		err := db.DecodeBytesWithProto3(v, &value)
		if err != nil {
			common.Log.Errorf("item.Value error: %v", err)
			return nil
		}

		if sub != "" {
			if strings.HasSuffix(value.Name, sub) {
				nameMap[value.Name] = value.Sat
			}
		} else {
			if !strings.Contains(value.Name, ".") {
				nameMap[value.Name] = value.Sat
			}
		}

		return nil
	})
	common.Log.Infof("%s table takes %v", ns.DB_PREFIX_NAME, time.Since(startTime2))

	startTime := time.Now()
	common.Log.Infof("get all addresses ...")
	addrIdMap := make(map[uint64]int)
	p.nftDB.View(func(txn common.ReadBatch) error {
		for _, v := range nameMap {
			key := nft.GetSatKey(v)
			nfts := &common.NftsInSat{}
			err := db.GetValueFromTxnWithProto3([]byte(key), txn, nfts)
			if err == nil {
				addrIdMap[nfts.OwnerAddressId]++
			}
		}
		return nil
	})
	common.Log.Infof("name count %d takes %v", len(nameMap), time.Since(startTime))

	type AddressAndCount struct {
		Address string
		Count   int
	}

	startTime = time.Now()
	addrVect := make([]*AddressAndCount, 0)
	for k, v := range addrIdMap {
		addr := p.GetAddressById(k)
		if addr != "" {
			addrVect = append(addrVect, &AddressAndCount{Address: addr, Count: v})
		}
	}
	common.Log.Infof("address count %d takes %v", len(addrIdMap), time.Since(startTime))

	sort.Slice(addrVect, func(i, j int) bool {
		return addrVect[i].Count > addrVect[j].Count
	})

	filepath := "./statistic" + sub + ".txt"
	os.Remove(filepath)
	file, err := os.OpenFile(filepath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0644)
	if err != nil {
		common.Log.Infof("OpenFile failed %v", err)
		return true
	}

	file.WriteString(fmt.Sprintf("statistic: address=%d name=%d (height %d)\n", len(addrVect), len(nameMap), p.base.GetHeight()))
	for _, v := range addrVect {
		str := fmt.Sprintf("%s : %d\n", v.Address, v.Count)
		_, err = file.WriteString(str)
		if err != nil {
			common.Log.Infof("WriteString failed %v", err)
			break
		}
	}

	file.Close()

	return true
}

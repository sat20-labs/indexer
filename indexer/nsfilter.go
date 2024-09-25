package indexer

import (
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"
)

func filterName(name string, filterv []string) bool {
	if name == "" {
		return false
	}

	for _, f := range filterv {
		if f == "1Han" || f == "2Han" || f == "3Han" || f == "4Han" {
			expectedL, _ := extractLeadingNumber(f)
			ok, l := isChinese(name)
			if ok && l == expectedL {
				continue
			}
			return false
		}

		if f == "1D" || f == "2D" || f == "3D" || f == "4D" || f == "5D" ||
		f == "6D" || f == "7D" || f == "8D" || f == "9D" || f == "10D" ||
		f == "11D" || f == "12D" {
			expectedL, _ := extractLeadingNumber(f)
			if len(name) == expectedL && isDigit(name) {
				continue
			}
			return false
		}

		if f == "1L" || f == "2L" || f == "3L" || f == "4L" {
			expectedL, _ := extractLeadingNumber(f)
			if len(name) == expectedL && isAlphabet(name) {
				continue
			}
			return false
		}

		if f == "cmn" {
			if isCMobileNumber(name) {
				continue
			}
			return false
		}
		
		if f == "SDate" {// YYMMDD
			if isDateDigitName(name) {
				continue
			}
			return false
		}

		if f == "FDate" {// YYYYMMDD
			if isFullDateDigitName(name) {
				continue
			}
			return false
		}

		if f == "cvcv" {
			if isCVCV(name) {
				continue
			}
			return false
		}

		if f == "same" {
			if isSameCharName(name) {
				continue
			}
			return false
		}

		if f == "consecutive" {
			if isConsecutiveDigit(name) {
				continue
			}
			return false
		}

		if f == "symmetric" {
			if isSymmetricNumber(name) {
				continue
			}
			return false
		}

		if f == "lucky" {
			if isConsecutiveLuckDigit(name) {
				continue
			}
			return false
		}

		if f == "2char" {
			if is2CharName(name) {
				continue
			}
			return false
		}

		if f == "DaL2" {
			if bothDigitAndLetter(name, 2) {
				continue
			}
			return false
		}

		if f == "DaL3" {
			if bothDigitAndLetter(name, 3) {
				continue
			}
			return false
		}
	}
	
	return true
}

func bothDigitAndLetter(s string, lenght int) bool {
    if len(s) != lenght {
        return false
    }

    hasLetter := false
    hasDigit := false

    for _, char := range s {
        if unicode.IsLetter(char) {
            hasLetter = true
        } else if unicode.IsDigit(char) {
            hasDigit = true
        }

        if hasLetter && hasDigit {
            return true
        }
    }

    return hasLetter && hasDigit
}

func isAlphabet(s string) bool {
	// 正则表达式匹配英文字母
	pattern := `^[A-Za-z]+$`

	// 编译正则表达式
	re := regexp.MustCompile(pattern)

	// 判断字符串是否匹配
	return re.MatchString(s)
}

func isDigit(s string) bool {
	match, _ := regexp.MatchString(`^[0-9]+$`, s)
	return match
}

func isChinese(s string) (bool, int) {
	number := 0
	for len(s) > 0 {
		r, size := utf8.DecodeRuneInString(s)
		if !unicode.Is(unicode.Han, r) {
			return false, number
		}
		s = s[size:]
		number++
	}
	return true, number
}

// isSameCharName
// category name: "repeat" , eg: "000000", "aaaaaa"
func isSameCharName(name string) bool {
	char := name[0]
	for i := 0; i < len(name); i++ {
		if name[i] != char {
			return false
		}
	}
	return true
}

// isSameCharName
// category name: "2Number" , eg: "000000", "aaaaaa"
func is2CharName(name string) bool {
	char := name[0]
	char2 := char
	for i := 0; i < len(name); i++ {
		if name[i] != char {
			if char2 == char {
				char2 = name[i]
				continue
			}

			if char2 != name[i] {
				return false
			}
		}
	}
	return char2 != char
}

// isDateDigitName
// category name: "date" , eg: "890719", "220420"
func isDateDigitName(name string) bool {
	length := len(name)
	if length != 6 {
		return false
	}

	// yymmdd
	year, err := strconv.Atoi(string(name[0:2]))
	if err != nil {
		return false
	}
	month, err := strconv.Atoi(string(name[2:4]))
	if err != nil {
		return false
	}
	day, err := strconv.Atoi(string(name[4:]))
	if err != nil {
		return false
	}
	if year < 24 {
		year += 2000
	} else {
		year += 1900
	}

	if isValidDate(year, month, day) {
		return true
	}
	// yyyymm
	year, err = strconv.Atoi(string(name[0:4]))
	if err != nil {
		return false
	}
	month, err = strconv.Atoi(string(name[4:]))
	if err != nil {
		return false
	}

	if isValidDate(year, month, 1) {
		return true
	}

	return false

}

// isFullDateDigitName
// category name: "fulldate" , eg: "19870601", "20230101"
func isFullDateDigitName(name string) bool {
	length := len(name)
	if length != 8 {
		return false
	}
	year, err := strconv.Atoi(string(name[0:4]))
	if err != nil {
		return false
	}
	month, err := strconv.Atoi(string(name[4:6]))
	if err != nil {
		return false
	}
	day, err := strconv.Atoi(string(name[6:]))
	if err != nil {
		return false
	}
	return isValidDate(year, month, day)
}
func isValidDate(year, month, day int) bool {
	if year < 1900 || year > 2100 {
		return false
	}
	// 尝试创建一个日期对象
	date := time.Date(year, time.Month(month), day, 0, 0, 0, 0, time.UTC)

	// 检查生成的日期的年、月、日是否与输入的相同
	return date.Year() == year && int(date.Month()) == month && date.Day() == day
}

// isConsecutiveDigit
// category name: "ConsecutiveNumber" , eg: "123456", "987654"
func isConsecutiveDigit(name string) bool {
	if len(name) < 2 {
		return false
	}
	interval := int(name[1] - name[0])
	if interval != 1 && interval != -1 {
		return false
	}

	for i := 0; i < len(name)-1; i++ {
		if int(name[i+1]-name[i]) != interval {
			return false
		}
	}
	return true
}

// isCMobileNumber
// category name: "CMobileNumber" , eg: "13134560987", "18931334556"
// const MobileReg = "^1[3456789]\\d{9}$"
func isCMobileNumber(number string) bool {
	pattern := `^1[3-9]\d{9}$`

	// 编译正则表达式
	re := regexp.MustCompile(pattern)

	// 使用正则表达式进行匹配
	return re.MatchString(number)
}

// isSymmetricNumber
// category name: "SymmetricNumber" , eg: "123321", "12321"
//const MobileReg = "^1[3456789]\\d{9}$"

func isSymmetricNumber(number string) bool {
	length := len(number)
	if length < 2 {
		return true
	}

	midLength := length / 2
	// 前后数据是否一致
	for i := 0; i < midLength; i++ {
		if number[i] != number[length-1-i] {
			return false
		}
	}
	return true
}

// isConsecutiveLuckDigit
// category name: "ConsecutiveLuckDigit" , eg: "888***", "**666*"
// const MobileReg = "^1[3456789]\\d{9}$"

func isConsecutiveLuckDigit(number string) bool {
	count := 0 // 连续数字个数
	var currentLuckNumber byte
	currentLuckNumber = 0
	for i := 0; i < len(number); i++ {
		if number[i] == currentLuckNumber {
			count++
			if count >= 3 {
				return true
			}
			continue
		}
		if isLuckDigit(number[i]) {
			// 当前数字是LuckNumber
			currentLuckNumber = number[i]
			count = 1
			continue
		}
		currentLuckNumber = 0
		count = 0
	}

	return false
}
func isLuckDigit(number byte) bool {
	if number == '6' || number == '8' || number == '9' {
		return true
	}

	return false
}

func isCVCV(name string) bool {
	length := len(name)
	if length != 4 {
		return false
	}
	for index, letter := range name {
		if index%2 == 0 {
			// 偶数位 (0, 2) 是辅音
			if isVowel(letter) {
				return false
			}
			continue
		}
		// 奇数位（1, 3）是元音
		if !isVowel(letter) {
			return false
		}
	}
	return true
}

func isVowel(r rune) bool {
	// 将字符转换为小写
	lower := unicode.ToLower(r)

	// 检查是否为元音字母
	return strings.ContainsRune("aeiou", lower)
}

func extractLeadingNumber(input string) (int, error) {
    if len(input) == 0 {
        return 0, nil
    }

    var numStr string
    for _, char := range input {
        if unicode.IsDigit(char) {
            numStr += string(char)
        } else {
            break
        }
    }

    if numStr == "" {
        return 0, nil
    }

    return strconv.Atoi(numStr)
}

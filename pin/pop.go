package pin

import (
	"fmt"
	"manindexer/common"
	"math"
	"math/big"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

func PopLevelCount(chainName, pop string) (lv int, lastStr string) {
	PopCutNum := 1000
	switch chainName {
	case "btc":
		PopCutNum = common.Config.Btc.PopCutNum
	case "mvc":
		PopCutNum = common.Config.Mvc.PopCutNum
	}
	if len(pop) < PopCutNum {
		lv = -1
		lastStr = pop
		return
	}
	cnt := len(pop) - len(strings.TrimLeft(pop, "0"))
	if cnt <= PopCutNum {
		lv = -1
		lastStr = pop[PopCutNum:]
		return
	} else {
		lv = cnt - PopCutNum
		lastStr = pop[PopCutNum:]
		return
	}
}
func RarityScoreBinary(chainName, binaryStr string) int {
	popCutNum := 0
	switch chainName {
	case "btc":
		popCutNum = common.Config.Btc.PopCutNum
	case "mvc":
		popCutNum = common.Config.Mvc.PopCutNum
	}
	if len(binaryStr) < popCutNum {
		return 0
	}
	binaryStr = binaryStr[popCutNum:]
	// Step 1: Count the number of leading zeros
	n := len(binaryStr) - len(strings.TrimLeft(binaryStr, "0"))

	// Step 2: Remove leading zeros and calculate the decimal value of the rest part
	restPart := strings.TrimLeft(binaryStr, "0")
	if restPart == "" {
		// In case the binary string is all zeros
		return int(math.Pow(2, float64(n)))
	}

	//fmt.Println("rest:", restPart)
	// restValue, err := strconv.ParseInt(restPart, 2, 64)
	// if err != nil {
	// 	fmt.Printf("Error parsing binary string: %v\n", err)
	// 	return 0
	// }
	// k := len(restPart)
	// // Step 3: Normalize the rest value and invert it
	// normalizedValue := (1 - (float64(restValue)+1)/math.Pow(2, float64(k))) * 2
	bigInt := new(big.Int)
	bigInt.SetString(restPart, 10)
	base := new(big.Int)
	max := int64(170 - popCutNum)
	base.Exp(big.NewInt(10), big.NewInt(max), nil)
	bigFloat := new(big.Float).SetInt(bigInt)
	baseFloat := new(big.Float).SetInt(base)
	normalizedFloat := new(big.Float).Quo(bigFloat, baseFloat)
	normalizedValue, _ := normalizedFloat.Float64()
	// Step 4: Calculate the final score
	score := math.Pow(2, float64(n)) + normalizedValue*math.Pow(2, float64(n))
	// Step 5: Round the score to the nearest integer
	return int(math.Round(score))
}

func GetPoPScore(popStr string, lvNum int64, extractCount int) decimal.Decimal {
	// 计算前面0的个数
	zeroCount := 0
	for i := 0; i < len(popStr); i++ {
		if popStr[i] == '0' {
			zeroCount++
		} else {
			break
		}
	}
	// 如果0的个数少于extractCount，直接返回0
	if zeroCount < extractCount {
		//result := 0.0
		return decimal.NewFromInt(0)
	}

	// 1. 裁剪掉前面所有的0
	popSub := strings.TrimLeft(popStr, "0")

	// 2. 取前4位，构造小数
	popSub4 := popSub
	if len(popSub) > 4 {
		popSub4 = popSub[:4]
	}
	popDecStr := "0." + popSub4

	// 3. 八进制转十进制，使用octalFractionToUniformDecimal
	popDecFloat := octalFractionToUniformDecimal(popDecStr)

	// 4. lvNum + (1 - popDec)
	one := new(big.Float).SetFloat64(1)
	oneMinusPopDec := new(big.Float).Sub(one, new(big.Float).SetFloat64(popDecFloat))
	popLevelDecimal := new(big.Float).Add(new(big.Float).SetInt64(lvNum), oneMinusPopDec)
	popLevelDecimalFloat, _ := popLevelDecimal.Float64()

	// 5. 计算8的PoPLevelDecimal次方
	ln8 := math.Log(8)
	exponent := popLevelDecimalFloat * ln8
	result := math.Exp(exponent)

	// 保留4位小数，直接向下取整
	result = math.Floor(result*10000) / 10000
	return decimal.NewFromFloat(result)
}

// 将八进制小数（如"2152"）转为[0,1]区间均匀分布的十进制小数
func octalFractionToUniformDecimal(octalStr string) float64 {
	if len(octalStr) > 2 && octalStr[:2] == "0." {
		octalStr = octalStr[2:]
	}
	x := 0.0
	for i, c := range octalStr {
		digit := float64(c - '0')
		pow := math.Pow(8, float64(i+1))
		add := digit / pow
		x += add
	}

	y := (x - 1.0/8.0) * 8.0 / 7.0
	return y
}

func GetPoPScoreV1(pop string, popLv int) decimal.Decimal {
	lv := int64(popLv)
	if lv <= 0 {
		lv = int64(1)
	}
	dv, _ := OctalStringToDecimal(pop, 4, 10000)
	dvDecimal := decimal.Zero
	if dv != nil {
		dvDecimal = decimal.NewFromFloat(*dv)
	}
	value := decimal.NewFromInt(1 * 8)
	return decimal.NewFromInt(lv).Mul(value).Add(dvDecimal)
}
func OctalStringToDecimal(octalStr string, intNum int, divisor float64) (*float64, error) {
	decimalNum := new(big.Int)
	base := big.NewInt(8)
	for _, char := range octalStr {
		digit := int64(char - '0')
		if digit < 0 || digit > 7 {
			return nil, fmt.Errorf("err: %c", char)
		}

		decimalNum.Mul(decimalNum, base)
		decimalNum.Add(decimalNum, big.NewInt(digit))
	}
	bigIntStrFull := decimalNum.String()
	bingIntStr := ""
	if len(bigIntStrFull) > intNum {
		bingIntStr = bigIntStrFull[:intNum]
	} else {
		bingIntStr = bigIntStrFull
	}
	firstFourInt, err := strconv.Atoi(bingIntStr)
	if err != nil {
		return nil, err
	}
	result := float64(firstFourInt) / divisor
	//rounded := math.Round(result*10000) / 10000
	return &result, nil
}

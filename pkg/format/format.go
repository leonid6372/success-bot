package format

import (
	"fmt"
	"strings"

	"github.com/leonid6372/success-bot/pkg/log"
	"go.uber.org/zap"
)

func PrettyNumber(number any, separator, decimalSeparator string, originalDecimals bool) string {
	var numStr string
	isNegative := false

	switch number.(type) {
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		numStr = fmt.Sprintf("%d", number)
	case float32, float64:
		if originalDecimals {
			n := decimalsCount(number)
			numStr = fmt.Sprintf("%.*f", n, number)
		} else {
			numStr = fmt.Sprintf("%.2f", number)
		}
	default:
		log.Error("PrettyNumber: unsupported type",
			zap.Any("value", number),
			zap.String("type", fmt.Sprintf("%T", number)),
		)

		return fmt.Sprint(number)
	}

	if separator == "" && decimalSeparator == "" {
		return numStr
	}

	if separator == "" {
		log.Warn("PrettyNumber: separator is empty")
	}

	if decimalSeparator == "" {
		log.Warn("PrettyNumber: decimalSeparator is empty")
	}

	if separator == decimalSeparator {
		log.Warn("PrettyNumber: separator and decimalSeparator are the same", zap.String("value", separator))
	}

	if strings.HasPrefix(numStr, "-") {
		isNegative = true
		numStr = strings.TrimPrefix(numStr, "-")
	}

	parts := strings.Split(numStr, ".")
	integerPart := parts[0]
	decimalPart := ""
	if len(parts) == 2 {
		decimalPart = decimalSeparator + parts[1]
	}

	length := len(integerPart)

	start := length % 3
	if start == 0 {
		start = 3
	}

	var intPart strings.Builder

	if isNegative {
		intPart.WriteString("-")
	}

	intPart.WriteString(integerPart[:start])

	for i := start; i < length; i += 3 {
		intPart.WriteString(separator)
		intPart.WriteString(integerPart[i : i+3])
	}

	return intPart.String() + decimalPart
}

func decimalsCount(f any) int {
	str := fmt.Sprintf("%f", f)

	str = strings.TrimRight(str, "0")

	if dot := strings.Index(str, "."); dot != -1 {
		return len(str) - dot - 1
	}

	return 0
}

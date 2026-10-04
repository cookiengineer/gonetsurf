package microsyntaxes

import (
	"math"
	"strconv"
)

func isASCIISpace(value byte) bool {
	return value == '\t' || value == '\n' || value == '\f' || value == '\r' || value == ' '
}

func isASCIIDigit(value byte) bool {
	return value >= '0' && value <= '9'
}

func skipSpace(input string, position int) int {
	for position < len(input) && isASCIISpace(input[position]) {
		position++
	}
	return position
}

// ParseInteger implements the rules for parsing integers. It returns the
// integer and whether parsing succeeded.
func ParseInteger(input string) (int, bool) {
	position := skipSpace(input, 0)
	if position >= len(input) {
		return 0, false
	}
	sign := 1
	switch input[position] {
	case '-':
		sign = -1
		position++
	case '+':
		position++
	}
	if position >= len(input) || !isASCIIDigit(input[position]) {
		return 0, false
	}
	start := position
	for position < len(input) && isASCIIDigit(input[position]) {
		position++
	}
	value, err := strconv.Atoi(input[start:position])
	if err != nil {
		return 0, false
	}
	return sign * value, true
}

// ParseNonNegativeInteger implements the rules for parsing non-negative
// integers.
func ParseNonNegativeInteger(input string) (int, bool) {
	value, ok := ParseInteger(input)
	if !ok || value < 0 {
		return 0, false
	}
	return value, true
}

// ParseFloat implements the rules for parsing floating-point number values.
func ParseFloat(input string) (float64, bool) {
	value, ok := scanFloat(input)
	if !ok {
		return 0, false
	}
	if math.IsInf(value, 0) || math.IsNaN(value) {
		return 0, false
	}
	return value, true
}

func scanFloat(input string) (float64, bool) {
	position := skipSpace(input, 0)
	start := position
	if position < len(input) && (input[position] == '-' || input[position] == '+') {
		position++
	}
	digitsStart := position
	for position < len(input) && isASCIIDigit(input[position]) {
		position++
	}
	hasDigits := position > digitsStart

	if position < len(input) && input[position] == '.' {
		fractionStart := position
		position++
		fractionDigits := position
		for position < len(input) && isASCIIDigit(input[position]) {
			position++
		}
		if position > fractionDigits {
			hasDigits = true
		} else {
			position = fractionStart
		}
	}
	if !hasDigits {
		return 0, false
	}

	if position < len(input) && (input[position] == 'e' || input[position] == 'E') {
		exponentStart := position
		position++
		if position < len(input) && (input[position] == '-' || input[position] == '+') {
			position++
		}
		exponentDigits := position
		for position < len(input) && isASCIIDigit(input[position]) {
			position++
		}
		if position == exponentDigits {
			position = exponentStart
		}
	}

	text := input[start:position]
	value, err := strconv.ParseFloat(text, 64)
	if err != nil {
		return 0, false
	}
	return value, true
}

// ParsePercentage implements the rules for parsing percentages.
func ParsePercentage(input string) (float64, bool) {
	value, ok := ParseFloat(trimTrailing(input))
	if !ok {
		return 0, false
	}
	return value, true
}

func trimTrailing(input string) string {
	end := len(input)
	for end > 0 && isASCIISpace(input[end-1]) {
		end--
	}
	return input[:end]
}

// ParseDimension implements the rules for parsing dimension values. It returns
// the non-negative value, whether the value is a percentage, and whether
// parsing succeeded.
func ParseDimension(input string) (value float64, percentage bool, ok bool) {
	position := skipSpace(input, 0)
	if position >= len(input) || !isASCIIDigit(input[position]) {
		return 0, false, false
	}
	integerStart := position
	for position < len(input) && isASCIIDigit(input[position]) {
		position++
	}
	integer, err := strconv.Atoi(input[integerStart:position])
	if err != nil {
		return 0, false, false
	}
	value = float64(integer)
	if position >= len(input) {
		return value, false, true
	}
	if input[position] == '.' {
		position++
		if position >= len(input) || !isASCIIDigit(input[position]) {
			return dimensionResult(value, input, position)
		}
		divisor := 1.0
		for position < len(input) && isASCIIDigit(input[position]) {
			divisor *= 10
			value += float64(input[position]-'0') / divisor
			position++
		}
	}
	return dimensionResult(value, input, position)
}

func dimensionResult(value float64, input string, position int) (float64, bool, bool) {
	if position < len(input) && input[position] == '%' {
		return value, true, true
	}
	return value, false, true
}

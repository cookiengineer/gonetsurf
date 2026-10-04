package microsyntaxes

import (
	"strconv"
	"strings"
)

// Month is a year and month.
type Month struct {
	Year  int
	Month int
}

// Date is a proleptic Gregorian date.
type Date struct {
	Year  int
	Month int
	Day   int
}

// YearlessDate is a month and day without a year.
type YearlessDate struct {
	Month int
	Day   int
}

// Time is a time of day with millisecond precision.
type Time struct {
	Hour        int
	Minute      int
	Second      int
	Millisecond int
}

// LocalDateTime is a date and a time with no time zone.
type LocalDateTime struct {
	Date Date
	Time Time
}

// TimeZoneOffset is a signed offset from UTC in minutes.
type TimeZoneOffset struct {
	Minutes int
}

// GlobalDateTime is a date, time, and time zone offset.
type GlobalDateTime struct {
	LocalDateTime LocalDateTime
	Offset        TimeZoneOffset
}

// Week is an ISO-8601 week date.
type Week struct {
	Year int
	Week int
}

func twoDigits(input string, position *int) (int, bool) {
	pos := *position
	if pos+2 > len(input) || !isASCIIDigit(input[pos]) || !isASCIIDigit(input[pos+1]) {
		return 0, false
	}
	value := int(input[pos]-'0')*10 + int(input[pos+1]-'0')
	*position = pos + 2
	return value, true
}

func parseYear(input string, position *int) (int, bool) {
	pos := *position
	start := pos
	for pos < len(input) && isASCIIDigit(input[pos]) {
		pos++
	}
	if pos-start < 4 {
		return 0, false
	}
	year, err := strconv.Atoi(input[start:pos])
	if err != nil || year < 1 {
		return 0, false
	}
	*position = pos
	return year, true
}

// ParseMonth parses a valid month string (YYYY-MM).
func ParseMonth(input string) (Month, bool) {
	position := 0
	year, ok := parseYear(input, &position)
	if !ok || position >= len(input) || input[position] != '-' {
		return Month{}, false
	}
	position++
	month, ok := twoDigits(input, &position)
	if !ok || month < 1 || month > 12 || position != len(input) {
		return Month{}, false
	}
	return Month{Year: year, Month: month}, true
}

// ParseDate parses a valid date string (YYYY-MM-DD).
func ParseDate(input string) (Date, bool) {
	position := 0
	year, ok := parseYear(input, &position)
	if !ok || position >= len(input) || input[position] != '-' {
		return Date{}, false
	}
	position++
	month, ok := twoDigits(input, &position)
	if !ok || position >= len(input) || input[position] != '-' {
		return Date{}, false
	}
	position++
	day, ok := twoDigits(input, &position)
	if !ok || position != len(input) || month < 1 || month > 12 || day < 1 {
		return Date{}, false
	}
	if day > daysInMonth(year, month) {
		return Date{}, false
	}
	return Date{Year: year, Month: month, Day: day}, true
}

// ParseYearlessDate parses a valid yearless date string (MM-DD).
func ParseYearlessDate(input string) (YearlessDate, bool) {
	position := 0
	month, ok := twoDigits(input, &position)
	if !ok || position >= len(input) || input[position] != '-' {
		return YearlessDate{}, false
	}
	position++
	day, ok := twoDigits(input, &position)
	if !ok || position != len(input) || month < 1 || month > 12 || day < 1 || day > daysInMonth(2000, month) {
		return YearlessDate{}, false
	}
	return YearlessDate{Month: month, Day: day}, true
}

// ParseTime parses a valid time string (HH:MM[:SS[.mmm]]).
func ParseTime(input string) (Time, bool) {
	position := 0
	hour, ok := twoDigits(input, &position)
	if !ok || hour > 23 || position >= len(input) || input[position] != ':' {
		return Time{}, false
	}
	position++
	minute, ok := twoDigits(input, &position)
	if !ok || minute > 59 {
		return Time{}, false
	}
	result := Time{Hour: hour, Minute: minute}
	if position < len(input) && input[position] == ':' {
		position++
		second, ok := twoDigits(input, &position)
		if !ok || second > 59 {
			return Time{}, false
		}
		result.Second = second
		if position < len(input) && input[position] == '.' {
			position++
			start := position
			for position < len(input) && isASCIIDigit(input[position]) {
				position++
			}
			fraction := input[start:position]
			if len(fraction) < 1 || len(fraction) > 3 {
				return Time{}, false
			}
			milliseconds := 0
			for _, character := range fraction {
				milliseconds = milliseconds*10 + int(character-'0')
			}
			for count := len(fraction); count < 3; count++ {
				milliseconds *= 10
			}
			result.Millisecond = milliseconds
		}
	}
	if position != len(input) {
		return Time{}, false
	}
	return result, true
}

// ParseLocalDateTime parses a local date and time string.
func ParseLocalDateTime(input string) (LocalDateTime, bool) {
	for index := 0; index < len(input); index++ {
		if input[index] != 'T' && input[index] != ' ' {
			continue
		}
		date, ok := ParseDate(input[:index])
		if !ok {
			return LocalDateTime{}, false
		}
		clock, ok := ParseTime(input[index+1:])
		if !ok {
			return LocalDateTime{}, false
		}
		return LocalDateTime{Date: date, Time: clock}, true
	}
	return LocalDateTime{}, false
}

// ParseTimeZoneOffset parses a time-zone offset string ("Z" or "+HH:MM").
func ParseTimeZoneOffset(input string) (TimeZoneOffset, bool) {
	if input == "Z" || input == "z" {
		return TimeZoneOffset{Minutes: 0}, true
	}
	if len(input) != 6 || (input[0] != '+' && input[0] != '-') || input[3] != ':' {
		return TimeZoneOffset{}, false
	}
	position := 1
	hour, ok := twoDigits(input, &position)
	if !ok || hour > 23 || position != 3 {
		return TimeZoneOffset{}, false
	}
	position = 4
	minute, ok := twoDigits(input, &position)
	if !ok || minute > 59 {
		return TimeZoneOffset{}, false
	}
	offset := hour*60 + minute
	if input[0] == '-' {
		offset = -offset
	}
	return TimeZoneOffset{Minutes: offset}, true
}

// ParseGlobalDateTime parses a global date and time string, which is a date, a
// separator, a time, and a time-zone offset.
func ParseGlobalDateTime(input string) (GlobalDateTime, bool) {
	separator := -1
	for index := 0; index < len(input); index++ {
		if input[index] == 'T' || input[index] == ' ' {
			separator = index
			break
		}
	}
	if separator < 0 {
		return GlobalDateTime{}, false
	}
	date, ok := ParseDate(input[:separator])
	if !ok {
		return GlobalDateTime{}, false
	}
	remainder := input[separator+1:]
	var offset TimeZoneOffset
	var timeText string
	switch {
	case strings.HasSuffix(remainder, "Z") || strings.HasSuffix(remainder, "z"):
		offset = TimeZoneOffset{Minutes: 0}
		timeText = remainder[:len(remainder)-1]
	case len(remainder) >= 6 && (remainder[len(remainder)-6] == '+' || remainder[len(remainder)-6] == '-'):
		parsed, ok := ParseTimeZoneOffset(remainder[len(remainder)-6:])
		if !ok {
			return GlobalDateTime{}, false
		}
		offset = parsed
		timeText = remainder[:len(remainder)-6]
	default:
		return GlobalDateTime{}, false
	}
	clock, ok := ParseTime(timeText)
	if !ok {
		return GlobalDateTime{}, false
	}
	return GlobalDateTime{LocalDateTime: LocalDateTime{Date: date, Time: clock}, Offset: offset}, true
}

// ParseWeek parses a valid week string (YYYY-Www).
func ParseWeek(input string) (Week, bool) {
	position := 0
	year, ok := parseYear(input, &position)
	if !ok || position+2 > len(input) || input[position] != '-' || input[position+1] != 'W' {
		return Week{}, false
	}
	position += 2
	week, ok := twoDigits(input, &position)
	if !ok || position != len(input) || week < 1 || week > 53 {
		return Week{}, false
	}
	return Week{Year: year, Week: week}, true
}

// Duration is an ISO-8601 duration.
type Duration struct {
	Seconds  float64
	Months   int
	DayCount int
}

// ParseDuration parses a valid duration string.
func ParseDuration(input string) (Duration, bool) {
	if len(input) == 0 || (input[0] != 'P' && input[0] != 'p') {
		return Duration{}, false
	}
	position := 1
	var result Duration
	seenComponent := false
	inTime := false
	for position < len(input) {
		if input[position] == 'T' || input[position] == 't' {
			inTime = true
			position++
			continue
		}
		start := position
		for position < len(input) && (isASCIIDigit(input[position]) || input[position] == '.') {
			position++
		}
		if position == start || position >= len(input) {
			return Duration{}, false
		}
		number, err := strconv.ParseFloat(input[start:position], 64)
		if err != nil {
			return Duration{}, false
		}
		unit := input[position]
		position++
		switch unit {
		case 'Y', 'y':
			if inTime {
				return Duration{}, false
			}
			result.Months += int(number) * 12
		case 'M', 'm':
			if inTime {
				result.Seconds += number * 60
			} else {
				result.Months += int(number)
			}
		case 'W', 'w':
			if inTime {
				return Duration{}, false
			}
			result.DayCount += int(number) * 7
		case 'D', 'd':
			if inTime {
				return Duration{}, false
			}
			result.DayCount += int(number)
		case 'H', 'h':
			result.Seconds += number * 3600
		case 'S', 's':
			result.Seconds += number
		default:
			return Duration{}, false
		}
		seenComponent = true
	}
	if !seenComponent {
		return Duration{}, false
	}
	return result, true
}

func daysInMonth(year, month int) int {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return 31
	case 4, 6, 9, 11:
		return 30
	case 2:
		if isLeapYear(year) {
			return 29
		}
		return 28
	default:
		return 0
	}
}

func isLeapYear(year int) bool {
	return year%4 == 0 && (year%100 != 0 || year%400 == 0)
}

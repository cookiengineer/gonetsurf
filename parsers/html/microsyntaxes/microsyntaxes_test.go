package microsyntaxes

import "testing"

func TestParseInteger(t *testing.T) {
	tests := []struct {
		input   string
		value   int
		success bool
	}{
		{"0", 0, true},
		{"  +42 ", 42, true},
		{"-13abc", -13, true},
		{"abc", 0, false},
		{"+", 0, false},
		{"", 0, false},
	}
	for _, test := range tests {
		value, ok := ParseInteger(test.input)
		if ok != test.success || (ok && value != test.value) {
			t.Errorf("ParseInteger(%q) = (%d, %v), want (%d, %v)", test.input, value, ok, test.value, test.success)
		}
	}
}

func TestParseNonNegativeInteger(t *testing.T) {
	if _, ok := ParseNonNegativeInteger("-1"); ok {
		t.Error("expected negative input to fail")
	}
	if value, ok := ParseNonNegativeInteger("17"); !ok || value != 17 {
		t.Errorf("ParseNonNegativeInteger(17) = (%d, %v)", value, ok)
	}
}

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input   string
		value   float64
		success bool
	}{
		{"1.5", 1.5, true},
		{"-0.25", -0.25, true},
		{"1e3", 1000, true},
		{".5", 0.5, true},
		{"3.", 3, true},
		{"1e", 1, true},
		{"abc", 0, false},
		{"Infinity", 0, false},
	}
	for _, test := range tests {
		value, ok := ParseFloat(test.input)
		if ok != test.success || (ok && value != test.value) {
			t.Errorf("ParseFloat(%q) = (%v, %v), want (%v, %v)", test.input, value, ok, test.value, test.success)
		}
	}
}

func TestParseDimension(t *testing.T) {
	value, percentage, ok := ParseDimension("50%")
	if !ok || value != 50 || !percentage {
		t.Errorf("ParseDimension(50%%) = (%v, %v, %v)", value, percentage, ok)
	}
	value, percentage, ok = ParseDimension("12.5px")
	if !ok || value != 12.5 || percentage {
		t.Errorf("ParseDimension(12.5px) = (%v, %v, %v)", value, percentage, ok)
	}
	if _, _, ok := ParseDimension("-1"); ok {
		t.Error("expected negative dimension to fail")
	}
}

func TestParseSimpleColor(t *testing.T) {
	color, ok := ParseSimpleColor("#ff8000")
	if !ok || color.Red() != 0xFF || color.Green() != 0x80 || color.Blue() != 0 {
		t.Fatalf("ParseSimpleColor = (%v, %v)", color, ok)
	}
	if _, ok := ParseSimpleColor("#fff"); ok {
		t.Error("expected short hex to fail for simple color")
	}
	if got := SerializeSimpleColor(color); got != "#ff8000" {
		t.Errorf("SerializeSimpleColor = %q", got)
	}
}

func TestParseLegacyColor(t *testing.T) {
	tests := []struct {
		input string
		want  uint32
		ok    bool
	}{
		{"red", 0xFF0000, true},
		{"#f00", 0xFF0000, true},
		{"#ff0000", 0xFF0000, true},
		{"ff0000", 0xFF0000, true},
		{"transparent", 0, false},
	}
	for _, test := range tests {
		color, ok := ParseLegacyColor(test.input)
		if ok != test.ok || (ok && uint32(color) != test.want) {
			t.Errorf("ParseLegacyColor(%q) = (%#x, %v), want (%#x, %v)", test.input, uint32(color), ok, test.want, test.ok)
		}
	}
}

func TestSplitTokens(t *testing.T) {
	space := SplitSpaceSeparated("  a\tb\nc ")
	if len(space) != 3 || space[0] != "a" || space[2] != "c" {
		t.Errorf("SplitSpaceSeparated = %v", space)
	}
	comma := SplitCommaSeparated(" a ,b,,d d ")
	if len(comma) != 4 || comma[0] != "a" || comma[2] != "" || comma[3] != "d d" {
		t.Errorf("SplitCommaSeparated = %q", comma)
	}
	unique := UniqueSpaceSeparated("a b a c b")
	if len(unique) != 3 {
		t.Errorf("UniqueSpaceSeparated = %v", unique)
	}
}

func TestHashNameReference(t *testing.T) {
	if name, ok := HashNameReference("#section-1"); !ok || name != "section-1" {
		t.Errorf("HashNameReference = (%q, %v)", name, ok)
	}
	if _, ok := HashNameReference("#"); ok {
		t.Error("expected empty name to fail")
	}
	if _, ok := HashNameReference("section"); ok {
		t.Error("expected missing hash to fail")
	}
}

func TestParseDate(t *testing.T) {
	date, ok := ParseDate("2024-02-29")
	if !ok || date.Year != 2024 || date.Month != 2 || date.Day != 29 {
		t.Fatalf("ParseDate leap = (%+v, %v)", date, ok)
	}
	if _, ok := ParseDate("2023-02-29"); ok {
		t.Error("expected invalid leap date to fail")
	}
	if _, ok := ParseDate("2024-13-01"); ok {
		t.Error("expected invalid month to fail")
	}
}

func TestParseTime(t *testing.T) {
	clock, ok := ParseTime("23:59:58.5")
	if !ok || clock.Hour != 23 || clock.Minute != 59 || clock.Second != 58 || clock.Millisecond != 500 {
		t.Fatalf("ParseTime = (%+v, %v)", clock, ok)
	}
	if _, ok := ParseTime("24:00"); ok {
		t.Error("expected invalid hour to fail")
	}
}

func TestParseGlobalDateTime(t *testing.T) {
	value, ok := ParseGlobalDateTime("2024-06-01T12:30Z")
	if !ok || value.Offset.Minutes != 0 || value.LocalDateTime.Time.Hour != 12 {
		t.Fatalf("ParseGlobalDateTime Z = (%+v, %v)", value, ok)
	}
	value, ok = ParseGlobalDateTime("2024-06-01 12:30:15+02:00")
	if !ok || value.Offset.Minutes != 120 {
		t.Fatalf("ParseGlobalDateTime offset = (%+v, %v)", value, ok)
	}
}

func TestParseWeek(t *testing.T) {
	week, ok := ParseWeek("2024-W01")
	if !ok || week.Year != 2024 || week.Week != 1 {
		t.Fatalf("ParseWeek = (%+v, %v)", week, ok)
	}
	if _, ok := ParseWeek("2024-W54"); ok {
		t.Error("expected week 54 to fail")
	}
}

func TestParseDuration(t *testing.T) {
	duration, ok := ParseDuration("P1Y2M3DT4H5M6S")
	if !ok || duration.Months != 14 || duration.DayCount != 3 || duration.Seconds != 4*3600+5*60+6 {
		t.Fatalf("ParseDuration = (%+v, %v)", duration, ok)
	}
}

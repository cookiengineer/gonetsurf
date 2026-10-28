package microsyntaxes

import "testing"

func FuzzParsers(f *testing.F) {
	for _, seed := range []string{
		"",
		"42",
		"  -3.5e2 ",
		"2011-11-12T13:14:15Z",
		"#a1b2c3",
		"red",
		"a  b\tc",
		"a,b,,c",
	} {
		f.Add(seed)
	}
	f.Fuzz(func(t *testing.T, input string) {
		_, _ = ParseInteger(input)
		_, _ = ParseNonNegativeInteger(input)
		_, _ = ParseFloat(input)
		_, _ = ParsePercentage(input)
		_, _, _ = ParseDimension(input)
		_, _ = ParseMonth(input)
		_, _ = ParseDate(input)
		_, _ = ParseYearlessDate(input)
		_, _ = ParseTime(input)
		_, _ = ParseLocalDateTime(input)
		_, _ = ParseTimeZoneOffset(input)
		_, _ = ParseGlobalDateTime(input)
		_, _ = ParseWeek(input)
		_, _ = ParseDuration(input)
		_, _ = ParseSimpleColor(input)
		_, _ = ParseLegacyColor(input)
		_, _ = NamedColor(input)
		_, _ = HashNameReference(input)
		_ = SplitSpaceSeparated(input)
		_ = SplitCommaSeparated(input)
		_ = UniqueSpaceSeparated(input)
		_ = UniqueCommaSeparated(input)
	})
}

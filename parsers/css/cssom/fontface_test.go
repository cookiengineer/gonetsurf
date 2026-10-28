package cssom

import (
	"strings"
	"testing"
)

const fontFaceSource = `@font-face {
  font-family: "Test Face";
  src: local("Go"), url("https://example.com/font.woff2") format("woff2"), url(data:font/ttf;base64,AAAA) format("truetype");
  font-weight: 100 900;
  font-style: italic;
  font-stretch: 75% 100%;
  unicode-range: U+0000-00FF, U+4??;
  font-display: swap;
  font-feature-settings: "kern" 1;
  size-adjust: 90%;
  ascent-override: 88%;
  descent-override: 12%;
  line-gap-override: 0%;
}`

func TestParseFontFaceRule(t *testing.T) {
	stylesheet := Parse(fontFaceSource)
	if len(stylesheet.Rules) != 1 {
		t.Fatalf("rules = %d", len(stylesheet.Rules))
	}
	rule, ok := stylesheet.Rules[0].(*FontFaceRule)
	if !ok {
		t.Fatalf("rule = %T, want *FontFaceRule", stylesheet.Rules[0])
	}
	if rule.Family != "Test Face" {
		t.Errorf("family = %q", rule.Family)
	}
	if len(rule.Sources) != 3 {
		t.Fatalf("sources = %#v", rule.Sources)
	}
	if rule.Sources[0].Local != "Go" || rule.Sources[1].URL != "https://example.com/font.woff2" || rule.Sources[1].Format != "woff2" {
		t.Errorf("sources = %#v", rule.Sources)
	}
	if !strings.HasPrefix(rule.Sources[2].URL, "data:font/ttf") {
		t.Errorf("data source = %#v", rule.Sources[2])
	}
	if rule.FontWeight != "100 900" || rule.FontStyle != "italic" || rule.FontDisplay != "swap" {
		t.Errorf("descriptors = weight %q style %q display %q", rule.FontWeight, rule.FontStyle, rule.FontDisplay)
	}
	if len(rule.UnicodeRange) != 2 {
		t.Fatalf("unicode ranges = %#v", rule.UnicodeRange)
	}
	if rule.UnicodeRange[0].Start != 0 || rule.UnicodeRange[0].End != 0xFF {
		t.Errorf("first range = %#v", rule.UnicodeRange[0])
	}
	if rule.UnicodeRange[1].Start != 0x400 || rule.UnicodeRange[1].End != 0x4FF {
		t.Errorf("second range = %#v", rule.UnicodeRange[1])
	}
	if rule.SizeAdjust != "90%" || rule.AscentOverride != "88%" {
		t.Errorf("overrides = %q %q", rule.SizeAdjust, rule.AscentOverride)
	}
}

func TestFontFaceSerializeRoundTrip(t *testing.T) {
	stylesheet := Parse(fontFaceSource)
	rule := stylesheet.Rules[0].(*FontFaceRule)
	serialized := rule.Serialize()
	if !strings.HasPrefix(serialized, "@font-face {") {
		t.Fatalf("serialized = %q", serialized)
	}
	reparsed := Parse(serialized)
	if len(reparsed.Rules) != 1 {
		t.Fatalf("reparsed rules = %d", len(reparsed.Rules))
	}
	again, ok := reparsed.Rules[0].(*FontFaceRule)
	if !ok || again.Family != rule.Family || len(again.Sources) != len(rule.Sources) || len(again.UnicodeRange) != len(rule.UnicodeRange) {
		t.Errorf("round trip = %#v", again)
	}
}

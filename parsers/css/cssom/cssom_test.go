package cssom

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"testing"
)

func TestParseStylesheet(t *testing.T) {
	sheet := Parse("a { color: red; margin: 0 !important } b {}")
	if len(sheet.Rules) != 2 {
		t.Fatalf("got %d rules, want 2", len(sheet.Rules))
	}
	style, ok := sheet.Rules[0].(*StyleRule)
	if !ok {
		t.Fatalf("rule 0 is %T, want *StyleRule", sheet.Rules[0])
	}
	if style.Selectors == nil || len(style.Selectors.Selectors) != 1 {
		t.Fatalf("selectors = %#v", style.Selectors)
	}
	if len(style.Declarations) != 2 {
		t.Fatalf("got %d declarations, want 2", len(style.Declarations))
	}
	if style.Declarations[0].Name != "color" || style.Declarations[0].Important {
		t.Errorf("declaration 0 = %#v", style.Declarations[0])
	}
	if style.Declarations[1].Name != "margin" || !style.Declarations[1].Important {
		t.Errorf("declaration 1 = %#v", style.Declarations[1])
	}
}

func TestSerializeStyleRule(t *testing.T) {
	sheet := Parse("a, b { color: red; margin: 0 !important }")
	got := sheet.Serialize()
	want := "a, b { color: red; margin: 0 !important; }"
	if got != want {
		t.Errorf("Serialize() = %q, want %q", got, want)
	}
}

func TestParseAtRule(t *testing.T) {
	sheet := Parse("@media screen { a { color: red } }")
	if len(sheet.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(sheet.Rules))
	}
	atRule, ok := sheet.Rules[0].(*AtRule)
	if !ok {
		t.Fatalf("rule = %T, want *AtRule", sheet.Rules[0])
	}
	if atRule.Name != "media" {
		t.Errorf("name = %q, want media", atRule.Name)
	}
}

func TestParseStatementAtRule(t *testing.T) {
	sheet := Parse("@import url(foo.css);")
	if len(sheet.Rules) != 1 {
		t.Fatalf("got %d rules, want 1", len(sheet.Rules))
	}
	atRule := sheet.Rules[0].(*AtRule)
	if atRule.Block != nil {
		t.Errorf("block = %#v, want nil", atRule.Block)
	}
	if got := atRule.Serialize(); got != "@import url(foo.css);" {
		t.Errorf("Serialize() = %q", got)
	}
}

func TestSerializeRoundTrip(t *testing.T) {
	inputs := []string{
		"a { color: red; }",
		"a, b:hover > c::after { color: #abcdef; width: calc(100% - 2px); }",
		"@media screen { a { color: red } }",
		"@import url(foo.css);",
		"div { margin: 0 auto !important; --brand: green; }",
	}
	for _, input := range inputs {
		sheet := Parse(input)
		first := sheet.Serialize()
		reparsed := Parse(first)
		second := reparsed.Serialize()
		if first != second {
			t.Errorf("serialize not stable for %q:\n first: %q\nsecond: %q", input, first, second)
		}
		if len(reparsed.Rules) != len(sheet.Rules) {
			t.Errorf("rule count changed for %q: %d -> %d", input, len(sheet.Rules), len(reparsed.Rules))
		}
	}
}

func TestParseRule(t *testing.T) {
	rule, err := ParseRule("div { color: blue }")
	if err != nil {
		t.Fatalf("ParseRule: %v", err)
	}
	style, ok := rule.(*StyleRule)
	if !ok {
		t.Fatalf("rule = %T", rule)
	}
	if len(style.Declarations) != 1 {
		t.Errorf("declarations = %#v", style.Declarations)
	}
}

type mediaListCase struct {
	Input     string   `json:"input"`
	MediaText string   `json:"mediaText"`
	Items     []string `json:"items"`
}

func TestMediaListConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/media-list.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []mediaListCase
	if err := json.Unmarshal(encoded, &cases); err != nil {
		t.Fatal(err)
	}
	for _, testCase := range cases {
		list, err := ParseMediaList(testCase.Input)
		if err != nil {
			t.Errorf("ParseMediaList(%q): %v", testCase.Input, err)
			continue
		}
		if list.MediaText() != testCase.MediaText {
			t.Errorf("ParseMediaList(%q).MediaText() = %q, want %q", testCase.Input, list.MediaText(), testCase.MediaText)
		}
		if list.Length() != len(testCase.Items) {
			t.Errorf("ParseMediaList(%q).Length() = %d, want %d", testCase.Input, list.Length(), len(testCase.Items))
			continue
		}
		for index, want := range testCase.Items {
			if got := list.Item(index); got != want {
				t.Errorf("ParseMediaList(%q).Item(%d) = %q, want %q", testCase.Input, index, got, want)
			}
		}
	}
}

func TestMediaListMutation(t *testing.T) {
	list, _ := ParseMediaList("screen")
	list.AppendMedium("print")
	if got := list.MediaText(); got != "screen, print" {
		t.Errorf("append print = %q", got)
	}
	list.AppendMedium("SCREEN")
	if got := list.MediaText(); got != "screen, print" {
		t.Errorf("append duplicate = %q", got)
	}
	list.AppendMedium("bad$")
	if got := list.MediaText(); got != "screen, print, not all" {
		t.Errorf("append invalid = %q", got)
	}
	if err := list.DeleteMedium("SCREEN"); err != nil {
		t.Errorf("delete SCREEN: %v", err)
	}
	if err := list.DeleteMedium("print"); err != nil {
		t.Errorf("delete print: %v", err)
	}
	if err := list.DeleteMedium("print"); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete missing = %v, want ErrNotFound", err)
	}
}

func TestStyleDeclarationMutation(t *testing.T) {
	sheet := Parse("a { color: red; margin: 0 }")
	rule := sheet.CSSRules()[0].(*StyleRule)
	style := rule.Style()
	if style.Length() != 2 {
		t.Fatalf("length = %d, want 2", style.Length())
	}
	if got := style.GetPropertyValue("color"); got != "red" {
		t.Errorf("color = %q, want red", got)
	}
	if err := style.SetProperty("color", "blue", "important"); err != nil {
		t.Fatal(err)
	}
	if got := style.GetPropertyValue("color"); got != "blue" {
		t.Errorf("color after set = %q, want blue", got)
	}
	if got := style.GetPropertyPriority("color"); got != "important" {
		t.Errorf("priority = %q, want important", got)
	}
	if got := style.RemoveProperty("margin"); got != "0" {
		t.Errorf("removed margin = %q, want 0", got)
	}
	if style.Length() != 1 {
		t.Errorf("length after remove = %d, want 1", style.Length())
	}
	if err := style.SetProperty("color", "red", "urgent"); err == nil {
		t.Error("expected invalid priority error")
	}
	if !strings.Contains(style.CSSText(), "color: blue !important;") {
		t.Errorf("cssText = %q", style.CSSText())
	}
}

func TestRuleMutation(t *testing.T) {
	sheet := Parse("a { color: red }")
	if err := sheet.InsertRule("b { color: green }", 1); err != nil {
		t.Fatal(err)
	}
	if len(sheet.CSSRules()) != 2 {
		t.Fatalf("rules = %d, want 2", len(sheet.CSSRules()))
	}
	if sheet.CSSRules()[1].Serialize() != "b { color: green; }" {
		t.Errorf("inserted rule = %q", sheet.CSSRules()[1].Serialize())
	}
	if err := sheet.DeleteRule(0); err != nil {
		t.Fatal(err)
	}
	if len(sheet.CSSRules()) != 1 {
		t.Errorf("rules after delete = %d, want 1", len(sheet.CSSRules()))
	}
	if err := sheet.InsertRule("a {}", 5); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("insert out of range = %v", err)
	}
	if err := sheet.DeleteRule(5); !errors.Is(err, ErrIndexOutOfRange) {
		t.Errorf("delete out of range = %v", err)
	}
}

func TestNestedCSSOM(t *testing.T) {
	sheet := Parse("@media screen { a { color: red } } a { & .b { color: blue } }")
	if len(sheet.CSSRules()) != 2 {
		t.Fatalf("rules = %d, want 2", len(sheet.CSSRules()))
	}
	media := sheet.CSSRules()[0].(*AtRule)
	if media.Media().MediaText() != "screen" {
		t.Errorf("media = %q", media.Media().MediaText())
	}
	if len(media.CSSRules()) != 1 {
		t.Fatalf("media rules = %d, want 1", len(media.CSSRules()))
	}
	style := sheet.CSSRules()[1].(*StyleRule)
	if len(style.CSSRules()) != 1 {
		t.Fatalf("nested rules = %d, want 1", len(style.CSSRules()))
	}
	if err := style.SetSelectorText("div > span"); err != nil {
		t.Fatal(err)
	}
	if got := style.SelectorText(); got != "div > span" {
		t.Errorf("selector = %q", got)
	}
	if err := style.SetSelectorText("div >"); err == nil {
		t.Error("expected invalid selector error")
	}
}

func TestDeclarationBlockAndAtRuleStyle(t *testing.T) {
	style := ParseDeclarations("color: red; margin: 0 !important")
	if style.Length() != 2 || style.GetPropertyPriority("margin") != "important" {
		t.Errorf("declarations = %q", style.CSSText())
	}
	sheet := Parse("@font-face { font-family: X }")
	fontRule := sheet.CSSRules()[0].(*FontFaceRule)
	if got := fontRule.Style().GetPropertyValue("font-family"); got != "X" {
		t.Errorf("font-family = %q, want X", got)
	}
}

func TestLiveMediaList(t *testing.T) {
	sheet := Parse("@media screen { a { color: red } }")
	atRule := sheet.CSSRules()[0].(*AtRule)
	atRule.Media().SetMediaText("print,  screen ")
	if got := atRule.Media().MediaText(); got != "print, screen" {
		t.Errorf("media = %q, want print, screen", got)
	}
	if !strings.Contains(atRule.Serialize(), "@media print, screen") {
		t.Errorf("serialized at-rule = %q", atRule.Serialize())
	}
}

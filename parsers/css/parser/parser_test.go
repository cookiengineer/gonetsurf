package parser

import (
	"errors"
	"strconv"
	"strings"
	"testing"

	"gonetsurf/parsers/css/tokenizer"
)

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'g', -1, 64)
}

// dump renders a component value as a compact, deterministic string so tests
// stay readable.
func dump(value ComponentValue) string {
	switch typed := value.(type) {
	case Token:
		return dumpToken(typed.Token)
	case *Function:
		parts := make([]string, len(typed.Value))
		for index, argument := range typed.Value {
			parts[index] = dump(argument)
		}
		return "function(" + typed.Name + " [" + strings.Join(parts, " ") + "])"
	case *Block:
		parts := make([]string, len(typed.Value))
		for index, item := range typed.Value {
			parts[index] = dump(item)
		}
		return "block(" + openName(typed.Associated) + " [" + strings.Join(parts, " ") + "])"
	default:
		return "?"
	}
}

func numericKindName(kind tokenizer.NumericKind) string {
	if kind == tokenizer.Number {
		return "number"
	}
	return "integer"
}

func openName(tokenType tokenizer.TokenType) string {
	switch tokenType {
	case tokenizer.TokenOpenCurly:
		return "{"
	case tokenizer.TokenOpenSquare:
		return "["
	case tokenizer.TokenOpenParen:
		return "("
	default:
		return "?"
	}
}

func dumpToken(token tokenizer.Token) string {
	switch token.Type {
	case tokenizer.TokenIdent:
		return "ident(" + token.Value + ")"
	case tokenizer.TokenAtKeyword:
		return "at(" + token.Value + ")"
	case tokenizer.TokenHash:
		flag := "unrestricted"
		if token.HashFlag == tokenizer.HashID {
			flag = "id"
		}
		return "hash(" + token.Value + "," + flag + ")"
	case tokenizer.TokenString:
		return "string(" + token.Value + ")"
	case tokenizer.TokenBadString:
		return "bad-string"
	case tokenizer.TokenURL:
		return "url(" + token.Value + ")"
	case tokenizer.TokenBadURL:
		return "bad-url"
	case tokenizer.TokenDelim:
		return "delim(" + token.Value + ")"
	case tokenizer.TokenNumber:
		return "number(" + formatNumber(token.Number) + "," + numericKindName(token.NumericKind) + ")"
	case tokenizer.TokenPercentage:
		return "percentage(" + formatNumber(token.Number) + "," + numericKindName(token.NumericKind) + ")"
	case tokenizer.TokenDimension:
		return "dimension(" + formatNumber(token.Number) + "," + numericKindName(token.NumericKind) + "," + token.Value + ")"
	case tokenizer.TokenWhitespace:
		return "ws"
	case tokenizer.TokenCDO:
		return "cdo"
	case tokenizer.TokenCDC:
		return "cdc"
	case tokenizer.TokenColon:
		return "colon"
	case tokenizer.TokenSemicolon:
		return "semicolon"
	case tokenizer.TokenComma:
		return "comma"
	case tokenizer.TokenCloseParen:
		return "closeparen"
	case tokenizer.TokenCloseSquare:
		return "closesquare"
	case tokenizer.TokenCloseCurly:
		return "closecurly"
	case tokenizer.TokenEOF:
		return "eof"
	default:
		return "token"
	}
}

func dumpValues(values []ComponentValue) string {
	parts := make([]string, len(values))
	for index, value := range values {
		parts[index] = dump(value)
	}
	return strings.Join(parts, " ")
}

func TestParseComponentValueList(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"", ""},
		{"/* comment */", ""},
		{"a b", "ident(a) ws ident(b)"},
		{"1 2.5 3px 4%", "number(1,integer) ws number(2.5,number) ws dimension(3,integer,px) ws percentage(4,integer)"},
		{"rgba(1, 2)", "function(rgba [number(1,integer) comma ws number(2,integer)])"},
		{"{}", "block({ [])"},
		{"[a(b)]", "block([ [function(a [ident(b)])])"},
		{"(foo]{bar)", "block(( [ident(foo) closesquare block({ [ident(bar) closeparen])])"},
		{"#id #12", "hash(id,id) ws hash(12,unrestricted)"},
		{"'str' \"str2\"", "string(str) ws string(str2)"},
		{"url(foo) url('bar')", "url(foo) ws function(url [string(bar)])"},
		{"<!-- -->", "cdo ws cdc"},
	}
	for _, test := range tests {
		if got := dumpValues(ParseComponentValueList(test.input)); got != test.want {
			t.Errorf("ParseComponentValueList(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestParseComponentValue(t *testing.T) {
	value, err := ParseComponentValue("4px")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := dump(value); got != "dimension(4,integer,px)" {
		t.Errorf("got %q", got)
	}
	if _, err := ParseComponentValue("  "); !errors.Is(err, ErrEmpty) {
		t.Errorf("empty input error = %v, want ErrEmpty", err)
	}
	if _, err := ParseComponentValue("a b"); !errors.Is(err, ErrExtraInput) {
		t.Errorf("extra input error = %v, want ErrExtraInput", err)
	}
}

func TestParseCommaSeparatedComponentValueList(t *testing.T) {
	lists := ParseCommaSeparatedComponentValueList("a, b, c")
	if len(lists) != 3 {
		t.Fatalf("got %d lists, want 3", len(lists))
	}
	if dumpValues(lists[0]) != "ident(a)" || dumpValues(lists[1]) != "ws ident(b)" {
		t.Errorf("unexpected lists: %q, %q", dumpValues(lists[0]), dumpValues(lists[1]))
	}
}

func TestParseDeclaration(t *testing.T) {
	tests := []struct {
		input     string
		name      string
		value     string
		important bool
	}{
		{"color: red", "color", "ident(red)", false},
		{"color:red", "color", "ident(red)", false},
		{"color: red !important", "color", "ident(red)", true},
		{"color: red !IMPORTANT", "color", "ident(red)", true},
		{"color: red ! /*c*/ important", "color", "ident(red)", true},
		{"color: red !important!", "color", "ident(red) ws delim(!) ident(important) delim(!)", false},
		{"color: red important", "color", "ident(red) ws ident(important)", false},
		{"--custom: a b", "--custom", "ident(a) ws ident(b)", false},
	}
	for _, test := range tests {
		declaration, err := ParseDeclaration(test.input)
		if err != nil {
			t.Errorf("ParseDeclaration(%q) error: %v", test.input, err)
			continue
		}
		if declaration.Name != test.name {
			t.Errorf("ParseDeclaration(%q).Name = %q, want %q", test.input, declaration.Name, test.name)
		}
		if got := dumpValues(declaration.Value); got != test.value {
			t.Errorf("ParseDeclaration(%q).Value = %q, want %q", test.input, got, test.value)
		}
		if declaration.Important != test.important {
			t.Errorf("ParseDeclaration(%q).Important = %v, want %v", test.input, declaration.Important, test.important)
		}
	}
}

func TestParseDeclarationErrors(t *testing.T) {
	tests := []struct {
		input string
		want  error
	}{
		{"", ErrEmpty},
		{"  /**/ ", ErrEmpty},
		{"foo", ErrInvalid},
		{"@foo: bar", ErrInvalid},
		{"#foo: bar", ErrInvalid},
		{"foo*: bar", ErrInvalid},
	}
	for _, test := range tests {
		_, err := ParseDeclaration(test.input)
		if !errors.Is(err, test.want) {
			t.Errorf("ParseDeclaration(%q) error = %v, want %v", test.input, err, test.want)
		}
	}
}

func TestParseListOfDeclarations(t *testing.T) {
	nodes := ParseListOfDeclarations("a:b; c:d 42!important; @import 'x.css';")
	if len(nodes) != 3 {
		t.Fatalf("got %d nodes, want 3: %v", len(nodes), nodes)
	}
	first, ok := nodes[0].(*Declaration)
	if !ok || first.Name != "a" || dumpValues(first.Value) != "ident(b)" || first.Important {
		t.Errorf("first node = %#v", nodes[0])
	}
	second, ok := nodes[1].(*Declaration)
	if !ok || second.Name != "c" || !second.Important {
		t.Errorf("second node = %#v", nodes[1])
	}
	atRule, ok := nodes[2].(*AtRule)
	if !ok || atRule.Name != "import" || dumpValues(atRule.Prelude) != "ws string(x.css)" || atRule.Block != nil {
		t.Errorf("third node = %#v", nodes[2])
	}
}

func TestParseListOfDeclarationsRecovery(t *testing.T) {
	nodes := ParseListOfDeclarations("z;a:b")
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1: %v", len(nodes), nodes)
	}
	declaration, ok := nodes[0].(*Declaration)
	if !ok || declaration.Name != "a" {
		t.Errorf("node = %#v", nodes[0])
	}
}

func TestParseRule(t *testing.T) {
	rule, err := ParseRule("div > p { color: red }")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	qualified, ok := rule.(*QualifiedRule)
	if !ok {
		t.Fatalf("rule = %T, want *QualifiedRule", rule)
	}
	if dumpValues(qualified.Prelude) != "ident(div) ws delim(>) ws ident(p) ws" {
		t.Errorf("prelude = %q", dumpValues(qualified.Prelude))
	}
	if qualified.Block == nil || dumpValues(qualified.Block.Value) != "ws ident(color) colon ws ident(red) ws" {
		t.Errorf("block = %#v", qualified.Block)
	}

	atRuleValue, err := ParseRule("@media screen { div {} }")
	if err != nil {
		t.Fatalf("error: %v", err)
	}
	atRule, ok := atRuleValue.(*AtRule)
	if !ok || atRule.Name != "media" || atRule.Block == nil {
		t.Errorf("at-rule = %#v", atRuleValue)
	}
}

func TestParseRuleErrors(t *testing.T) {
	tests := []struct {
		input string
		want  error
	}{
		{"", ErrEmpty},
		{"foo", ErrInvalid},
		{"div {} extra", ErrExtraInput},
	}
	for _, test := range tests {
		_, err := ParseRule(test.input)
		if !errors.Is(err, test.want) {
			t.Errorf("ParseRule(%q) error = %v, want %v", test.input, err, test.want)
		}
	}
}

func TestStylesheetIgnoresCDOAndCDC(t *testing.T) {
	nodes := ParseStylesheet("<!-- a{} --> b{}")
	if len(nodes) != 2 {
		t.Fatalf("got %d nodes, want 2: %v", len(nodes), nodes)
	}
	first := nodes[0].(*QualifiedRule)
	if dumpValues(first.Prelude) != "ident(a)" {
		t.Errorf("first prelude = %q", dumpValues(first.Prelude))
	}
}

func TestListOfRulesTreatsCDOAsRule(t *testing.T) {
	nodes := ParseListOfRules("<!-- a{}")
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1: %v", len(nodes), nodes)
	}
	rule := nodes[0].(*QualifiedRule)
	if dumpValues(rule.Prelude) != "cdo ws ident(a)" {
		t.Errorf("prelude = %q", dumpValues(rule.Prelude))
	}
}

func TestParseStylesheetInvalidRulesDropped(t *testing.T) {
	nodes := ParseStylesheet("foo 4; div {}")
	if len(nodes) != 1 {
		t.Fatalf("got %d nodes, want 1: %v", len(nodes), nodes)
	}
	if _, ok := nodes[0].(*QualifiedRule); !ok {
		t.Errorf("node = %T, want *QualifiedRule", nodes[0])
	}
}

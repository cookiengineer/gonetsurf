package styles

import (
	"testing"

	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/parser"
)

func FuzzExpandDeclaration(f *testing.F) {
	f.Add("background", "url(a) no-repeat center / cover #fff")
	f.Add("border-radius", "10px 20px / 5px")
	f.Add("border", "1px solid red")
	f.Add("flex", "1 1 auto")
	f.Fuzz(func(t *testing.T, name, value string) {
		switch name {
		case "background", "border-radius", "border", "border-top", "margin", "padding",
			"overflow", "flex", "flex-flow", "gap", "inset", "margin-inline":
		default:
			return
		}
		declaration := &cssom.Declaration{Name: name, Value: parser.ParseComponentValueList(value)}
		for _, expanded := range expandDeclaration(declaration) {
			if expanded == nil {
				t.Fatal("nil expanded declaration")
			}
			_ = parser.SerializeValues(expanded.Value)
		}
	})
}

package html

import (
	"testing"

	"gonetsurf/dom"
)

func FuzzParse(f *testing.F) {
	for _, seed := range []string{
		"",
		"<!doctype html><p>hello",
		"<table><tr><td>x",
		"<svg><foreignObject><div></div></foreignObject></svg>",
		"<meta charset=windows-1251><p>\xe9",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		document, err := Parse(data, ParseOptions{NodeBudget: 50_000})
		if err != nil || document == nil {
			return
		}
		_ = Serialize(&document.Node)
	})
}

func FuzzParseFragment(f *testing.F) {
	for _, seed := range []string{
		"",
		"<div>text</div>",
		"<td>cell",
		"<option>value",
	} {
		f.Add([]byte(seed))
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		document := dom.NewDocument()
		context := document.CreateElement("div")
		nodes, err := ParseFragment(context, data)
		if err != nil {
			return
		}
		for _, node := range nodes {
			_ = Serialize(node)
		}
	})
}

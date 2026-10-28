package adapters

import (
	"strings"
	"testing"

	"gonetsurf/parsers/html"
	"gonetsurf/urls"
)

func TestStylesheetLoaderLoadsLinksAndImports(t *testing.T) {
	document, err := html.ParseString(`<!doctype html><html><head>
		<link rel="stylesheet" href="/site.css">
		<style>@import "/extra.css"; p { color: blue }</style>
	</head><body><p id="text">hello</p><div id="box"></div></body></html>`,
		html.ParseOptions{URL: mustURL(t, "https://example.com/index.html")})
	if err != nil {
		t.Fatal(err)
	}
	sheets := map[string]string{
		"https://example.com/site.css":   `@import "nested.css"; #box { background-color: #ff0000 }`,
		"https://example.com/nested.css": `p { color: green }`,
		"https://example.com/extra.css":  `p { font-weight: bold }`,
	}
	loader := StylesheetLoader{Fetch: func(target *urls.URL) ([]byte, error) {
		return []byte(sheets[target.String()]), nil
	}}
	engine := loader.Engine(document, document.URL)
	computed := engine.ComputeDocumentTree(document)
	paragraph := document.GetElementById("text")
	box := document.GetElementById("box")
	if got := computed[paragraph].GetString("color"); got != "rgb(0, 0, 255)" {
		t.Errorf("color = %q", got)
	}
	if got := computed[paragraph].GetString("font-weight"); got != "700" {
		t.Errorf("font-weight = %q", got)
	}
	if got := computed[box].GetString("background-color"); got != "rgb(255, 0, 0)" {
		t.Errorf("background = %q", got)
	}
}

func TestStylesheetLoaderIgnoresImportCycles(t *testing.T) {
	document, err := html.ParseString(`<!doctype html><link rel="stylesheet" href="a.css"><p id="text">x</p>`,
		html.ParseOptions{URL: mustURL(t, "https://example.com/")})
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int{}
	loader := StylesheetLoader{Fetch: func(target *urls.URL) ([]byte, error) {
		counts[target.String()]++
		if strings.HasSuffix(target.String(), "a.css") {
			return []byte(`@import "b.css"; p { color: red }`), nil
		}
		return []byte(`@import "a.css"; p { font-weight: bold }`), nil
	}}
	engine := loader.Engine(document, document.URL)
	computed := engine.ComputeDocumentTree(document)
	if got := computed[document.GetElementById("text")].GetString("color"); got != "rgb(255, 0, 0)" {
		t.Errorf("color = %q", got)
	}
	if counts["https://example.com/a.css"] != 1 || counts["https://example.com/b.css"] != 1 {
		t.Errorf("fetch counts = %v", counts)
	}
}

func mustURL(t *testing.T, value string) *urls.URL {
	t.Helper()
	parsed, err := urls.Parse(value)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestStylesheetLoaderAbsolutizesResourceURLs(t *testing.T) {
	document, err := html.ParseString(`<!doctype html><html><head>
		<link rel="stylesheet" href="/css/page.css">
	</head><body><div id="box"></div></body></html>`,
		html.ParseOptions{URL: mustURL(t, "https://example.com/index.html")})
	if err != nil {
		t.Fatal(err)
	}
	loader := StylesheetLoader{Fetch: func(target *urls.URL) ([]byte, error) {
		return []byte(`
			@font-face { font-family: "site"; src: url("./fonts/site.woff") format("woff") }
			#box { background-image: url(../images/bg.png) }
		`), nil
	}}
	engine := loader.Engine(document, document.URL)
	faces := engine.FontFaces()
	if len(faces) != 1 || len(faces[0].Sources) != 1 {
		t.Fatalf("font faces = %#v", faces)
	}
	if got := faces[0].Sources[0].URL; got != "https://example.com/css/fonts/site.woff" {
		t.Errorf("font URL = %q", got)
	}
	computed := engine.ComputeDocumentTree(document)
	if got := computed[document.GetElementById("box")].GetString("background-image"); got != `url("https://example.com/images/bg.png")` {
		t.Errorf("background-image = %q", got)
	}
}

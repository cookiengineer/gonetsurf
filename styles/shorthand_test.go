package styles

import (
	"strings"
	"testing"

	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/parser"
)

func expandTestDeclaration(t *testing.T, name, value string) map[string]string {
	t.Helper()
	declaration := &cssom.Declaration{Name: name, Value: parser.ParseComponentValueList(value)}
	expanded := expandDeclaration(declaration)
	result := make(map[string]string, len(expanded))
	for _, item := range expanded {
		result[item.Name] = parser.SerializeValues(item.Value)
	}
	return result
}

func TestExpandBorderRadius(t *testing.T) {
	expanded := expandTestDeclaration(t, "border-radius", "10px 20px / 5px")
	want := map[string]string{
		"border-top-left-radius":     "10px 5px",
		"border-top-right-radius":    "20px 5px",
		"border-bottom-right-radius": "10px 5px",
		"border-bottom-left-radius":  "20px 5px",
	}
	for property, expected := range want {
		if expanded[property] != expected {
			t.Errorf("%s = %q, want %q", property, expanded[property], expected)
		}
	}
}

func TestExpandBackgroundColor(t *testing.T) {
	expanded := expandTestDeclaration(t, "background", "#f0d080")
	if expanded["background-color"] != "#f0d080" {
		t.Errorf("background-color = %q", expanded["background-color"])
	}
	if expanded["background-image"] != "none" {
		t.Errorf("background-image = %q", expanded["background-image"])
	}
}

func TestExpandBackgroundLayer(t *testing.T) {
	expanded := expandTestDeclaration(t, "background",
		`url("data:image/png;base64,AAAA") no-repeat center / cover`)
	if !strings.Contains(expanded["background-image"], "url(") {
		t.Errorf("background-image = %q", expanded["background-image"])
	}
	if expanded["background-repeat"] != "no-repeat" {
		t.Errorf("background-repeat = %q", expanded["background-repeat"])
	}
	if expanded["background-position"] != "center" {
		t.Errorf("background-position = %q", expanded["background-position"])
	}
	if expanded["background-size"] != "cover" {
		t.Errorf("background-size = %q", expanded["background-size"])
	}
}

func TestExpandBackgroundMultipleLayers(t *testing.T) {
	expanded := expandTestDeclaration(t, "background",
		`url(a) no-repeat, url(b) repeat red`)
	if expanded["background-color"] != "red" {
		t.Errorf("background-color = %q", expanded["background-color"])
	}
	if strings.Count(expanded["background-image"], "url(") != 2 {
		t.Errorf("background-image = %q", expanded["background-image"])
	}
	if strings.Count(expanded["background-repeat"], ",") != 1 {
		t.Errorf("background-repeat = %q", expanded["background-repeat"])
	}
}

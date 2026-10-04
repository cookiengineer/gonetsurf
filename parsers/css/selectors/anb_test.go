package selectors

import (
	"encoding/json"
	"os"
	"testing"
)

func TestAnBConformance(t *testing.T) {
	encoded, err := os.ReadFile("testdata/anb.json")
	if err != nil {
		t.Fatalf("read corpus: %v", err)
	}
	var items []any
	if err := json.Unmarshal(encoded, &items); err != nil {
		t.Fatalf("decode corpus: %v", err)
	}
	if len(items)%2 != 0 {
		t.Fatalf("corpus has an odd number of items: %d", len(items))
	}
	for index := 0; index < len(items); index += 2 {
		input, ok := items[index].(string)
		if !ok {
			t.Fatalf("case %d: input is not a string", index/2)
		}
		expected := items[index+1]
		actual, matched := ParseAnB(input)
		if expected == nil {
			if matched {
				t.Errorf("ParseAnB(%q) = %+v, want no match", input, actual)
			}
			continue
		}
		pair, ok := expected.([]any)
		if !ok || len(pair) != 2 {
			t.Fatalf("case %d: bad expected %#v", index/2, expected)
		}
		want := AnB{A: int(pair[0].(float64)), B: int(pair[1].(float64))}
		if !matched {
			t.Errorf("ParseAnB(%q) did not match, want %+v", input, want)
			continue
		}
		if actual != want {
			t.Errorf("ParseAnB(%q) = %+v, want %+v", input, actual, want)
		}
	}
}

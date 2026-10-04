package properties

import (
	"testing"

	"gonetsurf/parsers/css/grammar"
)

func TestLookup(t *testing.T) {
	color, ok := Lookup("color")
	if !ok {
		t.Fatal("color is not registered")
	}
	if !color.Inherited {
		t.Error("color should be inherited")
	}
	if color.Animation != ByComputedValue {
		t.Errorf("color animation = %v", color.Animation)
	}
	if _, ok := Lookup("not-a-property"); ok {
		t.Error("unknown property should not be found")
	}
}

func TestGrammarParses(t *testing.T) {
	for _, name := range Names() {
		property, _ := Lookup(name)
		if _, err := grammar.Parse(property.Grammar); err != nil {
			t.Errorf("property %s has an unparsable grammar %q: %v", name, property.Grammar, err)
		}
	}
}

func TestNamesSorted(t *testing.T) {
	names := Names()
	for index := 1; index < len(names); index++ {
		if names[index-1] > names[index] {
			t.Fatalf("names are not sorted: %q before %q", names[index-1], names[index])
		}
	}
}

func TestRegistrySize(t *testing.T) {
	if Len() < 30 {
		t.Errorf("registry has only %d properties", Len())
	}
}

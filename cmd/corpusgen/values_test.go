package main

import "testing"

func TestExtractValueCallsWithSerialization(t *testing.T) {
	source := `
test_valid_value("display", "flow", "block");
test_valid_value("animation", "1s", ["1s", "1s ease 0s 1 normal none running none"]);
test_valid_value("color", 'red');
test_invalid_value("width", "10");
`
	calls := extractValueCalls(source)
	if len(calls) != 4 {
		t.Fatalf("got %d calls, want 4", len(calls))
	}
	if calls[0].Property != "display" || calls[0].Value != "flow" || !calls[0].Valid {
		t.Errorf("call 0 = %#v", calls[0])
	}
	if len(calls[0].Serialized) != 1 || calls[0].Serialized[0] != "block" {
		t.Errorf("call 0 serialized = %#v", calls[0].Serialized)
	}
	if len(calls[1].Serialized) != 2 || calls[1].Serialized[1] != "1s ease 0s 1 normal none running none" {
		t.Errorf("call 1 serialized = %#v", calls[1].Serialized)
	}
	if len(calls[2].Serialized) != 0 {
		t.Errorf("call 2 serialized = %#v, want none", calls[2].Serialized)
	}
	if calls[3].Valid {
		t.Errorf("call 3 valid = true, want false")
	}
}

func TestExtractValueCallsAcrossLines(t *testing.T) {
	source := `
test_valid_value(
  "font-size-adjust",
  "cap-height calc(0.5 + 1)",
  "cap-height calc(1.5)"
);
`
	calls := extractValueCalls(source)
	if len(calls) != 1 {
		t.Fatalf("got %d calls, want 1", len(calls))
	}
	if calls[0].Property != "font-size-adjust" {
		t.Errorf("property = %q", calls[0].Property)
	}
	if len(calls[0].Serialized) != 1 || calls[0].Serialized[0] != "cap-height calc(1.5)" {
		t.Errorf("serialized = %#v", calls[0].Serialized)
	}
}

func TestExtractSelectorCalls(t *testing.T) {
	source := `
test_valid_selector("a > b");
test_invalid_selector("a >");
`
	calls := extractSelectorCalls(source)
	if len(calls) != 2 {
		t.Fatalf("got %d calls, want 2", len(calls))
	}
	if calls[0].Selector != "a > b" || !calls[0].Valid {
		t.Errorf("call 0 = %#v", calls[0])
	}
	if calls[1].Selector != "a >" || calls[1].Valid {
		t.Errorf("call 1 = %#v", calls[1])
	}
}

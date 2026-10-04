package safecall

import "testing"

func TestCallRecoversPanic(t *testing.T) {
	if err := Call(func() { panic("boom") }); err == nil {
		t.Fatal("expected an error from a panicking function")
	}
	if err := Call(func() {}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValueRecoversPanic(t *testing.T) {
	value, err := Value(func() int { panic("boom") })
	if err == nil {
		t.Fatal("expected an error from a panicking function")
	}
	if value != 0 {
		t.Fatalf("expected zero value, got %d", value)
	}
	result, err := Value(func() int { return 42 })
	if err != nil || result != 42 {
		t.Fatalf("Value() = (%d, %v), want (42, nil)", result, err)
	}
}

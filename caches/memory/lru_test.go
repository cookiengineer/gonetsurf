package memory

import "testing"

func TestLRUEviction(t *testing.T) {
	cache := New[string, int](2)
	cache.Put("a", 1)
	cache.Put("b", 2)
	if _, ok := cache.Get("a"); !ok {
		t.Fatal("expected a")
	}
	cache.Put("c", 3) // evicts b
	if _, ok := cache.Get("b"); ok {
		t.Fatal("expected b to be evicted")
	}
	if value, ok := cache.Get("a"); !ok || value != 1 {
		t.Fatal("expected a to remain")
	}
	if cache.Len() != 2 {
		t.Fatalf("Len = %d, want 2", cache.Len())
	}
}

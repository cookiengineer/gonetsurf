// Package conformance loads the pure-data test corpora used to verify the
// implementation against the pinned specifications.
//
// A corpus is a JSON array of alternating inputs and expected outputs. Many
// upstream corpora (notably the CSS parsing tests) use this shape.
package conformance

import (
	"encoding/json"
	"fmt"
	"os"
)

// Entry pairs one corpus input with its expected output.
type Entry struct {
	Input    any
	Expected any
}

// Load reads a flat [input, expected, ...] JSON array into paired entries.
func Load(path string) ([]Entry, error) {
	items, err := LoadRaw(path)
	if err != nil {
		return nil, err
	}
	if len(items)%2 != 0 {
		return nil, fmt.Errorf("conformance: %s has an odd number of items (%d)", path, len(items))
	}
	entries := make([]Entry, 0, len(items)/2)
	for index := 0; index < len(items); index += 2 {
		entries = append(entries, Entry{Input: items[index], Expected: items[index+1]})
	}
	return entries, nil
}

// LoadRaw reads a JSON array without pairing.
func LoadRaw(path string) ([]any, error) {
	encoded, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []any
	if err := json.Unmarshal(encoded, &items); err != nil {
		return nil, fmt.Errorf("conformance: decode %s: %w", path, err)
	}
	return items, nil
}

// Skip marks a corpus case that is intentionally not required to match,
// normally because the pinned specification and the upstream corpus differ.
type Skip struct {
	Input  string
	Task   string
	Reason string
}

// FindSkip returns the skip entry for input, if any.
func FindSkip(skips []Skip, input string) (Skip, bool) {
	for _, skip := range skips {
		if skip.Input == input {
			return skip, true
		}
	}
	return Skip{}, false
}

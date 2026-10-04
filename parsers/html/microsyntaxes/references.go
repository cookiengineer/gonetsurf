package microsyntaxes

import "strings"

// HashNameReference returns the name portion of a hash-name reference. It
// returns failure when the input does not start with '#' or the name is empty.
func HashNameReference(input string) (string, bool) {
	if len(input) < 2 || input[0] != '#' {
		return "", false
	}
	name := input[1:]
	if name == "" {
		return "", false
	}
	return name, true
}

// ResolveHashNameReference returns the first identifier for which resolve
// returns true, in document order. The resolve function receives each candidate
// name from the reference; callers iterate elements in tree order.
func ResolveHashNameReference(input string, candidates []string) (string, bool) {
	name, ok := HashNameReference(input)
	if !ok {
		return "", false
	}
	for _, candidate := range candidates {
		if strings.EqualFold(candidate, name) {
			return candidate, true
		}
	}
	return "", false
}

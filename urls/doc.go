// Package urls implements the WHATWG URL Standard.
//
// It is the single source of truth for URL parsing, serialization, origin, and
// site computation. It must be used for every security decision; net/url is not
// used on security or cache-key paths.
//
// The parser is written against specifications/web/url.html and is verified
// against the Web Platform Tests urltestdata corpus in testdata/.
package urls

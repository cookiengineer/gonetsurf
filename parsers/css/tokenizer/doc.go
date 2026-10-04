// Package tokenizer implements the CSS Syntax Level 3 tokenizer.
//
// It performs input preprocessing, produces every token kind defined by the
// specification, and preserves each token's Representation for the algorithms
// that depend on the exact source text. The package is verified against the
// tokenization cases of the reference CSS parsing test suite in testdata/.
//
// See specifications/css/syntax-3.html §3.3 and §4.
package tokenizer

package tokenizer

const (
	eofCodePoint         rune = -1
	replacementCodePoint rune = '\uFFFD'
	newlineCodePoint     rune = '\n'
)

// Preprocess filters a stream of code points as required before tokenization,
// per CSS Syntax 3 §3.3:
//
//   - CR, FF, and CRLF pairs become a single LF;
//   - NULL and surrogate code points become U+FFFD.
//
// The input is not modified.
func Preprocess(input []rune) []rune {
	result := make([]rune, 0, len(input))
	for index := 0; index < len(input); index++ {
		current := input[index]
		switch {
		case current == '\r':
			if index+1 < len(input) && input[index+1] == '\n' {
				index++
			}
			result = append(result, '\n')
		case current == '\f':
			result = append(result, '\n')
		case current == 0 || isSurrogate(current):
			result = append(result, replacementCodePoint)
		default:
			result = append(result, current)
		}
	}
	return result
}

// isSurrogate reports whether the code point is in the UTF-16 surrogate range.
func isSurrogate(codePoint rune) bool {
	return codePoint >= 0xD800 && codePoint <= 0xDFFF
}

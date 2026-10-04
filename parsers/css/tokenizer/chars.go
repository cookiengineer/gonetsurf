package tokenizer

// isDigit reports whether the code point is U+0030..U+0039.
func isDigit(codePoint rune) bool {
	return codePoint >= '0' && codePoint <= '9'
}

// isHexDigit reports whether the code point is a hexadecimal digit.
func isHexDigit(codePoint rune) bool {
	return isDigit(codePoint) ||
		(codePoint >= 'A' && codePoint <= 'F') ||
		(codePoint >= 'a' && codePoint <= 'f')
}

// isUppercaseLetter reports whether the code point is A..Z.
func isUppercaseLetter(codePoint rune) bool {
	return codePoint >= 'A' && codePoint <= 'Z'
}

// isLowercaseLetter reports whether the code point is a..z.
func isLowercaseLetter(codePoint rune) bool {
	return codePoint >= 'a' && codePoint <= 'z'
}

// isLetter reports whether the code point is an ASCII letter.
func isLetter(codePoint rune) bool {
	return isUppercaseLetter(codePoint) || isLowercaseLetter(codePoint)
}

// isNonASCIIIdentifier reports whether the code point is a "non-ASCII ident
// code point". The current CSS Syntax 3 editor's draft restricts the set to
// these Unicode ranges rather than admitting every code point above U+007F;
// the definition is shared with the Web Platform Tests suite. The pinned 2021
// Candidate Recommendation predates this refinement and treated all code
// points from U+0080 upward as identifier code points.
func isNonASCIIIdentifier(codePoint rune) bool {
	switch {
	case codePoint == 0xB7:
		return true
	case codePoint >= 0xC0 && codePoint <= 0xD6:
		return true
	case codePoint >= 0xD8 && codePoint <= 0xF6:
		return true
	case codePoint >= 0xF8 && codePoint <= 0x37D:
		return true
	case codePoint >= 0x37F && codePoint <= 0x1FFF:
		return true
	case codePoint >= 0x200C && codePoint <= 0x200D:
		return true
	case codePoint >= 0x203F && codePoint <= 0x2040:
		return true
	case codePoint >= 0x2070 && codePoint <= 0x218F:
		return true
	case codePoint >= 0x2C00 && codePoint <= 0x2FEF:
		return true
	case codePoint >= 0x3001 && codePoint <= 0xD7FF:
		return true
	case codePoint >= 0xF900 && codePoint <= 0xFDCF:
		return true
	case codePoint >= 0xFDF0 && codePoint <= 0xFFFD:
		return true
	case codePoint >= 0x10000 && codePoint <= 0x10FFFF:
		return true
	default:
		return false
	}
}

// isIdentifierStart reports whether the code point may start an identifier.
func isIdentifierStart(codePoint rune) bool {
	return isLetter(codePoint) || isNonASCIIIdentifier(codePoint) || codePoint == '_'
}

// isIdentifierCodePoint reports whether the code point may appear in an
// identifier.
func isIdentifierCodePoint(codePoint rune) bool {
	return isIdentifierStart(codePoint) || isDigit(codePoint) || codePoint == '-'
}

// isNewline reports whether the code point is U+000A LINE FEED.
func isNewline(codePoint rune) bool {
	return codePoint == newlineCodePoint
}

// isWhitespace reports whether the code point is CSS whitespace.
func isWhitespace(codePoint rune) bool {
	return codePoint == newlineCodePoint || codePoint == '\t' || codePoint == ' '
}

// isNonPrintable reports whether the code point is a non-printable code point
// as defined by CSS Syntax 3 §4.2.
func isNonPrintable(codePoint rune) bool {
	switch {
	case codePoint >= 0x00 && codePoint <= 0x08:
		return true
	case codePoint == 0x0B:
		return true
	case codePoint >= 0x0E && codePoint <= 0x1F:
		return true
	case codePoint == 0x7F:
		return true
	default:
		return false
	}
}

// isExponentIndicator reports whether the code point is "e" or "E".
func isExponentIndicator(codePoint rune) bool {
	return codePoint == 'e' || codePoint == 'E'
}

// wouldStartUnicodeRange reports whether the three code points would start a
// unicode-range token. See CSS Syntax 3 §4.3.11.
func wouldStartUnicodeRange(first, second, third rune) bool {
	if first != 'U' && first != 'u' {
		return false
	}
	if second != '+' {
		return false
	}
	return third == '?' || isHexDigit(third)
}

// isValidEscape reports whether first and second form a valid CSS escape. See
// CSS Syntax 3 §4.3.8. A backslash followed by EOF is a valid escape and
// decodes to U+FFFD.
func isValidEscape(first, second rune) bool {
	if first != '\\' {
		return false
	}
	return !isNewline(second)
}

// wouldStartIdentifier reports whether the three code points would start an
// identifier. See CSS Syntax 3 §4.3.9.
func wouldStartIdentifier(first, second, third rune) bool {
	switch {
	case first == '-':
		if isIdentifierStart(second) || second == '-' {
			return true
		}
		return isValidEscape(second, third)
	case isIdentifierStart(first):
		return true
	case first == '\\':
		return isValidEscape(first, second)
	default:
		return false
	}
}

// wouldStartNumber reports whether the three code points would start a number.
// See CSS Syntax 3 §4.3.10.
func wouldStartNumber(first, second, third rune) bool {
	switch {
	case first == '+' || first == '-':
		if isDigit(second) {
			return true
		}
		return second == '.' && isDigit(third)
	case first == '.':
		return isDigit(second)
	case isDigit(first):
		return true
	default:
		return false
	}
}

// asciiEqualFold reports whether the strings are equal under ASCII
// case-insensitive comparison.
func asciiEqualFold(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := 0; index < len(left); index++ {
		leftByte := left[index]
		rightByte := right[index]
		if leftByte == rightByte {
			continue
		}
		if isUppercaseLetter(rune(leftByte)) {
			leftByte += 'a' - 'A'
		}
		if isUppercaseLetter(rune(rightByte)) {
			rightByte += 'a' - 'A'
		}
		if leftByte != rightByte {
			return false
		}
	}
	return true
}

package microsyntaxes

import "strings"

// SimpleColor is an opaque 24-bit RGB color.
type SimpleColor uint32

// Red returns the red component.
func (color SimpleColor) Red() uint8 { return uint8(color >> 16) }

// Green returns the green component.
func (color SimpleColor) Green() uint8 { return uint8(color >> 8) }

// Blue returns the blue component.
func (color SimpleColor) Blue() uint8 { return uint8(color) }

// ParseSimpleColor implements the rules for parsing simple color values. The
// input must be exactly "#rrggbb".
func ParseSimpleColor(input string) (SimpleColor, bool) {
	if len(input) != 7 || input[0] != '#' {
		return 0, false
	}
	for index := 1; index < 7; index++ {
		if !isASCIIHex(input[index]) {
			return 0, false
		}
	}
	red := hexPair(input[1], input[2])
	green := hexPair(input[3], input[4])
	blue := hexPair(input[5], input[6])
	return SimpleColor(uint32(red)<<16 | uint32(green)<<8 | uint32(blue)), true
}

// SerializeSimpleColor implements the rules for serializing simple color
// values.
func SerializeSimpleColor(color SimpleColor) string {
	const digits = "0123456789abcdef"
	buffer := []byte{'#', 0, 0, 0, 0, 0, 0}
	buffer[1] = digits[(color>>20)&0xF]
	buffer[2] = digits[(color>>16)&0xF]
	buffer[3] = digits[(color>>12)&0xF]
	buffer[4] = digits[(color>>8)&0xF]
	buffer[5] = digits[(color>>4)&0xF]
	buffer[6] = digits[color&0xF]
	return string(buffer)
}

// ParseLegacyColor implements the rules for parsing a legacy color value. It
// returns the color and whether parsing succeeded.
func ParseLegacyColor(input string) (SimpleColor, bool) {
	if input == "" {
		return 0, false
	}
	input = strings.Trim(input, " \t\n\f\r")
	if strings.EqualFold(input, "transparent") {
		return 0, false
	}
	if color, ok := NamedColor(input); ok {
		return color, true
	}
	if len(input) == 4 && input[0] == '#' &&
		isASCIIHex(input[1]) && isASCIIHex(input[2]) && isASCIIHex(input[3]) {
		red := hexValue(input[1]) * 17
		green := hexValue(input[2]) * 17
		blue := hexValue(input[3]) * 17
		return SimpleColor(uint32(red)<<16 | uint32(green)<<8 | uint32(blue)), true
	}

	runes := make([]rune, 0, len(input))
	for _, character := range input {
		if character > 0xFFFF {
			runes = append(runes, '0', '0')
			continue
		}
		runes = append(runes, character)
	}
	if len(runes) > 128 {
		runes = runes[:128]
	}
	if len(runes) > 0 && runes[0] == '#' {
		runes = runes[1:]
	}
	for index, character := range runes {
		if !isASCIIHexRune(character) {
			runes[index] = '0'
		}
	}
	for len(runes) == 0 || len(runes)%3 != 0 {
		runes = append(runes, '0')
	}
	length := len(runes) / 3
	red := runes[:length]
	green := runes[length : 2*length]
	blue := runes[2*length:]
	if length > 8 {
		red = red[length-8:]
		green = green[length-8:]
		blue = blue[length-8:]
		length = 8
	}
	for length > 2 && red[0] == '0' && green[0] == '0' && blue[0] == '0' {
		red = red[1:]
		green = green[1:]
		blue = blue[1:]
		length--
	}
	if length > 2 {
		red = red[:2]
		green = green[:2]
		blue = blue[:2]
	}
	return SimpleColor(uint32(parseHexRunes(red))<<16 |
		uint32(parseHexRunes(green))<<8 |
		uint32(parseHexRunes(blue))), true
}

// namedColorTable holds the basic HTML colors. The full CSS color table is
// supplied by the CSS parser (CSS Color 4) in a later phase.
var namedColorTable = map[string]SimpleColor{
	"black":   0x000000,
	"silver":  0xC0C0C0,
	"gray":    0x808080,
	"white":   0xFFFFFF,
	"maroon":  0x800000,
	"red":     0xFF0000,
	"purple":  0x800080,
	"fuchsia": 0xFF00FF,
	"green":   0x008000,
	"lime":    0x00FF00,
	"olive":   0x808000,
	"yellow":  0xFFFF00,
	"navy":    0x000080,
	"blue":    0x0000FF,
	"teal":    0x008080,
	"aqua":    0x00FFFF,
}

// NamedColor returns the color for a CSS named color. Only the basic HTML
// colors are recognized here; the CSS parser provides the full set.
func NamedColor(name string) (SimpleColor, bool) {
	color, ok := namedColorTable[strings.ToLower(strings.TrimSpace(name))]
	return color, ok
}

func isASCIIHex(value byte) bool {
	return (value >= '0' && value <= '9') ||
		(value >= 'a' && value <= 'f') ||
		(value >= 'A' && value <= 'F')
}

func isASCIIHexRune(character rune) bool {
	return (character >= '0' && character <= '9') ||
		(character >= 'a' && character <= 'f') ||
		(character >= 'A' && character <= 'F')
}

func hexValue(value byte) int {
	switch {
	case value >= '0' && value <= '9':
		return int(value - '0')
	case value >= 'a' && value <= 'f':
		return int(value-'a') + 10
	default:
		return int(value-'A') + 10
	}
}

func hexPair(high, low byte) int {
	return hexValue(high)<<4 | hexValue(low)
}

func parseHexRunes(runes []rune) int {
	value := 0
	for _, character := range runes {
		digit := 0
		switch {
		case character >= '0' && character <= '9':
			digit = int(character - '0')
		case character >= 'a' && character <= 'f':
			digit = int(character-'a') + 10
		case character >= 'A' && character <= 'F':
			digit = int(character-'A') + 10
		}
		value = value<<4 | digit
	}
	return value
}

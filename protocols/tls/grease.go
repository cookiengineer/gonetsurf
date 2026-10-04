package tls

// GreaseValue returns the RFC 8701 GREASE value at index (0 to 15). The values
// are 0x0A0A, 0x1A1A, 0x2A2A, ... 0xFAFA.
//
// Go's crypto/tls does not emit GREASE or allow ClientHello customization, so
// these values are provided for the future custom ClientHello transport.
func GreaseValue(index int) uint16 {
	index &= 0x0F
	return 0x0A0A + uint16(index)*0x1010
}

// GreaseValues returns all sixteen GREASE values in order.
func GreaseValues() []uint16 {
	values := make([]uint16, 16)
	for index := range values {
		values[index] = GreaseValue(index)
	}
	return values
}

// IsGrease reports whether a value is one of the reserved GREASE values.
func IsGrease(value uint16) bool {
	if value&0x0F0F != 0x0A0A {
		return false
	}
	return value>>8 == value&0xFF
}

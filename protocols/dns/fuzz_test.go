package dns

import "testing"

func FuzzParseMessage(f *testing.F) {
	message := &Message{
		ID:    0x1234,
		Flags: 0x0100,
		Questions: []Question{{
			Name:  "example.com",
			Type:  TypeA,
			Class: ClassIN,
		}},
	}
	seed, err := message.Pack()
	if err == nil {
		f.Add(seed)
	}
	f.Add([]byte{})
	f.Add([]byte{0x00, 0x01, 0x02})
	f.Fuzz(func(t *testing.T, data []byte) {
		parsed, err := ParseMessage(data)
		if err != nil {
			return
		}
		packed, err := parsed.Pack()
		if err != nil {
			return
		}
		_, _ = ParseMessage(packed)
	})
}

package dns

import (
	"context"
	"net"
	"net/netip"
	"testing"
	"time"
)

func TestMessageRoundTrip(t *testing.T) {
	message := &Message{
		ID:    0x1234,
		Flags: FlagResponse | FlagRecursionAvailable,
		Questions: []Question{
			{Name: "example.com", Type: TypeA, Class: ClassIN},
		},
		Answers: []RR{
			{Name: "example.com", Type: TypeA, Class: ClassIN, TTL: 300, Address: netip.MustParseAddr("93.184.216.34")},
			{Name: "example.com", Type: TypeCNAME, Class: ClassIN, TTL: 300, Target: "alias.example.com"},
		},
	}
	packed, err := message.Pack()
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := ParseMessage(packed)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.ID != 0x1234 || len(parsed.Answers) != 2 {
		t.Fatalf("parsed = %+v", parsed)
	}
	if parsed.Answers[0].Address != netip.MustParseAddr("93.184.216.34") {
		t.Errorf("address = %v", parsed.Answers[0].Address)
	}
	if parsed.Answers[1].Target != "alias.example.com" {
		t.Errorf("target = %q", parsed.Answers[1].Target)
	}
}

func TestCompressionPointer(t *testing.T) {
	message := &Message{
		ID:    1,
		Flags: FlagResponse,
		Questions: []Question{
			{Name: "example.com", Type: TypeA, Class: ClassIN},
		},
		Answers: []RR{
			{Name: "example.com", Type: TypeA, Class: ClassIN, TTL: 60, Address: netip.MustParseAddr("1.2.3.4")},
		},
	}
	packed, _ := message.Pack()
	// Manually craft a response using a compression pointer for the answer name.
	pointer := []byte{0xC0, 0x0C}
	encodedName, _ := encodeName("example.com")
	questionEnd := 12 + len(encodedName) + 4
	answer := append([]byte{}, pointer...)
	answer = append(answer, 0x00, 0x01, 0x00, 0x01, 0x00, 0x00, 0x00, 0x3C, 0x00, 0x04, 1, 2, 3, 4)
	full := append([]byte{}, packed[:questionEnd]...)
	full = append(full, answer...)
	// Fix answer count to 1.
	full[6], full[7] = 0, 1
	parsed, err := ParseMessage(full)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Answers[0].Name != "example.com" {
		t.Errorf("compressed name = %q", parsed.Answers[0].Name)
	}
}

func TestResolverSpecials(t *testing.T) {
	resolver := NewResolver()
	addresses, _, err := resolver.LookupHost(context.Background(), "localhost")
	if err != nil || len(addresses) != 2 {
		t.Fatalf("localhost = (%v, %v)", addresses, err)
	}
	if _, _, err := resolver.LookupHost(context.Background(), "something.invalid"); err == nil {
		t.Error("expected invalid to fail")
	}
}

func TestResolverUDP(t *testing.T) {
	connection, err := net.ListenPacket("udp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer connection.Close()
	go func() {
		buffer := make([]byte, 4096)
		for {
			count, address, err := connection.ReadFrom(buffer)
			if err != nil {
				return
			}
			query, err := ParseMessage(buffer[:count])
			if err != nil {
				continue
			}
			response := &Message{
				ID:        query.ID,
				Flags:     FlagResponse | FlagRecursionAvailable,
				Questions: query.Questions,
				Answers: []RR{{
					Name:    query.Questions[0].Name,
					Type:    TypeA,
					Class:   ClassIN,
					TTL:     30,
					Address: netip.MustParseAddr("10.0.0.1"),
				}},
			}
			packed, _ := response.Pack()
			connection.WriteTo(packed, address)
		}
	}()

	resolver := NewResolver()
	resolver.Servers = []string{connection.LocalAddr().String()}
	resolver.Timeout = 2 * time.Second
	addresses, _, err := resolver.LookupHost(context.Background(), "example.com")
	if err != nil {
		t.Fatal(err)
	}
	if len(addresses) != 1 || addresses[0] != netip.MustParseAddr("10.0.0.1") {
		t.Fatalf("addresses = %v", addresses)
	}
	// The second lookup should be served from the TTL cache.
	addresses, _, err = resolver.LookupHost(context.Background(), "example.com")
	if err != nil || len(addresses) != 1 {
		t.Fatalf("cached lookup = (%v, %v)", addresses, err)
	}
}

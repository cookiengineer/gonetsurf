package dns

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"

	ourhttp "gonetsurf/protocols/http1"
)

func TestExchangeDoH(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodPost {
			writer.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		body, err := io.ReadAll(request.Body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		query, err := ParseMessage(body)
		if err != nil {
			writer.WriteHeader(http.StatusBadRequest)
			return
		}
		response := &Message{
			ID:        query.ID,
			Flags:     FlagResponse | FlagRecursionAvailable,
			Questions: query.Questions,
			Answers: []RR{{
				Name:    query.Questions[0].Name,
				Type:    TypeA,
				Class:   ClassIN,
				TTL:     60,
				Address: netip.MustParseAddr("203.0.113.7"),
			}},
		}
		packed, _ := response.Pack()
		writer.Header().Set("Content-Type", "application/dns-message")
		writer.Write(packed)
	}))
	defer server.Close()

	resolver := NewResolver()
	message, err := resolver.ExchangeDoH(context.Background(), "example.com", TypeA, server.URL, &ourhttp.Client{Transport: &ourhttp.Transport{}})
	if err != nil {
		t.Fatal(err)
	}
	if len(message.Answers) != 1 || message.Answers[0].Address != netip.MustParseAddr("203.0.113.7") {
		t.Fatalf("answers = %+v", message.Answers)
	}
}

func TestDoHGetURL(t *testing.T) {
	got := DoHGetURL("https://dns.example.com/dns-query", []byte{0x00, 0x01})
	if got != "https://dns.example.com/dns-query?dns=AAE" {
		t.Errorf("DoHGetURL = %q", got)
	}
}

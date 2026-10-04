package dns

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/binary"
	"io"
	"math/big"
	"net"
	"net/netip"
	"testing"
	"time"
)

// TestExchangeDoT exercises DNS-over-TLS against a local TLS server.
func TestExchangeDoT(t *testing.T) {
	certificate := generateDoTCertificate(t)
	listener, err := tls.Listen("tcp", "127.0.0.1:0", &tls.Config{
		Certificates: []tls.Certificate{certificate},
		MinVersion:   tls.VersionTLS13,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()

	serverError := make(chan error, 1)
	go func() {
		serverError <- serveDoT(listener)
	}()

	roots := x509.NewCertPool()
	roots.AddCert(certificate.Leaf)
	config := &tls.Config{
		ServerName: "localhost",
		RootCAs:    roots,
		MinVersion: tls.VersionTLS13,
	}
	resolver := NewResolver()
	message, err := resolver.ExchangeDoT(context.Background(), "example.com", TypeA, listener.Addr().String(), config)
	if err != nil {
		t.Fatalf("ExchangeDoT: %v", err)
	}
	if err := <-serverError; err != nil {
		t.Fatalf("server: %v", err)
	}
	if !message.IsResponse() {
		t.Error("expected a response message")
	}
	if len(message.Answers) != 1 {
		t.Fatalf("got %d answers, want 1", len(message.Answers))
	}
	if message.Answers[0].Address != netip.MustParseAddr("93.184.216.34") {
		t.Errorf("answer address = %v", message.Answers[0].Address)
	}
}

func serveDoT(listener net.Listener) error {
	connection, err := listener.Accept()
	if err != nil {
		return err
	}
	defer connection.Close()
	lengthBuffer := make([]byte, 2)
	if _, err := io.ReadFull(connection, lengthBuffer); err != nil {
		return err
	}
	length := int(binary.BigEndian.Uint16(lengthBuffer))
	queryBytes := make([]byte, length)
	if _, err := io.ReadFull(connection, queryBytes); err != nil {
		return err
	}
	query, err := ParseMessage(queryBytes)
	if err != nil {
		return err
	}
	name := ""
	if len(query.Questions) > 0 {
		name = query.Questions[0].Name
	}
	response := &Message{
		ID:        query.ID,
		Flags:     FlagResponse | FlagRecursionAvailable,
		Questions: query.Questions,
		Answers: []RR{{
			Name:    name,
			Type:    TypeA,
			Class:   ClassIN,
			TTL:     60,
			Address: netip.MustParseAddr("93.184.216.34"),
		}},
	}
	packed, err := response.Pack()
	if err != nil {
		return err
	}
	framed := make([]byte, 2+len(packed))
	binary.BigEndian.PutUint16(framed[:2], uint16(len(packed)))
	copy(framed[2:], packed)
	_, err = connection.Write(framed)
	return err
}

func generateDoTCertificate(t *testing.T) tls.Certificate {
	t.Helper()
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "localhost"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
		DNSNames:     []string{"localhost"},
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	leaf, err := x509.ParseCertificate(der)
	if err != nil {
		t.Fatal(err)
	}
	return tls.Certificate{Certificate: [][]byte{der}, PrivateKey: privateKey, Leaf: leaf}
}

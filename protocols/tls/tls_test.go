package tls

import (
	"context"
	cryptotls "crypto/tls"
	"crypto/x509"
	"io"
	"net"
	"testing"
)

func TestGrease(t *testing.T) {
	values := GreaseValues()
	if values[0] != 0x0A0A || values[15] != 0xFAFA {
		t.Fatalf("GreaseValues = %#x", values)
	}
	if !IsGrease(0x2A2A) {
		t.Error("expected 0x2A2A to be GREASE")
	}
	if IsGrease(0x1234) {
		t.Error("expected 0x1234 not to be GREASE")
	}
}

func TestNewClientConfig(t *testing.T) {
	config := NewClientConfig(Options{ServerName: "example.com"})
	if config.MinVersion != cryptotls.VersionTLS13 {
		t.Error("expected TLS 1.3 minimum")
	}
	if len(config.NextProtos) == 0 || config.NextProtos[0] != ALPNHTTP11 {
		t.Errorf("NextProtos = %v", config.NextProtos)
	}
}

func TestDialWithLocalServer(t *testing.T) {
	certificate := generateCertificate(t)
	server := cryptotls.NewListener(mustListen(t), &cryptotls.Config{
		Certificates: []cryptotls.Certificate{certificate},
		MinVersion:   cryptotls.VersionTLS13,
	})
	defer server.Close()
	go func() {
		connection, err := server.Accept()
		if err != nil {
			return
		}
		io.Copy(io.Discard, connection)
		connection.Close()
	}()

	pool := x509.NewCertPool()
	pool.AddCert(certificate.Leaf)
	config := NewClientConfig(Options{ServerName: "localhost"})
	config.RootCAs = pool
	connection, err := Dial(context.Background(), &net.Dialer{}, server.Addr().String(), config)
	if err != nil {
		t.Fatal(err)
	}
	connection.Close()
}

func mustListen(t *testing.T) net.Listener {
	t.Helper()
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return listener
}

// Package tls configures TLS 1.3 connections with privacy-preserving defaults.
// See specs/other/rfc8446.txt and rfc8701.txt.
package tls

import (
	"context"
	cryptotls "crypto/tls"
	"net"
)

// Application-layer protocol identifiers.
const (
	ALPNHTTP11 = "http/1.1"
	ALPNHTTP2  = "h2"
)

// Dialer establishes raw network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

// Options configures a client TLS connection.
type Options struct {
	ServerName         string
	NextProtos         []string
	ECHConfigList      []byte
	InsecureSkipVerify bool
}

// NewClientConfig returns a TLS client configuration with TLS 1.3 as the
// minimum version, ALPN, and 0-RTT disabled. The caller owns the returned
// config.
func NewClientConfig(options Options) *cryptotls.Config {
	nextProtos := options.NextProtos
	if len(nextProtos) == 0 {
		nextProtos = []string{ALPNHTTP11}
	}
	config := &cryptotls.Config{
		ServerName:             options.ServerName,
		MinVersion:             cryptotls.VersionTLS13,
		NextProtos:             nextProtos,
		InsecureSkipVerify:     options.InsecureSkipVerify,
		SessionTicketsDisabled: false,
		ClientSessionCache:     cryptotls.NewLRUClientSessionCache(64),
	}
	if len(options.ECHConfigList) > 0 {
		config.EncryptedClientHelloConfigList = options.ECHConfigList
	}
	return config
}

// Dial establishes a TLS connection to address using the dialer.
func Dial(ctx context.Context, dialer Dialer, address string, config *cryptotls.Config) (net.Conn, error) {
	rawConnection, err := dialer.DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	connection := cryptotls.Client(rawConnection, config)
	if err := connection.HandshakeContext(ctx); err != nil {
		rawConnection.Close()
		return nil, err
	}
	return connection, nil
}

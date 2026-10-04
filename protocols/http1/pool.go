package http1

import (
	"bufio"
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"time"

	"gonetsurf/urls"
)

// Dialer establishes network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

var defaultDialer = &net.Dialer{Timeout: 30 * time.Second}

type idleConnection struct {
	connection net.Conn
	since      time.Time
}

// Transport performs HTTP/1.1 round trips with connection pooling.
type Transport struct {
	Dialer         Dialer
	TLSConfig      *tls.Config
	MaxIdlePerHost int
	IdleTimeout    time.Duration
	// DisableKeepAlives forces a new connection per request.
	DisableKeepAlives bool

	mutex sync.Mutex
	idle  map[string][]idleConnection
}

func (transport *Transport) dialer() Dialer {
	if transport.Dialer != nil {
		return transport.Dialer
	}
	return defaultDialer
}

func (transport *Transport) maxIdle() int {
	if transport.MaxIdlePerHost > 0 {
		return transport.MaxIdlePerHost
	}
	return 2
}

func (transport *Transport) idleTimeout() time.Duration {
	if transport.IdleTimeout > 0 {
		return transport.IdleTimeout
	}
	return 90 * time.Second
}

// RoundTrip performs a single request/response exchange, reusing pooled
// connections when possible.
func (transport *Transport) RoundTrip(ctx context.Context, request *Request) (*Response, error) {
	address, useTLS := addressFor(request.URL)
	connection, err := transport.connection(ctx, address, useTLS, request.URL.Host)
	if err != nil {
		return nil, err
	}
	writer := bufio.NewWriter(connection)
	if err := WriteRequest(writer, request); err != nil {
		connection.Close()
		return nil, err
	}
	reader := bufio.NewReader(connection)
	response, err := ReadResponse(reader, request)
	if err != nil {
		connection.Close()
		return nil, err
	}
	reusable := !transport.DisableKeepAlives && !responseShouldClose(response, request)
	response.Body = &pooledBody{
		inner:      response.Body,
		transport:  transport,
		address:    address,
		connection: connection,
		reader:     reader,
		reusable:   reusable,
	}
	return response, nil
}

func (transport *Transport) connection(ctx context.Context, address string, useTLS bool, host string) (net.Conn, error) {
	if !transport.DisableKeepAlives {
		if connection := transport.takeIdle(address); connection != nil {
			return connection, nil
		}
	}
	rawConnection, err := transport.dialer().DialContext(ctx, "tcp", address)
	if err != nil {
		return nil, err
	}
	if !useTLS {
		return rawConnection, nil
	}
	config := transport.TLSConfig
	if config != nil {
		config = config.Clone()
	} else {
		config = &tls.Config{}
	}
	if config.ServerName == "" {
		config.ServerName = host
	}
	tlsConnection := tls.Client(rawConnection, config)
	if err := tlsConnection.HandshakeContext(ctx); err != nil {
		rawConnection.Close()
		return nil, err
	}
	return tlsConnection, nil
}

func (transport *Transport) takeIdle(address string) net.Conn {
	transport.mutex.Lock()
	defer transport.mutex.Unlock()
	if transport.idle == nil {
		return nil
	}
	connections := transport.idle[address]
	for len(connections) > 0 {
		entry := connections[len(connections)-1]
		connections = connections[:len(connections)-1]
		if time.Since(entry.since) > transport.idleTimeout() {
			entry.connection.Close()
			continue
		}
		transport.idle[address] = connections
		return entry.connection
	}
	transport.idle[address] = connections
	return nil
}

func (transport *Transport) putIdle(address string, connection net.Conn) {
	transport.mutex.Lock()
	defer transport.mutex.Unlock()
	if transport.idle == nil {
		transport.idle = make(map[string][]idleConnection)
	}
	connections := transport.idle[address]
	if len(connections) >= transport.maxIdle() {
		connection.Close()
		return
	}
	transport.idle[address] = append(connections, idleConnection{connection: connection, since: time.Now()})
}

// CloseIdle closes all pooled connections.
func (transport *Transport) CloseIdle() {
	transport.mutex.Lock()
	defer transport.mutex.Unlock()
	for address, connections := range transport.idle {
		for _, entry := range connections {
			entry.connection.Close()
		}
		delete(transport.idle, address)
	}
}

func addressFor(url *urls.URL) (address string, useTLS bool) {
	host := url.Host
	port := url.Port
	switch url.Scheme {
	case "https", "wss":
		useTLS = true
		if port == "" {
			port = "443"
		}
	default:
		if port == "" {
			port = "80"
		}
	}
	return net.JoinHostPort(host, port), useTLS
}

func responseShouldClose(response *Response, request *Request) bool {
	if tokenListContains(response.Header.Values("Connection"), "close") {
		return true
	}
	if tokenListContains(request.Header.Values("Connection"), "close") {
		return true
	}
	if response.Proto == "HTTP/1.0" && !tokenListContains(response.Header.Values("Connection"), "keep-alive") {
		return true
	}
	return false
}

func tokenListContains(values []string, token string) bool {
	for _, value := range values {
		for _, part := range splitList(value) {
			if part == token {
				return true
			}
		}
	}
	return false
}

type pooledBody struct {
	inner      io.ReadCloser
	transport  *Transport
	address    string
	connection net.Conn
	reader     *bufio.Reader
	reusable   bool
	closed     bool
}

func (body *pooledBody) Read(target []byte) (int, error) {
	count, err := body.inner.Read(target)
	if err == io.EOF && body.reusable {
		body.transport.putIdle(body.address, body.connection)
		body.reusable = false
		body.closed = true
	}
	return count, err
}

func (body *pooledBody) Close() error {
	if body.closed {
		return nil
	}
	body.closed = true
	if body.reusable {
		body.transport.putIdle(body.address, body.connection)
		body.reusable = false
		return nil
	}
	return body.inner.Close()
}

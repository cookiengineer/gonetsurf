package dns

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"strconv"

	ourhttp "gonetsurf/protocols/http1"
	ourtls "gonetsurf/protocols/tls"
	"gonetsurf/urls"
)

// Query builds a query message with a random ID.
func Query(name string, recordType Type) (*Message, error) {
	return &Message{
		ID:    randomID(),
		Flags: FlagRecursionDesired,
		Questions: []Question{{
			Name:  normalizeName(name),
			Type:  recordType,
			Class: ClassIN,
		}},
	}, nil
}

// ExchangeDoT performs a DNS-over-TLS query (RFC 7858).
func (resolver *Resolver) ExchangeDoT(ctx context.Context, name string, recordType Type, server string, config *tls.Config) (*Message, error) {
	query, err := Query(name, recordType)
	if err != nil {
		return nil, err
	}
	packed, err := query.Pack()
	if err != nil {
		return nil, err
	}
	connection, err := ourtls.Dial(ctx, resolver.dialer(), server, config)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	connection.SetDeadline(resolver.deadline(ctx))
	framed := make([]byte, 2+len(packed))
	binary.BigEndian.PutUint16(framed[:2], uint16(len(packed)))
	copy(framed[2:], packed)
	if _, err := connection.Write(framed); err != nil {
		return nil, err
	}
	lengthBuffer := make([]byte, 2)
	if _, err := io.ReadFull(connection, lengthBuffer); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint16(lengthBuffer))
	response := make([]byte, length)
	if _, err := io.ReadFull(connection, response); err != nil {
		return nil, err
	}
	return ParseMessage(response)
}

// ExchangeDoH performs a DNS-over-HTTPS query (RFC 8484) using POST.
func (resolver *Resolver) ExchangeDoH(ctx context.Context, name string, recordType Type, endpoint string, client *ourhttp.Client) (*Message, error) {
	query, err := Query(name, recordType)
	if err != nil {
		return nil, err
	}
	// RFC 8484 recommends an ID of 0 for cache friendliness over DoH.
	query.ID = 0
	packed, err := query.Pack()
	if err != nil {
		return nil, err
	}
	endpointURL, err := urls.Parse(endpoint)
	if err != nil {
		return nil, err
	}
	request := &ourhttp.Request{
		Method: http.MethodPost,
		URL:    endpointURL,
		Proto:  "HTTP/1.1",
		Header: ourhttp.Header{
			{Name: "Content-Type", Value: "application/dns-message"},
			{Name: "Accept", Value: "application/dns-message"},
			{Name: "Content-Length", Value: strconv.Itoa(len(packed))},
		},
		Body: io.NopCloser(bytes.NewReader(packed)),
	}
	if client == nil {
		client = &ourhttp.Client{Transport: &ourhttp.Transport{}}
	}
	response, err := client.Do(ctx, request)
	if err != nil {
		return nil, err
	}
	defer response.Close()
	if response.StatusCode != http.StatusOK {
		return nil, ErrMalformed
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 64*1024))
	if err != nil {
		return nil, err
	}
	return ParseMessage(body)
}

// DoHGetURL builds a DNS-over-HTTPS GET URL with the base64url-encoded query.
func DoHGetURL(endpoint string, packed []byte) string {
	encoded := base64.RawURLEncoding.EncodeToString(packed)
	separator := "?"
	if bytes.ContainsRune([]byte(endpoint), '?') {
		separator = "&"
	}
	return endpoint + separator + "dns=" + encoded
}

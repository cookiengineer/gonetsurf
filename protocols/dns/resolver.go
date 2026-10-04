package dns

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"net"
	"net/netip"
	"strings"
	"sync"
	"time"
)

// ErrNameNotFound is returned when a name does not exist.
var ErrNameNotFound = errors.New("dns: name not found")

// Dialer establishes network connections.
type Dialer interface {
	DialContext(ctx context.Context, network, address string) (net.Conn, error)
}

type cacheEntry struct {
	records  []RR
	target   string
	expires  time.Time
	negative bool
}

// Resolver resolves DNS names with an in-memory TTL cache.
type Resolver struct {
	Dialer  Dialer
	Servers []string
	Timeout time.Duration

	mutex sync.Mutex
	cache map[string]cacheEntry
}

// NewResolver returns a resolver with default servers.
func NewResolver() *Resolver {
	return &Resolver{
		Dialer:  &net.Dialer{},
		Servers: []string{"1.1.1.1:53", "8.8.8.8:53"},
		Timeout: 5 * time.Second,
		cache:   make(map[string]cacheEntry),
	}
}

// LookupHost resolves a name to addresses, following CNAME chains.
func (resolver *Resolver) LookupHost(ctx context.Context, name string) ([]netip.Addr, string, error) {
	name = normalizeName(name)
	if addresses, cname, handled, err := specialUseName(name); handled {
		return addresses, cname, err
	}

	var addresses []netip.Addr
	var canonical string
	originalName := name
	for depth := 0; depth < 8; depth++ {
		records, target, err := resolver.lookup(ctx, name, TypeA)
		if err != nil {
			return nil, "", err
		}
		if target != "" && target != name {
			canonical = target
			name = target
			continue
		}
		for _, record := range records {
			if record.Type == TypeA && record.Address.IsValid() {
				addresses = append(addresses, record.Address)
			}
		}
		break
	}
	if len(addresses) == 0 {
		// Try AAAA as a fallback.
		if records, _, err := resolver.lookup(ctx, name, TypeAAAA); err == nil {
			for _, record := range records {
				if record.Type == TypeAAAA && record.Address.IsValid() {
					addresses = append(addresses, record.Address)
				}
			}
		}
	}
	if canonical == "" {
		canonical = name
	}
	if len(addresses) == 0 {
		return nil, canonical, ErrNameNotFound
	}
	_ = originalName
	return addresses, canonical, nil
}

// CNAME returns the CNAME chain for a name, if any.
func (resolver *Resolver) CNAME(ctx context.Context, name string) (string, error) {
	records, _, err := resolver.lookup(ctx, normalizeName(name), TypeCNAME)
	if err != nil {
		return "", err
	}
	for _, record := range records {
		if record.Type == TypeCNAME {
			return record.Target, nil
		}
	}
	return "", nil
}

func (resolver *Resolver) lookup(ctx context.Context, name string, recordType Type) ([]RR, string, error) {
	key := name + "|" + typeString(recordType)
	if entry, ok := resolver.fromCache(key); ok {
		if entry.negative {
			return nil, entry.target, ErrNameNotFound
		}
		return entry.records, entry.target, nil
	}

	response, err := resolver.exchange(ctx, name, recordType)
	if err != nil {
		return nil, "", err
	}
	switch response.ResponseCode() {
	case RCodeNameError:
		resolver.toCache(key, cacheEntry{negative: true, expires: negativeExpiry(response, resolver.now())})
		return nil, "", ErrNameNotFound
	case RCodeSuccess:
	default:
		return nil, "", errors.New("dns: server failure")
	}

	var records []RR
	target := ""
	ttl := uint32(60)
	for _, record := range response.Answers {
		if record.Type == recordType {
			records = append(records, record)
		}
		if record.Type == TypeCNAME && target == "" {
			target = record.Target
		}
		if record.TTL > 0 && record.TTL < ttl {
			ttl = record.TTL
		}
	}
	if len(records) == 0 && target == "" {
		resolver.toCache(key, cacheEntry{negative: true, expires: negativeExpiry(response, resolver.now())})
		return nil, "", ErrNameNotFound
	}
	resolver.toCache(key, cacheEntry{records: records, target: target, expires: resolver.now().Add(time.Duration(ttl) * time.Second)})
	return records, target, nil
}

func (resolver *Resolver) exchange(ctx context.Context, name string, recordType Type) (*Message, error) {
	query := &Message{
		ID:    randomID(),
		Flags: FlagRecursionDesired,
		Questions: []Question{{
			Name:  name,
			Type:  recordType,
			Class: ClassIN,
		}},
	}
	packed, err := query.Pack()
	if err != nil {
		return nil, err
	}

	var lastError error
	for _, server := range resolver.Servers {
		responseData, err := resolver.queryUDP(ctx, server, packed)
		if err != nil {
			lastError = err
			continue
		}
		response, err := ParseMessage(responseData)
		if err != nil {
			lastError = err
			continue
		}
		if response.ID != query.ID {
			lastError = ErrMalformed
			continue
		}
		if response.Flags&FlagTruncated != 0 {
			if responseData, err = resolver.queryTCP(ctx, server, packed); err != nil {
				lastError = err
				continue
			}
			if response, err = ParseMessage(responseData); err != nil {
				lastError = err
				continue
			}
		}
		return response, nil
	}
	if lastError == nil {
		lastError = errors.New("dns: no servers configured")
	}
	return nil, lastError
}

func (resolver *Resolver) queryUDP(ctx context.Context, server string, query []byte) ([]byte, error) {
	connection, err := resolver.dialer().DialContext(ctx, "udp", server)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	deadline := resolver.deadline(ctx)
	connection.SetDeadline(deadline)
	if _, err := connection.Write(query); err != nil {
		return nil, err
	}
	buffer := make([]byte, 4096)
	count, err := connection.Read(buffer)
	if err != nil {
		return nil, err
	}
	return buffer[:count], nil
}

func (resolver *Resolver) queryTCP(ctx context.Context, server string, query []byte) ([]byte, error) {
	connection, err := resolver.dialer().DialContext(ctx, "tcp", server)
	if err != nil {
		return nil, err
	}
	defer connection.Close()
	deadline := resolver.deadline(ctx)
	connection.SetDeadline(deadline)
	framed := make([]byte, 2+len(query))
	binary.BigEndian.PutUint16(framed[:2], uint16(len(query)))
	copy(framed[2:], query)
	if _, err := connection.Write(framed); err != nil {
		return nil, err
	}
	lengthBuffer := make([]byte, 2)
	if _, err := readFull(connection, lengthBuffer); err != nil {
		return nil, err
	}
	length := int(binary.BigEndian.Uint16(lengthBuffer))
	response := make([]byte, length)
	if _, err := readFull(connection, response); err != nil {
		return nil, err
	}
	return response, nil
}

func (resolver *Resolver) dialer() Dialer {
	if resolver.Dialer != nil {
		return resolver.Dialer
	}
	return &net.Dialer{}
}

func (resolver *Resolver) deadline(ctx context.Context) time.Time {
	timeout := resolver.Timeout
	if timeout == 0 {
		timeout = 5 * time.Second
	}
	if contextDeadline, ok := ctx.Deadline(); ok {
		if contextDeadline.Before(time.Now().Add(timeout)) {
			return contextDeadline
		}
	}
	return time.Now().Add(timeout)
}

func (resolver *Resolver) now() time.Time {
	return time.Now()
}

func (resolver *Resolver) fromCache(key string) (cacheEntry, bool) {
	resolver.mutex.Lock()
	defer resolver.mutex.Unlock()
	entry, ok := resolver.cache[key]
	if !ok || resolver.now().After(entry.expires) {
		return cacheEntry{}, false
	}
	return entry, true
}

func (resolver *Resolver) toCache(key string, entry cacheEntry) {
	resolver.mutex.Lock()
	defer resolver.mutex.Unlock()
	resolver.cache[key] = entry
}

func negativeExpiry(response *Message, now time.Time) time.Time {
	ttl := uint32(60)
	for _, record := range response.Authorities {
		if record.Type == TypeSOA && record.TTL > 0 && record.TTL < ttl {
			ttl = record.TTL
		}
	}
	return now.Add(time.Duration(ttl) * time.Second)
}

func normalizeName(name string) string {
	return strings.ToLower(strings.TrimSuffix(name, "."))
}

func randomID() uint16 {
	var buffer [2]byte
	if _, err := rand.Read(buffer[:]); err != nil {
		return 0
	}
	return binary.BigEndian.Uint16(buffer[:])
}

func readFull(connection net.Conn, buffer []byte) (int, error) {
	total := 0
	for total < len(buffer) {
		count, err := connection.Read(buffer[total:])
		total += count
		if err != nil {
			return total, err
		}
	}
	return total, nil
}

func typeString(recordType Type) string {
	switch recordType {
	case TypeA:
		return "A"
	case TypeAAAA:
		return "AAAA"
	case TypeCNAME:
		return "CNAME"
	case TypeMX:
		return "MX"
	case TypeTXT:
		return "TXT"
	case TypeSRV:
		return "SRV"
	default:
		return "OTHER"
	}
}

func specialUseName(name string) ([]netip.Addr, string, bool, error) {
	switch {
	case name == "localhost":
		return []netip.Addr{netip.MustParseAddr("127.0.0.1"), netip.MustParseAddr("::1")}, "localhost", true, nil
	case name == "invalid" || strings.HasSuffix(name, ".invalid"):
		return nil, name, true, ErrNameNotFound
	case name == "test" || strings.HasSuffix(name, ".test"):
		return nil, name, true, ErrNameNotFound
	default:
		return nil, "", false, nil
	}
}

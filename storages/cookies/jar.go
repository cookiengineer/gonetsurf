package cookies

import (
	"bufio"
	"encoding/json"
	"os"
	"sort"
	"sync"
	"time"
)

// Jar is a partitioned cookie store.
type Jar struct {
	mutex   sync.RWMutex
	entries map[string]map[Key]Cookie
}

// NewJar returns an empty cookie jar.
func NewJar() *Jar {
	return &Jar{entries: make(map[string]map[Key]Cookie)}
}

// Set stores or replaces a cookie in a partition.
func (jar *Jar) Set(partition string, cookie Cookie) {
	jar.mutex.Lock()
	defer jar.mutex.Unlock()
	if jar.entries[partition] == nil {
		jar.entries[partition] = make(map[Key]Cookie)
	}
	jar.entries[partition][cookie.Key()] = cookie
}

// CookiesFor returns the cookies that apply to a request, ordered by path
// length descending and creation time ascending.
func (jar *Jar) CookiesFor(partition, requestHost, requestPath string, isSecure bool, now time.Time) []Cookie {
	jar.mutex.RLock()
	defer jar.mutex.RUnlock()
	var matched []Cookie
	for _, cookie := range jar.entries[partition] {
		if cookie.matchesAt(requestHost, requestPath, isSecure, now) {
			matched = append(matched, cookie)
		}
	}
	sort.SliceStable(matched, func(first, second int) bool {
		if len(matched[first].Path) != len(matched[second].Path) {
			return len(matched[first].Path) > len(matched[second].Path)
		}
		return matched[first].CreatedAt.Before(matched[second].CreatedAt)
	})
	return matched
}

func (cookie Cookie) matchesAt(requestHost, requestPath string, isSecure bool, now time.Time) bool {
	if cookie.HasExpired(now) {
		return false
	}
	if cookie.HostOnly {
		if !equalFold(requestHost, cookie.Domain) {
			return false
		}
	} else if !domainMatches(requestHost, cookie.Domain) {
		return false
	}
	if !pathMatches(requestPath, cookie.Path) {
		return false
	}
	if cookie.Secure && !isSecure {
		return false
	}
	return true
}

// Delete removes a cookie.
func (jar *Jar) Delete(partition string, key Key) {
	jar.mutex.Lock()
	defer jar.mutex.Unlock()
	delete(jar.entries[partition], key)
}

// DeleteDomain removes all cookies for a domain and its subdomains.
func (jar *Jar) DeleteDomain(partition, domain string) int {
	jar.mutex.Lock()
	defer jar.mutex.Unlock()
	deleted := 0
	for key, cookie := range jar.entries[partition] {
		if domainMatches(cookie.Domain, domain) || equalFold(cookie.Domain, domain) {
			delete(jar.entries[partition], key)
			deleted++
		}
	}
	return deleted
}

// DeleteExpired removes expired cookies and returns the count removed.
func (jar *Jar) DeleteExpired(partition string, now time.Time) int {
	jar.mutex.Lock()
	defer jar.mutex.Unlock()
	deleted := 0
	for key, cookie := range jar.entries[partition] {
		if cookie.HasExpired(now) {
			delete(jar.entries[partition], key)
			deleted++
		}
	}
	return deleted
}

// Clear removes all cookies in a partition.
func (jar *Jar) Clear(partition string) {
	jar.mutex.Lock()
	defer jar.mutex.Unlock()
	delete(jar.entries, partition)
}

// All returns all cookies in a partition.
func (jar *Jar) All(partition string) []Cookie {
	jar.mutex.RLock()
	defer jar.mutex.RUnlock()
	result := make([]Cookie, 0, len(jar.entries[partition]))
	for _, cookie := range jar.entries[partition] {
		result = append(result, cookie)
	}
	return result
}

// Save writes the jar to a JSON Lines file.
func (jar *Jar) Save(path string) error {
	jar.mutex.RLock()
	defer jar.mutex.RUnlock()
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	writer := bufio.NewWriter(file)
	encoder := json.NewEncoder(writer)
	for partition, cookies := range jar.entries {
		for _, cookie := range cookies {
			record := struct {
				Partition string `json:"partition"`
				Cookie    Cookie `json:"cookie"`
			}{Partition: partition, Cookie: cookie}
			if err := encoder.Encode(record); err != nil {
				return err
			}
		}
	}
	return writer.Flush()
}

// Load reads a jar from a JSON Lines file.
func (jar *Jar) Load(path string) error {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	jar.mutex.Lock()
	defer jar.mutex.Unlock()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var record struct {
			Partition string `json:"partition"`
			Cookie    Cookie `json:"cookie"`
		}
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}
		if jar.entries[record.Partition] == nil {
			jar.entries[record.Partition] = make(map[Key]Cookie)
		}
		jar.entries[record.Partition][record.Cookie.Key()] = record.Cookie
	}
	return scanner.Err()
}

func equalFold(first, second string) bool {
	if len(first) != len(second) {
		return false
	}
	for index := 0; index < len(first); index++ {
		if lowerASCII(first[index]) != lowerASCII(second[index]) {
			return false
		}
	}
	return true
}

func lowerASCII(value byte) byte {
	if value >= 'A' && value <= 'Z' {
		return value + 32
	}
	return value
}

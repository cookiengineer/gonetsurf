package network

import (
	"sync"
	"time"

	httpcache "gonetsurf/caches/http"
)

type cacheKey struct {
	partition string
	method    string
	url       string
	varyKey   string
}

type cacheRecord struct {
	response Response
	entry    httpcache.Entry
}

// Cache is a partition-aware in-memory HTTP response cache owned by the broker.
type Cache struct {
	mutex   sync.RWMutex
	records map[cacheKey]cacheRecord
	policy  httpcache.Policy
}

// NewCache returns a cache using the archive policy.
func NewCache() *Cache {
	return &Cache{records: make(map[cacheKey]cacheRecord), policy: httpcache.Policy{Aggressive: true, Now: time.Now}}
}

// Get returns a cached response for request.
func (cache *Cache) Get(request Request) (Response, bool) {
	if cache == nil {
		return Response{}, false
	}
	method := request.Method
	if method == "" {
		method = "GET"
	}
	partition := request.Partition
	if partition == "" {
		partition = "default"
	}
	cache.mutex.RLock()
	defer cache.mutex.RUnlock()
	requestHeader := httpcache.Header(request.Header)
	for key, record := range cache.records {
		if key.partition != partition || key.method != method || key.url != request.URL {
			continue
		}
		if key.varyKey != httpcache.VaryKey(record.entry.Header.Get("Vary"), requestHeader) {
			continue
		}
		entry := record.entry
		if httpcache.IsFresh(&entry, cache.policy) {
			response := record.response
			response.Header = cloneHeader(response.Header)
			response.Body = append([]byte(nil), response.Body...)
			return response, true
		}
	}
	return Response{}, false
}

// Put stores response when HTTP caching rules allow it.
func (cache *Cache) Put(request Request, response Response) {
	if cache == nil {
		return
	}
	method := request.Method
	if method == "" {
		method = "GET"
	}
	requestHeader := httpcache.Header(request.Header)
	responseHeader := httpcache.Header(response.Header)
	if !httpcache.CanStore(method, response.Status, requestHeader, responseHeader) {
		return
	}
	partition := request.Partition
	if partition == "" {
		partition = "default"
	}
	now := time.Now()
	entry := httpcache.Entry{
		Method:       method,
		URL:          request.URL,
		Status:       response.Status,
		Header:       responseHeader,
		Body:         response.Body,
		StoredAt:     now,
		RequestTime:  now,
		ResponseTime: now,
		Partition:    partition,
		VaryKey:      httpcache.VaryKey(responseHeader.Get("Vary"), requestHeader),
	}
	key := cacheKey{partition: partition, method: method, url: request.URL, varyKey: entry.VaryKey}
	cache.mutex.Lock()
	cache.records[key] = cacheRecord{
		response: Response{
			Status: response.Status,
			Header: cloneHeader(response.Header),
			Body:   append([]byte(nil), response.Body...),
		},
		entry: entry,
	}
	cache.mutex.Unlock()
}

// Delete removes all cached entries for a URL in a partition.
func (cache *Cache) Delete(partition, url string) {
	if cache == nil {
		return
	}
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	for key := range cache.records {
		if key.partition == partition && key.url == url {
			delete(cache.records, key)
		}
	}
}

func cloneHeader(header map[string][]string) map[string][]string {
	if header == nil {
		return nil
	}
	cloned := make(map[string][]string, len(header))
	for name, values := range header {
		cloned[name] = append([]string(nil), values...)
	}
	return cloned
}

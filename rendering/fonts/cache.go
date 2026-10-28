package fonts

import "sync"

type fontResourceKey struct {
	partition  string
	url        string
	format     string
	descriptor string
}

type fontResource struct {
	data []byte
	err  error
}

type fontResourceCache struct {
	mutex   sync.Mutex
	entries map[fontResourceKey]fontResource
}

func newFontResourceCache() *fontResourceCache {
	return &fontResourceCache{entries: make(map[fontResourceKey]fontResource)}
}

func (cache *fontResourceCache) get(key fontResourceKey) ([]byte, error, bool) {
	if cache == nil {
		return nil, nil, false
	}
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	entry, ok := cache.entries[key]
	if !ok {
		return nil, nil, false
	}
	return append([]byte(nil), entry.data...), entry.err, true
}

func (cache *fontResourceCache) put(key fontResourceKey, data []byte, err error) {
	if cache == nil {
		return
	}
	cache.mutex.Lock()
	defer cache.mutex.Unlock()
	cache.entries[key] = fontResource{data: append([]byte(nil), data...), err: err}
}

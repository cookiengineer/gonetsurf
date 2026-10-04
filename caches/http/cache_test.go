package httpcache

import (
	"testing"
	"time"
)

func TestCacheControl(t *testing.T) {
	header := Header{"Cache-Control": {"public, max-age=60, no-transform"}}
	directives := CacheControl(header)
	if directives["max-age"] != "60" {
		t.Errorf("max-age = %q", directives["max-age"])
	}
	if _, ok := directives["public"]; !ok {
		t.Error("expected public directive")
	}
}

func TestCanStore(t *testing.T) {
	response := Header{"Cache-Control": {"max-age=60"}}
	if !CanStore("GET", 200, nil, response) {
		t.Error("expected max-age response to be storable")
	}
	noStore := Header{"Cache-Control": {"no-store"}}
	if CanStore("GET", 200, nil, noStore) {
		t.Error("no-store must not be stored")
	}
	if CanStore("POST", 200, nil, response) {
		t.Error("POST must not be stored")
	}
	varyAll := Header{"Cache-Control": {"max-age=60"}, "Vary": {"*"}}
	if CanStore("GET", 200, nil, varyAll) {
		t.Error("Vary: * must not be stored")
	}
}

func TestFreshnessLifetime(t *testing.T) {
	entry := &Entry{Header: Header{"Cache-Control": {"max-age=120"}}}
	if got := FreshnessLifetime(entry); got != 120*time.Second {
		t.Errorf("FreshnessLifetime = %v", got)
	}
}

func TestCurrentAge(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entry := &Entry{
		Header:       Header{"Date": {base.Format(time.RFC1123)}, "Age": {"10"}},
		RequestTime:  base,
		ResponseTime: base.Add(time.Second),
	}
	age := CurrentAge(entry, base.Add(5*time.Second))
	if age != 15*time.Second {
		t.Errorf("CurrentAge = %v, want 15s", age)
	}
}

func TestIsFreshAndAggressive(t *testing.T) {
	base := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	entry := &Entry{
		Header:       Header{"Cache-Control": {"max-age=10"}, "Date": {base.Format(time.RFC1123)}},
		RequestTime:  base,
		ResponseTime: base,
	}
	policy := Policy{Now: func() time.Time { return base.Add(5 * time.Second) }}
	if !IsFresh(entry, policy) {
		t.Error("expected fresh entry")
	}
	stale := Policy{Now: func() time.Time { return base.Add(time.Hour) }}
	if IsFresh(entry, stale) {
		t.Error("expected stale entry")
	}
	aggressive := Policy{Aggressive: true, Now: stale.Now}
	if !IsFresh(entry, aggressive) {
		t.Error("expected aggressive policy to serve stale entries")
	}
}

func TestVaryKey(t *testing.T) {
	request := Header{"Accept-Encoding": {"gzip"}, "Accept-Language": {"en"}}
	key := VaryKey("Accept-Encoding, Accept-Language", request)
	if key == "" {
		t.Fatal("expected non-empty vary key")
	}
	if key == VaryKey("Accept-Encoding", request) {
		t.Error("different Vary headers must produce different keys")
	}
}

func TestValidatorsMatch(t *testing.T) {
	entry := &Entry{Header: Header{"ETag": {`"abc"`}}}
	if !ValidatorsMatch(entry, Header{"ETag": {`W/"abc"`}}) {
		t.Error("weak comparison of identical ETags should match")
	}
	if ValidatorsMatch(entry, Header{"ETag": {`"def"`}}) {
		t.Error("different ETags must not match")
	}
}

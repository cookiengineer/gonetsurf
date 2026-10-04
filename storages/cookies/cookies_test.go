package cookies

import (
	"testing"
	"time"
)

var testNow = time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

func TestParseSetCookie(t *testing.T) {
	cookie, ok := ParseSetCookie("www.example.com", "/a/b", "session=abc; Path=/a; Secure; HttpOnly; SameSite=Lax", testNow)
	if !ok {
		t.Fatal("expected cookie to parse")
	}
	if cookie.Name != "session" || cookie.Value != "abc" || cookie.Path != "/a" || !cookie.Secure || !cookie.HttpOnly {
		t.Fatalf("cookie = %+v", cookie)
	}
	if cookie.SameSite != SameSiteLax {
		t.Errorf("SameSite = %v", cookie.SameSite)
	}
}

func TestDefaultPath(t *testing.T) {
	cookie, ok := ParseSetCookie("example.com", "/a/b/c", "x=1", testNow)
	if !ok || cookie.Path != "/a/b" {
		t.Fatalf("default path = (%q, %v)", cookie.Path, ok)
	}
}

func TestDomainValidation(t *testing.T) {
	if _, ok := ParseSetCookie("example.com", "/", "x=1; Domain=com", testNow); ok {
		t.Error("public suffix must be rejected")
	}
	cookie, ok := ParseSetCookie("www.example.com", "/", "x=1; Domain=example.com", testNow)
	if !ok || cookie.HostOnly {
		t.Errorf("expected a domain cookie, got %+v", cookie)
	}
	if _, ok := ParseSetCookie("example.com", "/", "x=1; Domain=other.com", testNow); ok {
		t.Error("non-matching domain must be rejected")
	}
}

func TestSecurePrefixes(t *testing.T) {
	if _, ok := ParseSetCookie("example.com", "/", "__Secure-id=1", testNow); ok {
		t.Error("__Secure- without Secure must be rejected")
	}
	if _, ok := ParseSetCookie("example.com", "/", "__Host-id=1; Secure; Path=/x; Domain=example.com", testNow); ok {
		t.Error("__Host- with Domain must be rejected")
	}
}

func TestCookiesFor(t *testing.T) {
	jar := NewJar()
	cookie, _ := ParseSetCookie("www.example.com", "/", "a=1", testNow)
	jar.Set("site", cookie)
	secure, _ := ParseSetCookie("www.example.com", "/", "b=2; Secure", testNow)
	jar.Set("site", secure)
	other, _ := ParseSetCookie("other.com", "/", "c=3", testNow)
	jar.Set("site", other)

	plain := jar.CookiesFor("site", "www.example.com", "/", false, testNow)
	if len(plain) != 1 || plain[0].Name != "a" {
		t.Errorf("plain cookies = %+v", plain)
	}
	secureCookies := jar.CookiesFor("site", "www.example.com", "/", true, testNow)
	if len(secureCookies) != 2 {
		t.Errorf("secure cookies = %+v", secureCookies)
	}
}

func TestTrackingSessionReset(t *testing.T) {
	jar := NewJar()
	manager := NewSessionManager(jar)

	firstParty, _ := ParseSetCookie("shop.example.com", "/", "session=1", testNow)
	jar.Set("shop.example.com", firstParty)
	login, _ := ParseSetCookie("shop.example.com", "/", "login=persist; Max-Age=3600", testNow)
	jar.Set("shop.example.com", login)
	thirdParty, _ := ParseSetCookie("tracker.net", "/", "tid=abc", testNow)
	jar.Set("shop.example.com", thirdParty)
	cloaked, _ := ParseSetCookie("metrics.example.com", "/", "mid=xyz", testNow)
	jar.Set("shop.example.com", cloaked)

	manager.RecordRequest("example.com", "shop.example.com", "")
	manager.RecordRequest("example.com", "tracker.net", "")
	manager.RecordRequest("example.com", "metrics.example.com", "collect.tracker.net")

	deleted := manager.ResetTrackingSession("shop.example.com", "example.com", testNow)
	if deleted < 3 {
		t.Fatalf("deleted = %d, want at least 3", deleted)
	}
	remaining := jar.All("shop.example.com")
	if len(remaining) != 1 || remaining[0].Name != "login" {
		t.Errorf("remaining cookies = %+v", remaining)
	}
}

func TestCloakedTrackerGrouping(t *testing.T) {
	graph := NewDependencyGraph()
	graph.Record("example.com", "a.example.com", "collect.tracker.net")
	graph.Record("example.com", "b.example.com", "collect.tracker.net")
	graph.Record("example.com", "safe.example.com", "")
	hosts := graph.ClearableHosts("example.com")
	found := map[string]bool{}
	for _, host := range hosts {
		found[host] = true
	}
	if !found["a.example.com"] || !found["b.example.com"] {
		t.Errorf("cloaked hosts not grouped: %v", hosts)
	}
	if found["safe.example.com"] {
		t.Errorf("first-party host should not be clearable: %v", hosts)
	}
}

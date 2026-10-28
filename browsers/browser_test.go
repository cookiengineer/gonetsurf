package browsers

import (
	"testing"

	"gonetsurf/urls"
)

func parse(t *testing.T, input string) *urls.URL {
	t.Helper()
	parsed, err := urls.Parse(input)
	if err != nil {
		t.Fatal(err)
	}
	return parsed
}

func TestTabsAndNavigation(t *testing.T) {
	browser := New()
	if browser.ActiveTab() == nil {
		t.Fatal("new browser has no active tab")
	}
	first := browser.ActiveTab()
	if err := browser.Navigate(first, parse(t, "https://example.com/a")); err != nil {
		t.Fatal(err)
	}
	if err := browser.Navigate(first, parse(t, "https://example.com/b")); err != nil {
		t.Fatal(err)
	}
	if len(first.History) != 2 || first.HistoryIndex != 1 {
		t.Fatalf("history = %d/%d", len(first.History), first.HistoryIndex)
	}
	if !browser.Back() || first.URL.String() != "https://example.com/a" {
		t.Errorf("back URL = %v", first.URL)
	}
	if !browser.Forward() || first.URL.String() != "https://example.com/b" {
		t.Errorf("forward URL = %v", first.URL)
	}
	if browser.Forward() {
		t.Error("forward past the end succeeded")
	}
	second := browser.NewTab(parse(t, "https://example.org/"))
	if len(browser.Tabs) != 2 || browser.Active != 1 || second.Title != "example.org" {
		t.Errorf("new tab state = %#v", browser)
	}
	if err := browser.SelectTab(0); err != nil {
		t.Fatal(err)
	}
	if err := browser.CloseTab(0); err != nil {
		t.Fatal(err)
	}
	if len(browser.Tabs) != 1 || browser.ActiveTab() != second {
		t.Errorf("close tab state = %#v", browser)
	}
}

func TestNavigationTruncatesForwardHistory(t *testing.T) {
	browser := New()
	tab := browser.ActiveTab()
	for _, address := range []string{"https://example.com/1", "https://example.com/2", "https://example.com/3"} {
		if err := browser.Navigate(tab, parse(t, address)); err != nil {
			t.Fatal(err)
		}
	}
	browser.Back()
	if err := browser.Navigate(tab, parse(t, "https://example.com/new")); err != nil {
		t.Fatal(err)
	}
	if len(tab.History) != 3 || tab.History[2].String() != "https://example.com/new" {
		t.Errorf("history = %v", tab.History)
	}
}

func TestSessionRoundTrip(t *testing.T) {
	browser := New()
	tab := browser.ActiveTab()
	browser.Navigate(tab, parse(t, "https://example.com/a"))
	browser.Navigate(tab, parse(t, "https://example.com/b"))
	browser.NewTab(parse(t, "https://example.org/"))
	browser.SelectTab(0)
	browser.ActiveTab().ScrollY = 42
	session := browser.Session()
	restored, err := Restore(session)
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Tabs) != len(browser.Tabs) || restored.Active != browser.Active {
		t.Fatalf("restored tabs = %d active %d", len(restored.Tabs), restored.Active)
	}
	if restored.Tabs[0].URL.String() != "https://example.com/b" || restored.Tabs[0].HistoryIndex != 1 || restored.Tabs[0].ScrollY != 42 {
		t.Errorf("restored tab = %#v", restored.Tabs[0])
	}
	if restored.Tabs[1].URL.String() != "https://example.org/" {
		t.Errorf("restored second tab = %#v", restored.Tabs[1])
	}
}

func TestRestoreEmptyAndInvalid(t *testing.T) {
	restored, err := Restore(Session{})
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Tabs) != 1 || restored.ActiveTab() == nil {
		t.Errorf("empty restore = %#v", restored)
	}
	if _, err := Restore(Session{Tabs: []TabSession{{URL: "not a url"}}}); err == nil {
		t.Fatal("expected an invalid URL error")
	}
}

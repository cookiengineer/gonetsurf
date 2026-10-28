package browsers

import "testing"

func TestSameDocumentNavigationAndScrollRestoration(t *testing.T) {
	browser := New()
	tab := browser.ActiveTab()
	base := parse(t, "https://example.com/page")
	if err := browser.Navigate(tab, base); err != nil {
		t.Fatal(err)
	}
	tab.SetScroll(0, 120)
	fragment := parse(t, "https://example.com/page#section")
	if !ClassifyNavigation(tab.URL, fragment) {
		t.Fatal("fragment navigation was not classified as same-document")
	}
	if err := browser.NavigateSameDocument(tab, fragment); err != nil {
		t.Fatal(err)
	}
	if len(tab.History) != 2 || tab.HistoryIndex != 1 || tab.URL.Fragment != "section" {
		t.Fatalf("history = %d/%d fragment %q", len(tab.History), tab.HistoryIndex, tab.URL.Fragment)
	}
	if !browser.Back() {
		t.Fatal("back failed")
	}
	if tab.ScrollY != 120 {
		t.Errorf("restored scroll = %v", tab.ScrollY)
	}
	if !browser.Forward() || tab.URL.Fragment != "section" {
		t.Errorf("forward = %v", tab.URL)
	}
}

func TestOpenerAndTabReorder(t *testing.T) {
	browser := New()
	first := browser.ActiveTab()
	second := browser.NewTabWithOpener(parse(t, "https://example.org/"), first.ID)
	third := browser.NewTab(nil)
	if second.OpenerID != first.ID || first.OpenerID != 0 {
		t.Errorf("opener ids = %d/%d", first.OpenerID, second.OpenerID)
	}
	if err := browser.MoveTab(2, 0); err != nil {
		t.Fatal(err)
	}
	if browser.ActiveTab() != third {
		t.Errorf("active tab changed to %#v", browser.ActiveTab())
	}
	if browser.Tabs[0] != third || browser.Tabs[1] != first || browser.Tabs[2] != second {
		t.Errorf("tab order = %#v", browser.Tabs)
	}
}

func TestOmniboxNavigationAndSuggestions(t *testing.T) {
	address, err := NavigationTarget("example.com/path", DefaultSearchTemplate)
	if err != nil {
		t.Fatal(err)
	}
	if address.String() != "https://example.com/path" {
		t.Errorf("address = %q", address.String())
	}
	search, err := NavigationTarget("go lang", DefaultSearchTemplate)
	if err != nil {
		t.Fatal(err)
	}
	if search.Scheme != "https" || search.Hostname() != "duckduckgo.com" || search.Query != "q=go%20lang" {
		t.Errorf("search = %q", search.String())
	}
	browser := New()
	tab := browser.ActiveTab()
	browser.Navigate(tab, parse(t, "https://example.com/guide"))
	suggestions := browser.Suggestions("exam", 4)
	if len(suggestions) == 0 || suggestions[0].URL.Hostname() != "example.com" {
		t.Errorf("suggestions = %#v", suggestions)
	}
}

func TestFrameSessions(t *testing.T) {
	browser := New()
	tab := browser.ActiveTab()
	frame := tab.NewFrame("content", parse(t, "https://example.com/frame"), 1)
	if err := browser.NavigateFrame(tab, frame, parse(t, "https://example.com/next")); err != nil {
		t.Fatal(err)
	}
	if tab.FrameByName("content") != frame || len(frame.History) != 2 {
		t.Errorf("frame = %#v", frame)
	}
	restored, err := Restore(browser.Session())
	if err != nil {
		t.Fatal(err)
	}
	if len(restored.Tabs[0].Frames) != 1 || restored.Tabs[0].Frames[0].URL.String() != "https://example.com/next" {
		t.Errorf("restored frames = %#v", restored.Tabs[0].Frames)
	}
}

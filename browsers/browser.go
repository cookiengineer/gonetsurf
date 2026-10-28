// Package browsers implements the tab, navigation, history, and session model
// used by the browser shell.
package browsers

import (
	"fmt"

	"gonetsurf/urls"
)

// Tab is one browsing context.
type Tab struct {
	ID           int
	URL          *urls.URL
	Title        string
	Loading      bool
	History      []*urls.URL
	HistoryIndex int
	ScrollX      float64
	ScrollY      float64
}

// Browser is a tab collection with one active tab.
type Browser struct {
	Tabs   []*Tab
	Active int
	nextID int
}

// Session is the serializable form of a browser.
type Session struct {
	Active int          `json:"active"`
	Tabs   []TabSession `json:"tabs"`
}

// TabSession is the serializable form of a tab.
type TabSession struct {
	URL          string   `json:"url,omitempty"`
	Title        string   `json:"title,omitempty"`
	History      []string `json:"history,omitempty"`
	HistoryIndex int      `json:"historyIndex,omitempty"`
	ScrollX      float64  `json:"scrollX,omitempty"`
	ScrollY      float64  `json:"scrollY,omitempty"`
}

// New returns a browser with one blank tab.
func New() *Browser {
	browser := &Browser{nextID: 1}
	browser.NewTab(nil)
	return browser
}

// NewTab appends a tab and selects it.
func (browser *Browser) NewTab(address *urls.URL) *Tab {
	tab := &Tab{ID: browser.nextID, URL: address, ScrollX: 0, ScrollY: 0}
	browser.nextID++
	if address != nil {
		tab.Title = address.Hostname()
		tab.History = []*urls.URL{address}
		tab.HistoryIndex = 0
	}
	browser.Tabs = append(browser.Tabs, tab)
	browser.Active = len(browser.Tabs) - 1
	return tab
}

// ActiveTab returns the selected tab, or nil when no tabs exist.
func (browser *Browser) ActiveTab() *Tab {
	if browser == nil || browser.Active < 0 || browser.Active >= len(browser.Tabs) {
		return nil
	}
	return browser.Tabs[browser.Active]
}

// SelectTab makes index active.
func (browser *Browser) SelectTab(index int) error {
	if index < 0 || index >= len(browser.Tabs) {
		return fmt.Errorf("browsers: tab index %d out of range", index)
	}
	browser.Active = index
	return nil
}

// CloseTab removes the tab at index and selects a neighbor.
func (browser *Browser) CloseTab(index int) error {
	if index < 0 || index >= len(browser.Tabs) {
		return fmt.Errorf("browsers: tab index %d out of range", index)
	}
	browser.Tabs = append(browser.Tabs[:index], browser.Tabs[index+1:]...)
	if len(browser.Tabs) == 0 {
		browser.NewTab(nil)
		return nil
	}
	if browser.Active > index || browser.Active >= len(browser.Tabs) {
		browser.Active--
	}
	if browser.Active < 0 {
		browser.Active = 0
	}
	return nil
}

// Navigate adds a history entry and makes address current.
func (browser *Browser) Navigate(tab *Tab, address *urls.URL) error {
	if tab == nil {
		return fmt.Errorf("browsers: nil tab")
	}
	if address == nil {
		return fmt.Errorf("browsers: nil URL")
	}
	if tab.HistoryIndex >= 0 && tab.HistoryIndex < len(tab.History) && sameURL(tab.History[tab.HistoryIndex], address) {
		tab.URL = address
		return nil
	}
	if tab.HistoryIndex < len(tab.History)-1 {
		tab.History = tab.History[:tab.HistoryIndex+1]
	}
	tab.History = append(tab.History, address)
	tab.HistoryIndex = len(tab.History) - 1
	tab.URL = address
	tab.Title = address.Hostname()
	return nil
}

// NavigateActive navigates the active tab.
func (browser *Browser) NavigateActive(address *urls.URL) error {
	return browser.Navigate(browser.ActiveTab(), address)
}

// Back moves the active tab back in history.
func (browser *Browser) Back() bool {
	tab := browser.ActiveTab()
	if tab == nil || tab.HistoryIndex <= 0 {
		return false
	}
	tab.HistoryIndex--
	tab.URL = tab.History[tab.HistoryIndex]
	return true
}

// Forward moves the active tab forward in history.
func (browser *Browser) Forward() bool {
	tab := browser.ActiveTab()
	if tab == nil || tab.HistoryIndex >= len(tab.History)-1 {
		return false
	}
	tab.HistoryIndex++
	tab.URL = tab.History[tab.HistoryIndex]
	return true
}

// Reload re-selects the current history entry.
func (browser *Browser) Reload() bool {
	tab := browser.ActiveTab()
	if tab == nil || tab.HistoryIndex < 0 || tab.HistoryIndex >= len(tab.History) {
		return false
	}
	tab.URL = tab.History[tab.HistoryIndex]
	return true
}

// Session serializes the browser state.
func (browser *Browser) Session() Session {
	session := Session{Active: browser.Active}
	for _, tab := range browser.Tabs {
		tabSession := TabSession{
			Title:        tab.Title,
			HistoryIndex: tab.HistoryIndex,
			ScrollX:      tab.ScrollX,
			ScrollY:      tab.ScrollY,
		}
		if tab.URL != nil {
			tabSession.URL = tab.URL.String()
		}
		for _, entry := range tab.History {
			if entry != nil {
				tabSession.History = append(tabSession.History, entry.String())
			}
		}
		session.Tabs = append(session.Tabs, tabSession)
	}
	return session
}

// Restore rebuilds a browser from session data.
func Restore(session Session) (*Browser, error) {
	browser := &Browser{nextID: 1}
	for _, tabSession := range session.Tabs {
		address, err := parseOptionalURL(tabSession.URL)
		if err != nil {
			return nil, err
		}
		tab := &Tab{
			ID:           browser.nextID,
			Title:        tabSession.Title,
			HistoryIndex: tabSession.HistoryIndex,
			ScrollX:      tabSession.ScrollX,
			ScrollY:      tabSession.ScrollY,
		}
		browser.nextID++
		for _, entry := range tabSession.History {
			parsed, err := urls.Parse(entry)
			if err != nil {
				return nil, err
			}
			tab.History = append(tab.History, parsed)
		}
		if address == nil && len(tab.History) > 0 {
			address = tab.History[len(tab.History)-1]
		}
		tab.URL = address
		if tab.HistoryIndex < 0 || tab.HistoryIndex >= len(tab.History) {
			tab.HistoryIndex = len(tab.History) - 1
		}
		browser.Tabs = append(browser.Tabs, tab)
	}
	if len(browser.Tabs) == 0 {
		browser.NewTab(nil)
	}
	if session.Active < 0 || session.Active >= len(browser.Tabs) {
		browser.Active = 0
	} else {
		browser.Active = session.Active
	}
	return browser, nil
}

func parseOptionalURL(value string) (*urls.URL, error) {
	if value == "" {
		return nil, nil
	}
	return urls.Parse(value)
}

func sameURL(first, second *urls.URL) bool {
	if first == nil || second == nil {
		return first == second
	}
	return first.String() == second.String()
}

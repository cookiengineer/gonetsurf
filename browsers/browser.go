// Package browsers implements the tab, navigation, history, and session model
// used by the browser shell.
package browsers

import (
	"fmt"
	"sort"
	"strings"

	"gonetsurf/urls"
)

// DefaultSearchTemplate is used when omnibox input is not a URL.
const DefaultSearchTemplate = "https://duckduckgo.com/?q=%s"

// Context holds the navigation state shared by top-level tabs and child
// browsing contexts.
type Context struct {
	URL           *urls.URL
	Title         string
	History       []*urls.URL
	HistoryIndex  int
	HistoryScroll []ScrollPosition
}

// ScrollPosition records a viewport scroll offset.
type ScrollPosition struct {
	X float64 `json:"x,omitempty"`
	Y float64 `json:"y,omitempty"`
}

// Tab is one top-level browsing context.
type Tab struct {
	ID int
	Context
	OpenerID int
	Loading  bool
	ScrollX  float64
	ScrollY  float64
	Frames   []*Frame
}

// Frame is a nested browsing context owned by a tab.
type Frame struct {
	Context
	Name    string
	Sandbox uint32
	ScrollX float64
	ScrollY float64
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
	URL           string           `json:"url,omitempty"`
	Title         string           `json:"title,omitempty"`
	History       []string         `json:"history,omitempty"`
	HistoryIndex  int              `json:"historyIndex,omitempty"`
	ScrollX       float64          `json:"scrollX,omitempty"`
	ScrollY       float64          `json:"scrollY,omitempty"`
	OpenerID      int              `json:"openerID,omitempty"`
	HistoryScroll []ScrollPosition `json:"historyScroll,omitempty"`
	Frames        []FrameSession   `json:"frames,omitempty"`
}

// FrameSession is the serializable form of a nested browsing context.
type FrameSession struct {
	Name         string   `json:"name,omitempty"`
	URL          string   `json:"url,omitempty"`
	Title        string   `json:"title,omitempty"`
	History      []string `json:"history,omitempty"`
	HistoryIndex int      `json:"historyIndex,omitempty"`
	Sandbox      uint32   `json:"sandbox,omitempty"`
}

// Suggestion is an omnibox completion candidate.
type Suggestion struct {
	URL   *urls.URL
	Title string
	Score int
}

// New returns a browser with one blank tab.
func New() *Browser {
	browser := &Browser{nextID: 1}
	browser.NewTab(nil)
	return browser
}

// NewTab appends a tab and selects it.
func (browser *Browser) NewTab(address *urls.URL) *Tab {
	return browser.NewTabWithOpener(address, 0)
}

// NewTabWithOpener appends a tab opened by openerID and selects it.
func (browser *Browser) NewTabWithOpener(address *urls.URL, openerID int) *Tab {
	tab := &Tab{ID: browser.nextID, OpenerID: openerID}
	browser.nextID++
	if address != nil {
		tab.URL = address
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

// MoveTab moves the tab at from to index to and keeps the active tab selected.
func (browser *Browser) MoveTab(from, to int) error {
	if from < 0 || from >= len(browser.Tabs) {
		return fmt.Errorf("browsers: tab index %d out of range", from)
	}
	if to < 0 {
		to = 0
	}
	if to >= len(browser.Tabs) {
		to = len(browser.Tabs) - 1
	}
	if from == to {
		return nil
	}
	active := browser.ActiveTab()
	moving := browser.Tabs[from]
	browser.Tabs = append(browser.Tabs[:from], browser.Tabs[from+1:]...)
	browser.Tabs = append(browser.Tabs, nil)
	copy(browser.Tabs[to+1:], browser.Tabs[to:])
	browser.Tabs[to] = moving
	for index, tab := range browser.Tabs {
		if tab == active {
			browser.Active = index
			return nil
		}
	}
	browser.Active = 0
	return nil
}

// ClassifyNavigation reports whether target continues the current document.
func ClassifyNavigation(current, target *urls.URL) bool {
	if current == nil || target == nil {
		return false
	}
	return SameDocumentURL(current, target)
}

// SameDocumentURL reports whether two URLs differ only by fragment.
func SameDocumentURL(first, second *urls.URL) bool {
	if first == nil || second == nil {
		return first == second
	}
	left := *first
	right := *second
	left.Fragment = ""
	left.HasFragment = false
	right.Fragment = ""
	right.HasFragment = false
	return left.String() == right.String()
}

// Navigate adds a history entry and makes address current.
func (browser *Browser) Navigate(tab *Tab, address *urls.URL) error {
	if tab == nil {
		return fmt.Errorf("browsers: nil tab")
	}
	if address == nil {
		return fmt.Errorf("browsers: nil URL")
	}
	navigateContext(&tab.Context, &tab.ScrollX, &tab.ScrollY, address)
	return nil
}

// NavigateSameDocument records a fragment-style navigation without reloading.
func (browser *Browser) NavigateSameDocument(tab *Tab, address *urls.URL) error {
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
	navigateContext(&tab.Context, &tab.ScrollX, &tab.ScrollY, address)
	return nil
}

// NavigateFrame navigates a nested browsing context.
func (browser *Browser) NavigateFrame(tab *Tab, frame *Frame, address *urls.URL) error {
	if tab == nil || frame == nil {
		return fmt.Errorf("browsers: nil browsing context")
	}
	if address == nil {
		return fmt.Errorf("browsers: nil URL")
	}
	navigateContext(&frame.Context, &frame.ScrollX, &frame.ScrollY, address)
	return nil
}

// NewFrame appends a nested browsing context to tab.
func (tab *Tab) NewFrame(name string, address *urls.URL, sandbox uint32) *Frame {
	frame := &Frame{Name: name, Sandbox: sandbox}
	if address != nil {
		frame.URL = address
		frame.Title = address.Hostname()
		frame.History = []*urls.URL{address}
		frame.HistoryIndex = 0
	}
	tab.Frames = append(tab.Frames, frame)
	return frame
}

// FrameByName returns the named frame, or nil.
func (tab *Tab) FrameByName(name string) *Frame {
	if tab == nil {
		return nil
	}
	for _, frame := range tab.Frames {
		if frame.Name == name {
			return frame
		}
	}
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
	tab.recordScroll()
	tab.HistoryIndex--
	tab.URL = tab.History[tab.HistoryIndex]
	tab.restoreScroll()
	return true
}

// Forward moves the active tab forward in history.
func (browser *Browser) Forward() bool {
	tab := browser.ActiveTab()
	if tab == nil || tab.HistoryIndex >= len(tab.History)-1 {
		return false
	}
	tab.recordScroll()
	tab.HistoryIndex++
	tab.URL = tab.History[tab.HistoryIndex]
	tab.restoreScroll()
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

// Suggestions returns omnibox completions from tab history.
func (browser *Browser) Suggestions(input string, limit int) []Suggestion {
	query := strings.ToLower(strings.TrimSpace(input))
	if query == "" || browser == nil {
		return nil
	}
	seen := make(map[string]bool)
	var suggestions []Suggestion
	appendEntry := func(address *urls.URL, title string) {
		if address == nil {
			return
		}
		serialized := address.String()
		if seen[serialized] {
			return
		}
		score := suggestionScore(query, serialized, title)
		if score <= 0 {
			return
		}
		seen[serialized] = true
		suggestions = append(suggestions, Suggestion{URL: address, Title: title, Score: score})
	}
	for _, tab := range browser.Tabs {
		for index := len(tab.History) - 1; index >= 0; index-- {
			title := tab.Title
			if index != tab.HistoryIndex {
				title = ""
			}
			appendEntry(tab.History[index], title)
		}
	}
	sort.SliceStable(suggestions, func(first, second int) bool {
		return suggestions[first].Score > suggestions[second].Score
	})
	if limit > 0 && len(suggestions) > limit {
		suggestions = suggestions[:limit]
	}
	return suggestions
}

// NavigationTarget resolves omnibox input to a URL, using searchTemplate when
// the input is not URL-like.
func NavigationTarget(input, searchTemplate string) (*urls.URL, error) {
	value := strings.TrimSpace(input)
	if value == "" {
		return nil, urls.ErrInvalidURL
	}
	lower := strings.ToLower(value)
	if strings.Contains(lower, "://") || strings.HasPrefix(lower, "about:") ||
		strings.HasPrefix(lower, "data:") || strings.HasPrefix(lower, "file:") {
		return urls.Parse(value)
	}
	if !strings.ContainsAny(value, " \t\n") && looksLikeHost(value) {
		return urls.Parse("https://" + value)
	}
	if searchTemplate == "" {
		searchTemplate = DefaultSearchTemplate
	}
	return urls.Parse(strings.Replace(searchTemplate, "%s", escapeQuery(value), 1))
}

func looksLikeHost(value string) bool {
	if strings.EqualFold(value, "localhost") || strings.HasPrefix(strings.ToLower(value), "localhost:") {
		return true
	}
	if strings.HasPrefix(value, "[") {
		return true
	}
	return strings.Contains(value, ".")
}

func escapeQuery(value string) string {
	const unreserved = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-._~"
	var builder strings.Builder
	for _, character := range []byte(value) {
		if strings.IndexByte(unreserved, character) >= 0 {
			builder.WriteByte(character)
			continue
		}
		builder.WriteString("%")
		builder.WriteString(strings.ToUpper(hexPair(character)))
	}
	return builder.String()
}

func hexPair(value byte) string {
	const digits = "0123456789abcdef"
	return string([]byte{digits[value>>4], digits[value&0x0f]})
}

func suggestionScore(query, address, title string) int {
	lowerAddress := strings.ToLower(address)
	lowerTitle := strings.ToLower(title)
	switch {
	case strings.HasPrefix(lowerAddress, query):
		return 4
	case strings.HasPrefix(lowerTitle, query):
		return 3
	case strings.Contains(lowerAddress, query):
		return 2
	case strings.Contains(lowerTitle, query):
		return 1
	default:
		return 0
	}
}

func navigateContext(context *Context, scrollX, scrollY *float64, address *urls.URL) {
	if context.HistoryIndex < len(context.History) {
		recordContextScroll(context, *scrollX, *scrollY)
	}
	if context.HistoryIndex < len(context.History)-1 {
		context.History = context.History[:context.HistoryIndex+1]
		if context.HistoryIndex+1 < len(context.HistoryScroll) {
			context.HistoryScroll = context.HistoryScroll[:context.HistoryIndex+1]
		}
	}
	context.History = append(context.History, address)
	context.HistoryIndex = len(context.History) - 1
	context.URL = address
	context.Title = address.Hostname()
	*scrollX = 0
	*scrollY = 0
}

func recordContextScroll(context *Context, x, y float64) {
	if context.HistoryIndex < 0 || context.HistoryIndex >= len(context.History) {
		return
	}
	for len(context.HistoryScroll) <= context.HistoryIndex {
		context.HistoryScroll = append(context.HistoryScroll, ScrollPosition{})
	}
	context.HistoryScroll[context.HistoryIndex] = ScrollPosition{X: x, Y: y}
}

// recordScroll stores the tab's current scroll position for the active entry.
func (tab *Tab) recordScroll() {
	if tab == nil {
		return
	}
	recordContextScroll(&tab.Context, tab.ScrollX, tab.ScrollY)
}

// SetScroll updates the current viewport scroll position.
func (tab *Tab) SetScroll(x, y float64) {
	if tab == nil {
		return
	}
	tab.ScrollX = x
	tab.ScrollY = y
	tab.recordScroll()
}

func (tab *Tab) restoreScroll() {
	if tab == nil || tab.HistoryIndex < 0 || tab.HistoryIndex >= len(tab.HistoryScroll) {
		tab.ScrollX = 0
		tab.ScrollY = 0
		return
	}
	position := tab.HistoryScroll[tab.HistoryIndex]
	tab.ScrollX = position.X
	tab.ScrollY = position.Y
}

// Session serializes the browser state.
func (browser *Browser) Session() Session {
	session := Session{Active: browser.Active}
	for _, tab := range browser.Tabs {
		tabSession := TabSession{
			Title:         tab.Title,
			HistoryIndex:  tab.HistoryIndex,
			ScrollX:       tab.ScrollX,
			ScrollY:       tab.ScrollY,
			OpenerID:      tab.OpenerID,
			HistoryScroll: append([]ScrollPosition(nil), tab.HistoryScroll...),
		}
		if tab.URL != nil {
			tabSession.URL = tab.URL.String()
		}
		for _, entry := range tab.History {
			if entry != nil {
				tabSession.History = append(tabSession.History, entry.String())
			}
		}
		for _, frame := range tab.Frames {
			frameSession := FrameSession{
				Name:         frame.Name,
				Title:        frame.Title,
				HistoryIndex: frame.HistoryIndex,
				Sandbox:      frame.Sandbox,
			}
			if frame.URL != nil {
				frameSession.URL = frame.URL.String()
			}
			for _, entry := range frame.History {
				if entry != nil {
					frameSession.History = append(frameSession.History, entry.String())
				}
			}
			tabSession.Frames = append(tabSession.Frames, frameSession)
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
			ID:       browser.nextID,
			OpenerID: tabSession.OpenerID,
			ScrollX:  tabSession.ScrollX,
			ScrollY:  tabSession.ScrollY,
			Context:  Context{Title: tabSession.Title, HistoryIndex: tabSession.HistoryIndex, HistoryScroll: append([]ScrollPosition(nil), tabSession.HistoryScroll...)},
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
		for _, frameSession := range tabSession.Frames {
			frameAddress, err := parseOptionalURL(frameSession.URL)
			if err != nil {
				return nil, err
			}
			frame := &Frame{
				Name:    frameSession.Name,
				Sandbox: frameSession.Sandbox,
				Context: Context{Title: frameSession.Title, HistoryIndex: frameSession.HistoryIndex},
			}
			for _, entry := range frameSession.History {
				parsed, err := urls.Parse(entry)
				if err != nil {
					return nil, err
				}
				frame.History = append(frame.History, parsed)
			}
			if frameAddress == nil && len(frame.History) > 0 {
				frameAddress = frame.History[len(frame.History)-1]
			}
			frame.URL = frameAddress
			if frame.HistoryIndex < 0 || frame.HistoryIndex >= len(frame.History) {
				frame.HistoryIndex = len(frame.History) - 1
			}
			tab.Frames = append(tab.Frames, frame)
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

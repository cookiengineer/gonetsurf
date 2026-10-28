// Package shells implements the browser shell: chrome, input routing, tab
// navigation, and the event-driven frame loop.
package shells

import (
	"context"
	"encoding/json"
	"fmt"
	htmlstd "html"
	"image"
	"strings"
	"time"

	"gonetsurf/brokers/network"
	"gonetsurf/brokers/storage"
	"gonetsurf/browsers"
	"gonetsurf/dom"
	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/html"
	"gonetsurf/rendering"
	"gonetsurf/styles"
	"gonetsurf/urls"
	"gonetsurf/windowing"
)

const (
	topBarHeight    = 34
	statusBarHeight = 20
	tabWidth        = 150
)

type pageState struct {
	document *dom.Document
	engine   *styles.Engine
	url      *urls.URL
}

// BrokerClients groups the capability-scoped broker clients used by a shell.
type BrokerClients struct {
	Network *network.Client
	Storage *storage.Client
}

// Shell owns the browser model and the current page state.
type Shell struct {
	Browser *browsers.Browser
	Network *network.Client
	Storage *storage.Client
	Width   int
	Height  int

	OmniboxFocused bool
	OmniboxText    string
	Caret          int
	Status         string
	Error          string

	page  *pageState
	dirty bool
}

// New returns a shell with one blank tab and optional broker clients.
func New(width, height int, initial *urls.URL, brokers ...BrokerClients) *Shell {
	shell := &Shell{Browser: browsers.New(), Width: width, Height: height, dirty: true}
	if len(brokers) > 0 {
		shell.Network = brokers[0].Network
		shell.Storage = brokers[0].Storage
	}
	shell.OmniboxText = ""
	if initial != nil {
		shell.navigate(initial.String())
	}
	return shell
}

// NavigateInitial navigates the active tab after broker clients are attached.
func (shell *Shell) NavigateInitial(initial *urls.URL) {
	if initial != nil {
		shell.navigate(initial.String())
	}
}

// Dirty reports whether the shell needs a repaint.
func (shell *Shell) Dirty() bool {
	return shell.dirty
}

// MarkDirty requests a repaint.
func (shell *Shell) MarkDirty() {
	shell.dirty = true
}

// Resize updates the viewport.
func (shell *Shell) Resize(width, height int) {
	if width <= 0 || height <= 0 {
		return
	}
	shell.Width = width
	shell.Height = height
	shell.dirty = true
}

// Render builds a frame with content, chrome, and the status bar.
func (shell *Shell) Render() *rendering.Surface {
	result := rendering.NewSurface(shell.Width, shell.Height)
	result.Clear(rendering.NewColor(255, 255, 255))
	contentHeight := shell.Height - topBarHeight - statusBarHeight
	if contentHeight < 0 {
		contentHeight = 0
	}
	if shell.page != nil && contentHeight > 0 {
		options := shell.resourceOptions(shell.page.url)
		content := rendering.RenderDocumentWithOptions(shell.page.document, shell.page.engine, shell.Width, contentHeight, options)
		result.Blit(0, topBarHeight, content, rendering.Copy)
	}
	topBar := renderChrome(shell.chromeHTML(), shell.Width, topBarHeight)
	result.Blit(0, 0, topBar, rendering.Copy)
	statusBar := renderChrome(shell.statusHTML(), shell.Width, statusBarHeight)
	result.Blit(0, shell.Height-statusBarHeight, statusBar, rendering.Copy)
	return result
}

func (shell *Shell) chromeHTML() string {
	var builder strings.Builder
	builder.WriteString(`<!doctype html><html><head><style>
html, body { margin:0; padding:0; font-family:sans-serif; font-size:12px; color:#222222 }
.tab { position:absolute; top:4px; height:26px; box-sizing:border-box; padding:6px 8px; border:1px solid #b8b8bc; background:#e8e8ec; white-space:nowrap; overflow:hidden }
.tab.active { background:#ffffff; border-bottom-color:#ffffff }
.omnibox { position:absolute; top:5px; height:24px; box-sizing:border-box; padding:4px 8px; border:1px solid #888888; background:#ffffff; white-space:nowrap; overflow:hidden }
.privacy { position:absolute; top:8px; right:6px; color:#226622; font-size:11px }
</style></head><body>`)
	for index, tab := range shell.Browser.Tabs {
		title := tab.Title
		if title == "" {
			title = "New Tab"
		}
		if len(title) > 18 {
			title = title[:18]
		}
		class := "tab"
		if index == shell.Browser.Active {
			class += " active"
		}
		builder.WriteString(fmt.Sprintf(`<div class="%s" style="left:%dpx;width:%dpx">%s</div>`,
			class, index*tabWidth, tabWidth, htmlstd.EscapeString(title)))
	}
	omniboxLeft := len(shell.Browser.Tabs) * tabWidth
	omniboxWidth := shell.Width - omniboxLeft - 96
	if omniboxWidth < 80 {
		omniboxWidth = 80
	}
	text := shell.OmniboxText
	if shell.OmniboxFocused {
		caret := shell.Caret
		if caret < 0 {
			caret = 0
		}
		if caret > len(text) {
			caret = len(text)
		}
		text = text[:caret] + "|" + text[caret:]
	}
	builder.WriteString(fmt.Sprintf(`<div class="omnibox" style="left:%dpx;width:%dpx">%s</div>`,
		omniboxLeft, omniboxWidth, htmlstd.EscapeString(text)))
	builder.WriteString(`<div class="privacy">privacy: on</div>`)
	builder.WriteString(`</body></html>`)
	return builder.String()
}

func (shell *Shell) statusHTML() string {
	status := shell.Status
	if shell.Error != "" {
		status = shell.Error
	}
	if status == "" {
		status = "Ready"
	}
	if len(status) > 140 {
		status = status[:140]
	}
	return `<!doctype html><html><head><style>
html, body { margin:0; padding:0; font-family:sans-serif; font-size:11px; color:#333333 }
.status { position:absolute; left:6px; top:3px; white-space:nowrap }
.mode { position:absolute; right:6px; top:3px; color:#555555 }
</style></head><body><div class="status">` + htmlstd.EscapeString(status) + `</div><div class="mode">archive cache ready</div></body></html>`
}

func renderChrome(source string, width, height int) *rendering.Surface {
	if width <= 0 || height <= 0 {
		return rendering.NewSurface(width, height)
	}
	document, err := html.ParseString(source, html.ParseOptions{})
	if err != nil {
		return rendering.NewSurface(width, height)
	}
	engine := styleEngine(document)
	return rendering.RenderDocumentWithOptions(document, engine, width, height, rendering.Options{})
}

func styleEngine(document *dom.Document) *styles.Engine {
	engine := styles.NewEngine(document)
	document.Node.Descendants(func(node *dom.Node) bool {
		if node.Type == dom.ElementNode && node.Data == "style" {
			engine.AddStylesheet(cssom.Parse(node.TextContent()))
		}
		return true
	})
	return engine
}

// Run processes events and paints frames until the window closes or ctx ends.
func (shell *Shell) Run(ctx context.Context, window windowing.Window) error {
	input, _ := window.(windowing.InputWindow)
	ticker := time.NewTicker(16 * time.Millisecond)
	defer ticker.Stop()
	if shell.dirty {
		shell.present(window)
	}
	for {
		var events <-chan windowing.Event
		if input != nil {
			events = input.Events()
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if shell.dirty {
				shell.present(window)
			}
		case event, ok := <-events:
			if !ok {
				return nil
			}
			if event.Type == windowing.EventClose {
				return nil
			}
			if event.Type == windowing.EventResize {
				shell.Resize(event.Width, event.Height)
				continue
			}
			shell.HandleEvent(ctx, input, event)
		}
	}
}

func (shell *Shell) present(window windowing.Window) {
	surface := shell.Render()
	window.Surface().Blit(0, 0, surface, rendering.Copy)
	if err := window.Present(image.Rect(0, 0, shell.Width, shell.Height)); err != nil {
		shell.Error = err.Error()
	}
	shell.dirty = false
}

// HandleEvent routes one input event.
func (shell *Shell) HandleEvent(ctx context.Context, input windowing.InputWindow, event windowing.Event) {
	switch event.Type {
	case windowing.EventKeyPress:
		shell.handleKey(ctx, input, event)
	case windowing.EventMousePress:
		shell.handleClick(event)
	}
	shell.dirty = true
}

func (shell *Shell) handleKey(ctx context.Context, input windowing.InputWindow, event windowing.Event) {
	control := event.Modifiers.Control
	switch {
	case control && strings.EqualFold(event.Key, "l"):
		shell.focusOmnibox()
	case control && strings.EqualFold(event.Key, "t"):
		shell.Browser.NewTab(nil)
		shell.page = nil
		shell.focusOmnibox()
	case control && strings.EqualFold(event.Key, "w"):
		_ = shell.Browser.CloseTab(shell.Browser.Active)
		shell.page = nil
		shell.syncActiveTab()
	case control && strings.EqualFold(event.Key, "Tab"):
		shell.cycleTab(1)
	case control && strings.EqualFold(event.Key, "r"), strings.EqualFold(event.Key, "F5"):
		shell.reload()
	case control && strings.EqualFold(event.Key, "c"):
		if input != nil && shell.Browser.ActiveTab() != nil && shell.Browser.ActiveTab().URL != nil {
			input.SetClipboard(shell.Browser.ActiveTab().URL.String())
			shell.Status = "Copied URL"
		}
	case control && strings.EqualFold(event.Key, "v"):
		if input != nil {
			if text, err := input.GetClipboard(ctx); err == nil {
				shell.OmniboxText += text
				shell.Caret = len(shell.OmniboxText)
				shell.OmniboxFocused = true
			}
		}
	case strings.EqualFold(event.Key, "Escape"):
		shell.OmniboxFocused = false
	case strings.EqualFold(event.Key, "Enter"):
		if shell.OmniboxFocused {
			shell.navigate(shell.OmniboxText)
			shell.OmniboxFocused = false
		}
	case strings.EqualFold(event.Key, "BackSpace"):
		if shell.OmniboxFocused {
			shell.deleteBackward()
		} else {
			shell.Browser.Back()
			shell.syncActiveTab()
		}
	case strings.EqualFold(event.Key, "Delete"):
		if shell.OmniboxFocused {
			shell.deleteForward()
		}
	case strings.EqualFold(event.Key, "Left"):
		if shell.OmniboxFocused {
			if shell.Caret > 0 {
				shell.Caret--
			}
		} else if event.Modifiers.Alt {
			shell.Browser.Back()
			shell.syncActiveTab()
		}
	case strings.EqualFold(event.Key, "Right"):
		if shell.OmniboxFocused {
			if shell.Caret < len(shell.OmniboxText) {
				shell.Caret++
			}
		} else if event.Modifiers.Alt {
			shell.Browser.Forward()
			shell.syncActiveTab()
		}
	case strings.EqualFold(event.Key, "Home"):
		if shell.OmniboxFocused {
			shell.Caret = 0
		}
	case strings.EqualFold(event.Key, "End"):
		if shell.OmniboxFocused {
			shell.Caret = len(shell.OmniboxText)
		}
	default:
		if shell.OmniboxFocused && event.Text != "" {
			shell.insertText(event.Text)
		}
	}
}

func (shell *Shell) handleClick(event windowing.Event) {
	if event.Y < topBarHeight {
		tabIndex := event.X / tabWidth
		if tabIndex < len(shell.Browser.Tabs) {
			_ = shell.Browser.SelectTab(tabIndex)
			shell.syncActiveTab()
			shell.OmniboxFocused = false
			return
		}
		shell.focusOmnibox()
		return
	}
	shell.OmniboxFocused = false
}

func (shell *Shell) focusOmnibox() {
	shell.OmniboxFocused = true
	if tab := shell.Browser.ActiveTab(); tab != nil && tab.URL != nil {
		shell.OmniboxText = tab.URL.String()
	}
	shell.Caret = len(shell.OmniboxText)
}

func (shell *Shell) syncActiveTab() {
	tab := shell.Browser.ActiveTab()
	if tab == nil {
		shell.OmniboxText = ""
		shell.Caret = 0
		return
	}
	if tab.URL != nil {
		shell.OmniboxText = tab.URL.String()
	} else {
		shell.OmniboxText = ""
	}
	shell.Caret = len(shell.OmniboxText)
	shell.loadActiveTab(false)
}

func (shell *Shell) cycleTab(direction int) {
	if len(shell.Browser.Tabs) == 0 {
		return
	}
	index := (shell.Browser.Active + direction) % len(shell.Browser.Tabs)
	if index < 0 {
		index += len(shell.Browser.Tabs)
	}
	_ = shell.Browser.SelectTab(index)
	shell.syncActiveTab()
}

func (shell *Shell) insertText(text string) {
	caret := shell.Caret
	if caret < 0 {
		caret = 0
	}
	if caret > len(shell.OmniboxText) {
		caret = len(shell.OmniboxText)
	}
	shell.OmniboxText = shell.OmniboxText[:caret] + text + shell.OmniboxText[caret:]
	shell.Caret += len(text)
}

func (shell *Shell) deleteBackward() {
	if shell.Caret <= 0 || shell.Caret > len(shell.OmniboxText) {
		return
	}
	shell.OmniboxText = shell.OmniboxText[:shell.Caret-1] + shell.OmniboxText[shell.Caret:]
	shell.Caret--
}

func (shell *Shell) deleteForward() {
	if shell.Caret < 0 || shell.Caret >= len(shell.OmniboxText) {
		return
	}
	shell.OmniboxText = shell.OmniboxText[:shell.Caret] + shell.OmniboxText[shell.Caret+1:]
}

func (shell *Shell) navigate(input string) {
	address := strings.TrimSpace(input)
	if address == "" {
		return
	}
	parsed, err := urls.Parse(address)
	if err != nil {
		if candidate, candidateErr := urls.Parse("https://" + address); candidateErr == nil {
			parsed = candidate
		} else {
			shell.Error = err.Error()
			return
		}
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" && parsed.Scheme != "about" && parsed.Scheme != "data" {
		shell.Error = "unsupported scheme " + parsed.Scheme
		return
	}
	tab := shell.Browser.ActiveTab()
	if tab == nil {
		tab = shell.Browser.NewTab(parsed)
	}
	_ = shell.Browser.Navigate(tab, parsed)
	shell.OmniboxText = parsed.String()
	shell.Caret = len(shell.OmniboxText)
	shell.Status = "Loading " + parsed.String()
	shell.Error = ""
	shell.loadActiveTab(false)
	if tab.URL != nil {
		shell.OmniboxText = tab.URL.String()
		shell.Caret = len(shell.OmniboxText)
	}
}

func (shell *Shell) reload() {
	tab := shell.Browser.ActiveTab()
	if tab == nil || tab.URL == nil {
		return
	}
	shell.Browser.Reload()
	shell.loadActiveTab(true)
}

func (shell *Shell) loadActiveTab(reload bool) {
	tab := shell.Browser.ActiveTab()
	if tab == nil || tab.URL == nil {
		shell.page = nil
		return
	}
	if tab.URL.Scheme == "about" {
		shell.page = nil
		shell.Status = tab.URL.String()
		return
	}
	shell.Status = "Loading " + tab.URL.String()
	source, pageURL, err := shell.fetch(tab.URL, reload)
	if err != nil {
		shell.Error = err.Error()
		shell.page = nil
		shell.dirty = true
		return
	}
	document, err := html.Parse(source, html.ParseOptions{URL: pageURL})
	if err != nil {
		shell.Error = err.Error()
		shell.page = nil
		shell.dirty = true
		return
	}
	engine := styleEngine(document)
	shell.page = &pageState{document: document, engine: engine, url: pageURL}
	if tab.URL != nil {
		tab.URL = pageURL
	}
	if title := documentTitle(document); title != "" {
		tab.Title = title
	} else if pageURL != nil {
		tab.Title = pageURL.Hostname()
	}
	shell.Status = fmt.Sprintf("Loaded %s", tab.Title)
	shell.Error = ""
	shell.dirty = true
}

func documentTitle(document *dom.Document) string {
	title := ""
	document.Node.Descendants(func(node *dom.Node) bool {
		if node.Type == dom.ElementNode && node.Data == "title" && title == "" {
			title = strings.TrimSpace(node.TextContent())
		}
		return true
	})
	return title
}

func (shell *Shell) fetch(target *urls.URL, reload bool) ([]byte, *urls.URL, error) {
	switch target.Scheme {
	case "http", "https":
		if shell.Network == nil {
			return nil, target, fmt.Errorf("network broker is unavailable")
		}
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		response, err := shell.Network.Fetch(ctx, network.Request{
			URL:       target.String(),
			Method:    "GET",
			Partition: target.Site(),
			Reload:    reload,
		})
		if err != nil {
			return nil, target, err
		}
		if response.Status < 200 || response.Status >= 300 {
			return nil, target, fmt.Errorf("HTTP %d", response.Status)
		}
		if len(response.Body) > 8<<20 {
			return nil, target, fmt.Errorf("response is too large")
		}
		return response.Body, target, nil
	default:
		return nil, target, fmt.Errorf("unsupported scheme %q", target.Scheme)
	}
}

func (shell *Shell) fetchResource(resource string) ([]byte, error) {
	target, err := urls.Parse(resource)
	if err != nil {
		return nil, err
	}
	if target.Scheme != "http" && target.Scheme != "https" && target.Scheme != "data" {
		return nil, fmt.Errorf("unsupported resource scheme %q", target.Scheme)
	}
	if target.Scheme == "data" {
		_, body, ok := target.DataURL()
		if !ok {
			return nil, fmt.Errorf("invalid data URL")
		}
		return body, nil
	}
	if shell.Network == nil {
		return nil, fmt.Errorf("network broker is unavailable")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	response, err := shell.Network.Fetch(ctx, network.Request{URL: target.String(), Method: "GET", Partition: target.Site()})
	if err != nil {
		return nil, err
	}
	if response.Status < 200 || response.Status >= 300 {
		return nil, fmt.Errorf("HTTP %d", response.Status)
	}
	return response.Body, nil
}

// SaveSession stores the browser session through the storage broker.
func (shell *Shell) SaveSession(ctx context.Context) error {
	if shell.Storage == nil {
		return nil
	}
	data, err := json.Marshal(shell.Browser.Session())
	if err != nil {
		return err
	}
	return shell.Storage.Set(ctx, "session", "gonetsurf", "active", data)
}

// LoadSession restores a session from the storage broker when available.
func (shell *Shell) LoadSession(ctx context.Context) error {
	if shell.Storage == nil {
		return nil
	}
	data, found, err := shell.Storage.Get(ctx, "session", "gonetsurf", "active")
	if err != nil || !found {
		return err
	}
	var session browsers.Session
	if err := json.Unmarshal(data, &session); err != nil {
		return err
	}
	restored, err := browsers.Restore(session)
	if err != nil {
		return err
	}
	shell.Browser = restored
	shell.syncActiveTab()
	return nil
}

func (shell *Shell) resourceOptions(base *urls.URL) rendering.Options {
	return rendering.Options{
		Images: rendering.URLImageProvider{
			Base:  base,
			Fetch: shell.fetchResource,
		},
		LoadFont: shell.fetchResource,
	}
}

package disk

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// Entry is a journal record mapping a URL to its local path with cache
// metadata.
type Entry struct {
	URL          string    `json:"url"`
	Path         string    `json:"path"`
	Kind         string    `json:"kind,omitempty"`
	Status       int       `json:"status,omitempty"`
	ContentType  string    `json:"contentType,omitempty"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	CapturedAt   time.Time `json:"capturedAt,omitempty"`
}

// Journal is an append-only record of URL-to-path mappings and redirects.
type Journal struct {
	path string

	mutex     sync.RWMutex
	byURL     map[string]Entry
	byPath    map[string]Entry
	redirects map[string]Redirect
	file      *os.File
	writer    *bufio.Writer
}

// Redirect records a redirect from one URL to another.
type Redirect struct {
	From   string    `json:"from"`
	To     string    `json:"to"`
	Status int       `json:"status"`
	At     time.Time `json:"at"`
}

type journalRecord struct {
	Type         string    `json:"type"`
	URL          string    `json:"url,omitempty"`
	Path         string    `json:"path,omitempty"`
	Kind         string    `json:"kind,omitempty"`
	Status       int       `json:"status,omitempty"`
	ContentType  string    `json:"contentType,omitempty"`
	ETag         string    `json:"etag,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	CapturedAt   time.Time `json:"capturedAt,omitempty"`
	From         string    `json:"from,omitempty"`
	To           string    `json:"to,omitempty"`
	At           time.Time `json:"at,omitempty"`
}

// OpenJournal opens or creates a journal file and loads its records.
func OpenJournal(path string) (*Journal, error) {
	journal := &Journal{
		path:      path,
		byURL:     make(map[string]Entry),
		byPath:    make(map[string]Entry),
		redirects: make(map[string]Redirect),
	}
	if err := journal.load(); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	journal.file = file
	journal.writer = bufio.NewWriter(file)
	return journal, nil
}

func (journal *Journal) load() error {
	file, err := os.Open(journal.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var record journalRecord
		if err := json.Unmarshal(line, &record); err != nil {
			continue
		}
		journal.apply(record)
	}
	return scanner.Err()
}

func (journal *Journal) apply(record journalRecord) {
	switch record.Type {
	case "redirect":
		journal.redirects[record.From] = Redirect{From: record.From, To: record.To, Status: record.Status, At: record.At}
	default:
		entry := Entry{
			URL:          record.URL,
			Path:         record.Path,
			Kind:         record.Kind,
			Status:       record.Status,
			ContentType:  record.ContentType,
			ETag:         record.ETag,
			LastModified: record.LastModified,
			CapturedAt:   record.CapturedAt,
		}
		journal.byURL[entry.URL] = entry
		journal.byPath[entry.Path] = entry
	}
}

// Add records an entry.
func (journal *Journal) Add(entry Entry) error {
	if entry.CapturedAt.IsZero() {
		entry.CapturedAt = time.Now().UTC()
	}
	journal.mutex.Lock()
	defer journal.mutex.Unlock()
	journal.byURL[entry.URL] = entry
	journal.byPath[entry.Path] = entry
	return journal.append(journalRecord{
		Type:         "entry",
		URL:          entry.URL,
		Path:         entry.Path,
		Kind:         entry.Kind,
		Status:       entry.Status,
		ContentType:  entry.ContentType,
		ETag:         entry.ETag,
		LastModified: entry.LastModified,
		CapturedAt:   entry.CapturedAt,
	})
}

// AddRedirect records a redirect.
func (journal *Journal) AddRedirect(redirect Redirect) error {
	if redirect.At.IsZero() {
		redirect.At = time.Now().UTC()
	}
	journal.mutex.Lock()
	defer journal.mutex.Unlock()
	journal.redirects[redirect.From] = redirect
	return journal.append(journalRecord{
		Type:   "redirect",
		From:   redirect.From,
		To:     redirect.To,
		Status: redirect.Status,
		At:     redirect.At,
	})
}

// LookupURL returns the entry for a URL.
func (journal *Journal) LookupURL(url string) (Entry, bool) {
	journal.mutex.RLock()
	defer journal.mutex.RUnlock()
	entry, ok := journal.byURL[url]
	return entry, ok
}

// LookupPath returns the entry for a local path.
func (journal *Journal) LookupPath(path string) (Entry, bool) {
	journal.mutex.RLock()
	defer journal.mutex.RUnlock()
	entry, ok := journal.byPath[path]
	return entry, ok
}

// Redirects returns the redirect recorded for a URL, if any.
func (journal *Journal) Redirects(url string) (Redirect, bool) {
	journal.mutex.RLock()
	defer journal.mutex.RUnlock()
	redirect, ok := journal.redirects[url]
	return redirect, ok
}

// RedirectChain follows redirects from url, returning the final URL.
func (journal *Journal) RedirectChain(url string) string {
	seen := make(map[string]struct{})
	current := url
	for {
		if _, exists := seen[current]; exists {
			return current
		}
		seen[current] = struct{}{}
		redirect, ok := journal.Redirects(current)
		if !ok {
			return current
		}
		current = redirect.To
	}
}

func (journal *Journal) append(record journalRecord) error {
	if journal.writer == nil {
		return nil
	}
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	if _, err := journal.writer.Write(data); err != nil {
		return err
	}
	if err := journal.writer.WriteByte('\n'); err != nil {
		return err
	}
	return journal.writer.Flush()
}

// Close flushes and closes the journal.
func (journal *Journal) Close() error {
	journal.mutex.Lock()
	defer journal.mutex.Unlock()
	if journal.writer != nil {
		if err := journal.writer.Flush(); err != nil {
			return err
		}
	}
	if journal.file != nil {
		return journal.file.Close()
	}
	return nil
}

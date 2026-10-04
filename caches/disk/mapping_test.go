package disk

import (
	"testing"

	"gonetsurf/urls"
)

func TestRelativePath(t *testing.T) {
	layout := NewLayout("/home/user/NetSurf/cache")
	tests := []struct {
		input string
		want  string
	}{
		{"https://example.com/what/ever/index.html", "example.com/what/ever/index.html"},
		{"https://example.com/", "example.com/index.html"},
		{"https://example.com", "example.com/index.html"},
		{"https://whatever.com/index.php?category", "whatever.com/index.php/~q~/category/index.html"},
		{"https://whatever.com/index.php?category/subcategory", "whatever.com/index.php/~q~/category/subcategory/index.html"},
		{"https://example.com/a?b=c", "example.com/a/~q~/b=c/index.html"},
		{"https://example.com/?q=x", "example.com/~q~/q=x/index.html"},
		{"http://example.com/", "~http~example.com/index.html"},
		{"https://example.com:8443/x", "example.com~8443/x"},
	}
	for _, test := range tests {
		parsed, err := urls.Parse(test.input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", test.input, err)
		}
		if got := layout.RelativePath(parsed); got != test.want {
			t.Errorf("RelativePath(%q) = %q, want %q", test.input, got, test.want)
		}
	}
}

func TestQueryVariantsDoNotCollide(t *testing.T) {
	layout := NewLayout("cache")
	article, _ := urls.Parse("https://whatever.com/index.php?category")
	sub, _ := urls.Parse("https://whatever.com/index.php?category/subcategory")
	if layout.RelativePath(article) == layout.RelativePath(sub) {
		t.Fatal("query variants collided")
	}
}

func TestURLForRoundTrip(t *testing.T) {
	layout := NewLayout("cache")
	inputs := []string{
		"https://whatever.com/index.php?category",
		"https://whatever.com/index.php?category/subcategory",
		"https://example.com/a?b=c",
		"https://example.com/?q=x",
		"http://example.com/",
		"https://example.com:8443/x",
	}
	for _, input := range inputs {
		parsed, err := urls.Parse(input)
		if err != nil {
			t.Fatalf("Parse(%q): %v", input, err)
		}
		relative := layout.RelativePath(parsed)
		restored, err := layout.URLFor(relative)
		if err != nil {
			t.Fatalf("URLFor(%q): %v", relative, err)
		}
		if restored.String() != parsed.String() {
			t.Errorf("round trip: %q -> %q -> %q", input, relative, restored.String())
		}
	}
}

func TestJournal(t *testing.T) {
	directory := t.TempDir()
	journal, err := OpenJournal(directory + "/.netsurf/index.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	entry := Entry{URL: "https://example.com/", Path: "example.com/index.html", Kind: "document", Status: 200}
	if err := journal.Add(entry); err != nil {
		t.Fatal(err)
	}
	if err := journal.AddRedirect(Redirect{From: "http://example.com/", To: "https://example.com/", Status: 301}); err != nil {
		t.Fatal(err)
	}
	if got := journal.RedirectChain("http://example.com/"); got != "https://example.com/" {
		t.Errorf("RedirectChain = %q", got)
	}
	if stored, ok := journal.LookupPath("example.com/index.html"); !ok || stored.URL != entry.URL {
		t.Errorf("LookupPath = (%+v, %v)", stored, ok)
	}
	if err := journal.Close(); err != nil {
		t.Fatal(err)
	}

	reopened, err := OpenJournal(directory + "/.netsurf/index.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	if _, ok := reopened.LookupURL("https://example.com/"); !ok {
		t.Error("expected the journal to persist entries")
	}
	if got := reopened.RedirectChain("http://example.com/"); got != "https://example.com/" {
		t.Errorf("persisted RedirectChain = %q", got)
	}
}

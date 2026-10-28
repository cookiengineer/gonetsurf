package fonts

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"encoding/binary"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"golang.org/x/image/font/gofont/gomono"
	"golang.org/x/image/font/gofont/goregular"

	"gonetsurf/parsers/css/cssom"
)

func fontFaceRule(t *testing.T, source string) *cssom.FontFaceRule {
	t.Helper()
	stylesheet := cssom.Parse(source)
	if len(stylesheet.Rules) != 1 {
		t.Fatalf("rules = %d", len(stylesheet.Rules))
	}
	rule, ok := stylesheet.Rules[0].(*cssom.FontFaceRule)
	if !ok {
		t.Fatalf("rule = %T", stylesheet.Rules[0])
	}
	return rule
}

func fontDataURL(mediaType string, data []byte) string {
	return "data:" + mediaType + ";base64," + base64.StdEncoding.EncodeToString(data)
}

func TestAddAuthorFaceFromDataURL(t *testing.T) {
	rule := fontFaceRule(t, `@font-face { font-family: "AuthorTest"; src: url(`+fontDataURL("font/ttf", goregular.TTF)+`) format("truetype") }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{})
	face := library.Face(Spec{Family: "AuthorTest", Size: 20, Weight: 400})
	if face == nil {
		t.Fatal("author face is nil")
	}
	if _, _, ok := face.GlyphBounds('A'); !ok {
		t.Fatal("author face has no A glyph")
	}
}

func TestAddAuthorFaceFromLocal(t *testing.T) {
	rule := fontFaceRule(t, `@font-face { font-family: "LocalTest"; src: local("Go") }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{})
	face := library.Face(Spec{Family: "LocalTest", Size: 16})
	if face == nil {
		t.Fatal("local author face is nil")
	}
	if _, _, ok := face.GlyphBounds('Z'); !ok {
		t.Fatal("local author face has no Z glyph")
	}
}

func TestAuthorFaceLoader(t *testing.T) {
	rule := fontFaceRule(t, `@font-face { font-family: "LoaderTest"; src: url("https://example.com/font.ttf") format("truetype") }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{
		Loader: func(resource string) ([]byte, error) {
			if !strings.HasSuffix(resource, "/font.ttf") {
				t.Errorf("loader resource = %q", resource)
			}
			return goregular.TTF, nil
		},
	})
	if face := library.Face(Spec{Family: "LoaderTest", Size: 16}); face == nil {
		t.Fatal("loader-backed face is nil")
	}
}

func TestAuthorUnicodeRanges(t *testing.T) {
	upper := fontFaceRule(t, `@font-face { font-family: "RangedTest"; src: url(`+fontDataURL("font/ttf", goregular.TTF)+`); unicode-range: U+0041-005A }`)
	lower := fontFaceRule(t, `@font-face { font-family: "RangedTest"; src: url(`+fontDataURL("font/ttf", gomono.TTF)+`); unicode-range: U+0061-007A }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{upper, lower}, AuthorOptions{})
	faces := library.Faces(Spec{Family: "RangedTest", Size: 16})
	if len(faces) < 2 {
		t.Fatalf("faces = %d, want at least 2", len(faces))
	}
	uppercaseFace := FaceForRuneChain(faces, 'A')
	lowercaseFace := FaceForRuneChain(faces, 'a')
	if uppercaseFace == nil || lowercaseFace == nil {
		t.Fatal("missing range-selected faces")
	}
	if _, _, ok := uppercaseFace.GlyphBounds('a'); ok {
		t.Error("uppercase face accepted a lowercase glyph")
	}
	if _, _, ok := lowercaseFace.GlyphBounds('A'); ok {
		t.Error("lowercase face accepted an uppercase glyph")
	}
}

func TestAuthorWeightRangeAndOverrides(t *testing.T) {
	regular := fontFaceRule(t, `@font-face { font-family: "Weighted"; src: url(`+fontDataURL("font/ttf", goregular.TTF)+`); font-weight: 400 }`)
	bold := fontFaceRule(t, `@font-face { font-family: "Weighted"; src: url(`+fontDataURL("font/ttf", goregular.TTF)+`); font-weight: 700; size-adjust: 50%; ascent-override: 80%; descent-override: 20% }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{regular, bold}, AuthorOptions{})
	if entry := library.selectEntry(Spec{Family: "Weighted", Size: 16, Weight: 700}); entry == nil || entry.weightMax() != 700 {
		t.Fatalf("bold selection = %#v", entry)
	}
	metrics := library.MetricsForSpec(Spec{Family: "Weighted", Size: 20, Weight: 700})
	if metrics.Ascent < 15.9 || metrics.Ascent > 16.1 {
		t.Errorf("ascent override = %v, want 16", metrics.Ascent)
	}
	if metrics.Descent < 3.9 || metrics.Descent > 4.1 {
		t.Errorf("descent override = %v, want 4", metrics.Descent)
	}
}

func buildWOFF(t *testing.T, data []byte) []byte {
	t.Helper()
	if len(data) < 12 {
		t.Fatal("font is too short")
	}
	flavor := data[0:4]
	numTables := int(binary.BigEndian.Uint16(data[4:6]))
	type tableEntry struct {
		tag      uint32
		checksum uint32
		data     []byte
	}
	entries := make([]tableEntry, numTables)
	for index := 0; index < numTables; index++ {
		offset := 12 + index*16
		tag := binary.BigEndian.Uint32(data[offset : offset+4])
		checksum := binary.BigEndian.Uint32(data[offset+4 : offset+8])
		tableOffset := int(binary.BigEndian.Uint32(data[offset+8 : offset+12]))
		length := int(binary.BigEndian.Uint32(data[offset+12 : offset+16]))
		entries[index] = tableEntry{tag: tag, checksum: checksum, data: data[tableOffset : tableOffset+length]}
	}
	headerLength := 44 + numTables*20
	directory := make([]byte, headerLength)
	copy(directory[0:4], "wOFF")
	copy(directory[4:8], flavor)
	binary.BigEndian.PutUint16(directory[12:14], uint16(numTables))
	binary.BigEndian.PutUint32(directory[16:20], uint32(len(data)))
	payload := make([]byte, 0)
	offset := headerLength
	for index, entry := range entries {
		var compressed bytes.Buffer
		writer := zlib.NewWriter(&compressed)
		writer.Write(entry.data)
		writer.Close()
		tableData := compressed.Bytes()
		if len(tableData) >= len(entry.data) {
			tableData = entry.data
		}
		recordOffset := 44 + index*20
		binary.BigEndian.PutUint32(directory[recordOffset:recordOffset+4], entry.tag)
		binary.BigEndian.PutUint32(directory[recordOffset+4:recordOffset+8], uint32(offset))
		binary.BigEndian.PutUint32(directory[recordOffset+8:recordOffset+12], uint32(len(tableData)))
		binary.BigEndian.PutUint32(directory[recordOffset+12:recordOffset+16], uint32(len(entry.data)))
		binary.BigEndian.PutUint32(directory[recordOffset+16:recordOffset+20], entry.checksum)
		payload = append(payload, tableData...)
		offset += len(tableData)
		for offset%4 != 0 {
			payload = append(payload, 0)
			offset++
		}
	}
	result := append(directory, payload...)
	binary.BigEndian.PutUint32(result[8:12], uint32(len(result)))
	return result
}

func TestWOFFAuthorFace(t *testing.T) {
	woff := buildWOFF(t, goregular.TTF)
	rule := fontFaceRule(t, `@font-face { font-family: "WOFFTest"; src: url(`+fontDataURL("font/woff", woff)+`) format("woff") }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{})
	if face := library.Face(Spec{Family: "WOFFTest", Size: 16}); face == nil {
		t.Fatal("WOFF face is nil")
	}
}

func TestFontResourceCachePartitions(t *testing.T) {
	cache := newFontResourceCache()
	firstKey := fontResourceKey{partition: "site-a", url: "https://example.com/font.ttf"}
	cache.put(firstKey, []byte("font"), nil)
	if data, _, ok := cache.get(firstKey); !ok || string(data) != "font" {
		t.Errorf("cache hit = (%q,%v)", data, ok)
	}
	secondKey := fontResourceKey{partition: "site-b", url: "https://example.com/font.ttf"}
	if _, _, ok := cache.get(secondKey); ok {
		t.Error("cache leaked across partitions")
	}
}

func TestFontDisplaySwap(t *testing.T) {
	loaderStarted := make(chan struct{})
	releaseLoader := make(chan struct{})
	loader := func(resource string) ([]byte, error) {
		close(loaderStarted)
		<-releaseLoader
		return goregular.TTF, nil
	}
	rule := fontFaceRule(t, `@font-face { font-family: "SwapTest"; src: url("https://example.com/swap.ttf"); font-display: swap }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{Loader: loader})
	if face := library.Face(Spec{Family: "SwapTest", Size: 16}); face != nil {
		t.Fatal("swap face should not block the first render")
	}
	<-loaderStarted
	close(releaseLoader)
	deadline := time.Now().Add(2 * time.Second)
	for library.Face(Spec{Family: "SwapTest", Size: 16}) == nil && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if face := library.Face(Spec{Family: "SwapTest", Size: 16}); face == nil {
		t.Fatal("swap face did not become available")
	}
}

func TestFontDisplayBlock(t *testing.T) {
	rule := fontFaceRule(t, `@font-face { font-family: "BlockTest"; src: url(`+fontDataURL("font/ttf", goregular.TTF)+`); font-display: block }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{})
	if face := library.Face(Spec{Family: "BlockTest", Size: 16}); face == nil {
		t.Fatal("block face was not loaded synchronously")
	}
}

func TestFontLoaderCacheAndPartitions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		writer.Write(goregular.TTF)
	}))
	defer server.Close()
	var loads int32
	loader := func(resource string) ([]byte, error) {
		atomic.AddInt32(&loads, 1)
		response, err := http.Get(resource)
		if err != nil {
			return nil, err
		}
		defer response.Body.Close()
		return io.ReadAll(response.Body)
	}
	rule := fontFaceRule(t, `@font-face { font-family: "CacheFace"; src: url("`+server.URL+`/font.ttf") }`)
	base := NewLibrary()
	base.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{Partition: "site-a", Loader: loader})
	if face := base.Face(Spec{Family: "CacheFace", Size: 16}); face == nil {
		t.Fatal("initial font load failed")
	}
	if atomic.LoadInt32(&loads) != 1 {
		t.Fatalf("loads = %d, want 1", loads)
	}
	samePartition := NewLibrary()
	samePartition.resourceCache = base.resourceCache
	samePartition.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{Partition: "site-a", Loader: loader})
	if face := samePartition.Face(Spec{Family: "CacheFace", Size: 16}); face == nil {
		t.Fatal("cached font load failed")
	}
	if atomic.LoadInt32(&loads) != 1 {
		t.Errorf("same-partition cache miss: loads = %d", loads)
	}
	other := NewLibrary()
	other.resourceCache = base.resourceCache
	other.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{Partition: "site-b", Loader: loader})
	if face := other.Face(Spec{Family: "CacheFace", Size: 16}); face == nil {
		t.Fatal("other-partition font load failed")
	}
	if atomic.LoadInt32(&loads) != 2 {
		t.Errorf("cross-partition cache leak: loads = %d", loads)
	}
}

func TestFontFailureFallsBack(t *testing.T) {
	rule := fontFaceRule(t, `@font-face { font-family: "BrokenFace"; src: url("https://example.com/broken.ttf") }`)
	library := NewLibrary()
	library.AddFontFaceRules([]*cssom.FontFaceRule{rule}, AuthorOptions{
		Loader: func(resource string) ([]byte, error) {
			return []byte("not a font"), nil
		},
	})
	if face := library.Face(Spec{Family: "BrokenFace", Size: 16}); face != nil {
		t.Fatal("broken font produced a face")
	}
	faces := library.Faces(Spec{Family: "BrokenFace", Size: 16})
	if len(faces) == 0 {
		t.Fatal("missing fallback chain")
	}
	if _, _, ok := faces[0].GlyphBounds('A'); !ok {
		t.Fatal("fallback face has no glyphs")
	}
}

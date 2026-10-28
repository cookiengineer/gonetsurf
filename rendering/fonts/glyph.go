package fonts

import (
	"image"
	"image/draw"
	"reflect"
	"sync"

	"golang.org/x/image/font"
	"golang.org/x/image/math/fixed"
)

// CachedGlyph is a rasterized glyph mask positioned relative to the text origin.
type CachedGlyph struct {
	Bounds    image.Rectangle
	Mask      *image.Alpha
	MaskPoint image.Point
}

type glyphKey struct {
	face      font.Face
	character rune
}

// GlyphCache caches rasterized glyph masks per face and character.
type GlyphCache struct {
	mutex   sync.Mutex
	entries map[glyphKey]CachedGlyph
	max     int
}

// NewGlyphCache returns an empty glyph cache.
func NewGlyphCache() *GlyphCache {
	return &GlyphCache{entries: make(map[glyphKey]CachedGlyph), max: 8192}
}

// Glyph returns the cached mask for character, rasterizing it at the origin on
// a miss.
func (cache *GlyphCache) Glyph(face font.Face, character rune) (CachedGlyph, bool) {
	if cache == nil || !cacheableFace(face) {
		return CachedGlyph{}, false
	}
	key := glyphKey{face: face, character: character}
	cache.mutex.Lock()
	if glyph, ok := cache.entries[key]; ok {
		cache.mutex.Unlock()
		return glyph, true
	}
	cache.mutex.Unlock()

	bounds, mask, maskPoint, _, ok := face.Glyph(fixed.P(0, 0), character)
	if !ok || mask == nil {
		return CachedGlyph{}, false
	}
	copyBounds := image.Rect(0, 0, bounds.Dx(), bounds.Dy())
	copied := image.NewAlpha(copyBounds)
	draw.Draw(copied, copyBounds, mask, maskPoint, draw.Src)
	glyph := CachedGlyph{Bounds: bounds, Mask: copied}

	cache.mutex.Lock()
	if len(cache.entries) >= cache.max {
		cache.entries = make(map[glyphKey]CachedGlyph)
	}
	cache.entries[key] = glyph
	cache.mutex.Unlock()
	return glyph, true
}

func cacheableFace(face font.Face) bool {
	if face == nil {
		return false
	}
	faceType := reflect.TypeOf(face)
	return faceType != nil && faceType.Comparable()
}

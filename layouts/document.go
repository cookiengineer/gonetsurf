package layouts

import (
	"gonetsurf/dom"
	"gonetsurf/styles"
)

// DocumentLayout caches a laid-out box tree for a document and invalidates it
// when the document mutation version changes or the viewport is updated.
type DocumentLayout struct {
	Document       *dom.Document
	Engine         *styles.Engine
	ViewportWidth  float64
	ViewportHeight float64

	box       *Box
	layout    *Layout
	version   uint64
	hasLayout bool
}

// NewDocumentLayout creates a layout cache for a document.
func NewDocumentLayout(document *dom.Document, engine *styles.Engine, viewportWidth, viewportHeight float64) *DocumentLayout {
	return &DocumentLayout{
		Document:       document,
		Engine:         engine,
		ViewportWidth:  viewportWidth,
		ViewportHeight: viewportHeight,
	}
}

// Invalidate discards the cached layout.
func (documentLayout *DocumentLayout) Invalidate() {
	documentLayout.hasLayout = false
	documentLayout.box = nil
}

// Box returns the cached box tree, or nil before the first reflow.
func (documentLayout *DocumentLayout) Box() *Box {
	return documentLayout.box
}

// Reflow returns the laid-out box tree, rebuilding only when the document has
// mutated or the cache was invalidated.
func (documentLayout *DocumentLayout) Reflow() *Box {
	if documentLayout.Document == nil {
		return nil
	}
	if documentLayout.hasLayout && documentLayout.version == documentLayout.Document.MutationVersion() {
		return documentLayout.box
	}
	documentLayout.layout = New(documentLayout.ViewportWidth, documentLayout.ViewportHeight)
	documentLayout.box = BuildDocument(documentLayout.Document, documentLayout.Engine)
	documentLayout.layout.Layout(documentLayout.box)
	documentLayout.version = documentLayout.Document.MutationVersion()
	documentLayout.hasLayout = true
	return documentLayout.box
}

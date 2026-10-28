package layouts

import (
	"gonetsurf/styles"

	"golang.org/x/text/unicode/bidi"
)

// TextRun is one shaped text run.
type TextRun struct {
	Text          string
	FontSize      float64
	LetterSpacing float64
	WordSpacing   float64
}

// Glyph is one shaped character advance.
type Glyph struct {
	Text    string
	Advance float64
}

// TextShaper shapes a text run into glyph advances. A full OpenType shaper
// implements the same interface in a later phase.
type TextShaper interface {
	Shape(run TextRun) []Glyph
}

// SimpleShaper uses a fixed average advance width and spacing properties.
type SimpleShaper struct{}

// Shape implements TextShaper.
func (SimpleShaper) Shape(run TextRun) []Glyph {
	glyphs := make([]Glyph, 0, len([]rune(run.Text)))
	for _, character := range run.Text {
		advance := run.FontSize*0.5 + run.LetterSpacing
		if character == ' ' {
			advance += run.WordSpacing
		}
		glyphs = append(glyphs, Glyph{Text: string(character), Advance: advance})
	}
	return glyphs
}

func (layout *Layout) shapeText(text string, style *styles.Style) []Glyph {
	fontSize, _ := fontMetrics(style)
	if fontSize <= 0 {
		fontSize = 16
	}
	letterSpacing := 0.0
	wordSpacing := 0.0
	if style != nil {
		if value, ok := parseComputedPixels(style.GetString("letter-spacing")); ok {
			letterSpacing = value
		}
		if value, ok := parseComputedPixels(style.GetString("word-spacing")); ok {
			wordSpacing = value
		}
	}
	shaper := layout.Shaper
	if shaper == nil {
		shaper = SimpleShaper{}
	}
	return shaper.Shape(TextRun{Text: text, FontSize: fontSize, LetterSpacing: letterSpacing, WordSpacing: wordSpacing})
}

// BidiRun is a directional run of text.
type BidiRun struct {
	Text      string
	Direction bidi.Direction
	Start     int
	End       int
}

// SegmentBidi splits text into directional runs. The ordering is the bidi
// visual ordering from x/text; line layout integration is a later refinement.
func SegmentBidi(text string, defaultDirection bidi.Direction) []BidiRun {
	var paragraph bidi.Paragraph
	if _, err := paragraph.SetString(text, bidi.DefaultDirection(defaultDirection)); err != nil {
		return []BidiRun{{Text: text, Direction: defaultDirection}}
	}
	ordering, err := paragraph.Order()
	if err != nil || ordering.NumRuns() == 0 {
		return []BidiRun{{Text: text, Direction: defaultDirection}}
	}
	runs := make([]BidiRun, 0, ordering.NumRuns())
	for index := 0; index < ordering.NumRuns(); index++ {
		run := ordering.Run(index)
		start, end := run.Pos()
		runs = append(runs, BidiRun{
			Text:      run.String(),
			Direction: run.Direction(),
			Start:     start,
			End:       end,
		})
	}
	return runs
}

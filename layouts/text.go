package layouts

import (
	"strings"

	"gonetsurf/styles"

	"golang.org/x/text/unicode/bidi"
)

// TextRun is one shaped text run.
type TextRun struct {
	Text           string
	FontSize       float64
	LetterSpacing  float64
	WordSpacing    float64
	AverageAdvance float64
}

// FontMetrics describes the vertical metrics and average advance for a font.
type FontMetrics struct {
	Ascent         float64
	Descent        float64
	LineGap        float64
	AverageAdvance float64
}

// MetricsForStyle returns simple metrics for the computed font style.
func MetricsForStyle(style *styles.Style) FontMetrics {
	fontSize := 16.0
	if style != nil {
		if value, ok := parseComputedPixels(style.GetString("font-size")); ok && value > 0 {
			fontSize = value
		}
	}
	advanceFactor := 0.5
	if style != nil {
		family := strings.ToLower(style.GetString("font-family"))
		if strings.Contains(family, "monospace") || strings.Contains(family, "cursive") {
			advanceFactor = 0.6
		}
	}
	return FontMetrics{
		Ascent:         fontSize * 0.8,
		Descent:        fontSize * 0.2,
		LineGap:        0,
		AverageAdvance: fontSize * advanceFactor,
	}
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
	baseAdvance := run.AverageAdvance
	if baseAdvance <= 0 {
		baseAdvance = run.FontSize * 0.5
	}
	for _, character := range run.Text {
		advance := baseAdvance + run.LetterSpacing
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
	metrics := MetricsForStyle(style)
	return shaper.Shape(TextRun{Text: text, FontSize: fontSize, LetterSpacing: letterSpacing, WordSpacing: wordSpacing, AverageAdvance: metrics.AverageAdvance})
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

type visualWord struct {
	Text   string
	Visual string
}

func visualWords(text, direction string) []visualWord {
	defaultDirection := bidi.LeftToRight
	if strings.EqualFold(strings.TrimSpace(direction), "rtl") {
		defaultDirection = bidi.RightToLeft
	}
	runs := SegmentBidi(text, defaultDirection)
	words := []visualWord{}
	for _, run := range runs {
		runWords := strings.Fields(run.Text)
		if run.Direction == bidi.RightToLeft {
			for left, right := 0, len(runWords)-1; left < right; left, right = left+1, right-1 {
				runWords[left], runWords[right] = runWords[right], runWords[left]
			}
			for _, word := range runWords {
				words = append(words, visualWord{Text: word, Visual: reverseRunes(word)})
			}
			continue
		}
		for _, word := range runWords {
			words = append(words, visualWord{Text: word, Visual: word})
		}
	}
	return words
}

func reverseRunes(text string) string {
	runes := []rune(text)
	for left, right := 0, len(runes)-1; left < right; left, right = left+1, right-1 {
		runes[left], runes[right] = runes[right], runes[left]
	}
	return string(runes)
}

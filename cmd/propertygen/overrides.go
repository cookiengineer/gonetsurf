package main

// manualProperties fills gaps in the pinned specifications and the WPT
// grammar assertions. Only fields with a *Set flag are applied.

func generated(name, grammar string) propertySpec {
	return propertySpec{Name: name, Grammar: grammar, GrammarSet: true}
}

func generatedWith(name, grammar, initial string, inherited bool, animation string) propertySpec {
	return propertySpec{
		Name:         name,
		Grammar:      grammar,
		GrammarSet:   true,
		Initial:      initial,
		InitialSet:   true,
		Inherited:    inherited,
		InheritedSet: true,
		Animation:    animation,
		AnimationSet: true,
	}
}

var manualProperties = map[string]propertySpec{
	"all":         generatedWith("all", "initial | inherit | unset | revert | revert-layer", "", false, "discrete"),
	"display":     generatedWith("display", "<display-listitem> | [ <display-outside> || <display-inside> ] | <display-internal> | <display-box> | <display-legacy>", "inline", false, "discrete"),
	"line-height": generatedWith("line-height", "normal | <number-nonnegative> | <length-nonnegative> | <percentage-nonnegative>", "normal", true, "by computed value"),

	"width":      generatedWith("width", "auto | <box-size>", "auto", false, "by computed value"),
	"height":     generatedWith("height", "auto | <box-size>", "auto", false, "by computed value"),
	"min-width":  generatedWith("min-width", "auto | <box-size>", "auto", false, "by computed value"),
	"min-height": generatedWith("min-height", "auto | <box-size>", "auto", false, "by computed value"),
	"max-width":  generatedWith("max-width", "none | <box-size>", "none", false, "by computed value"),
	"max-height": generatedWith("max-height", "none | <box-size>", "none", false, "by computed value"),

	"margin-top":      generatedWith("margin-top", "<length-percentage> | auto", "0", false, "by computed value"),
	"margin-right":    generatedWith("margin-right", "<length-percentage> | auto", "0", false, "by computed value"),
	"margin-bottom":   generatedWith("margin-bottom", "<length-percentage> | auto", "0", false, "by computed value"),
	"margin-left":     generatedWith("margin-left", "<length-percentage> | auto", "0", false, "by computed value"),
	"padding-top":     generatedWith("padding-top", "<length-percentage [0,∞]>", "0", false, "by computed value"),
	"padding-right":   generatedWith("padding-right", "<length-percentage [0,∞]>", "0", false, "by computed value"),
	"padding-bottom":  generatedWith("padding-bottom", "<length-percentage [0,∞]>", "0", false, "by computed value"),
	"padding-left":    generatedWith("padding-left", "<length-percentage [0,∞]>", "0", false, "by computed value"),
	"overflow-block":  generatedWith("overflow-block", "visible | hidden | clip | scroll | auto | no-display | no-content", "visible", false, "discrete"),
	"overflow-inline": generatedWith("overflow-inline", "visible | hidden | clip | scroll | auto | no-display | no-content", "visible", false, "discrete"),
	"overflow-x":      generatedWith("overflow-x", "visible | hidden | clip | scroll | auto | no-display | no-content", "visible", false, "discrete"),
	"overflow-y":      generatedWith("overflow-y", "visible | hidden | clip | scroll | auto | no-display | no-content", "visible", false, "discrete"),

	"border":        generatedWith("border", "<line-width-nonnegative> || <line-style> || <color>", "medium none currentColor", false, "discrete"),
	"border-top":    generatedWith("border-top", "<line-width-nonnegative> || <line-style> || <color>", "medium none currentColor", false, "discrete"),
	"border-right":  generatedWith("border-right", "<line-width-nonnegative> || <line-style> || <color>", "medium none currentColor", false, "discrete"),
	"border-bottom": generatedWith("border-bottom", "<line-width-nonnegative> || <line-style> || <color>", "medium none currentColor", false, "discrete"),
	"border-left":   generatedWith("border-left", "<line-width-nonnegative> || <line-style> || <color>", "medium none currentColor", false, "discrete"),
	"border-width":  generatedWith("border-width", "<line-width-nonnegative>{1,4}", "medium", false, "by computed value"),

	"border-top-color":    generatedWith("border-top-color", "<color>", "currentColor", false, "by computed value"),
	"border-right-color":  generatedWith("border-right-color", "<color>", "currentColor", false, "by computed value"),
	"border-bottom-color": generatedWith("border-bottom-color", "<color>", "currentColor", false, "by computed value"),
	"border-left-color":   generatedWith("border-left-color", "<color>", "currentColor", false, "by computed value"),
	"border-top-style":    generatedWith("border-top-style", "<line-style>", "none", false, "discrete"),
	"border-right-style":  generatedWith("border-right-style", "<line-style>", "none", false, "discrete"),
	"border-bottom-style": generatedWith("border-bottom-style", "<line-style>", "none", false, "discrete"),
	"border-left-style":   generatedWith("border-left-style", "<line-style>", "none", false, "discrete"),
	"border-top-width":    generatedWith("border-top-width", "<line-width-nonnegative>", "medium", false, "by computed value"),
	"border-right-width":  generatedWith("border-right-width", "<line-width-nonnegative>", "medium", false, "by computed value"),
	"border-bottom-width": generatedWith("border-bottom-width", "<line-width-nonnegative>", "medium", false, "by computed value"),
	"border-left-width":   generatedWith("border-left-width", "<line-width-nonnegative>", "medium", false, "by computed value"),

	"border-top-left-radius":             generatedWith("border-top-left-radius", "<length-percentage [0,∞]>{1,2}", "0", false, "by computed value"),
	"border-top-right-radius":            generatedWith("border-top-right-radius", "<length-percentage [0,∞]>{1,2}", "0", false, "by computed value"),
	"border-bottom-right-radius":         generatedWith("border-bottom-right-radius", "<length-percentage [0,∞]>{1,2}", "0", false, "by computed value"),
	"border-bottom-left-radius":          generatedWith("border-bottom-left-radius", "<length-percentage [0,∞]>{1,2}", "0", false, "by computed value"),
	"-webkit-border-top-left-radius":     generated("-webkit-border-top-left-radius", "<length-percentage [0,∞]>{1,2}"),
	"-webkit-border-top-right-radius":    generated("-webkit-border-top-right-radius", "<length-percentage [0,∞]>{1,2}"),
	"-webkit-border-bottom-right-radius": generated("-webkit-border-bottom-right-radius", "<length-percentage [0,∞]>{1,2}"),
	"-webkit-border-bottom-left-radius":  generated("-webkit-border-bottom-left-radius", "<length-percentage [0,∞]>{1,2}"),

	"animation-range-start": generatedWith("animation-range-start", "normal | [ [ cover | contain | entry | exit | entry-crossing | exit-crossing ] <length-percentage>? | <length-percentage> ]#", "normal", false, "by computed value"),
	"animation-range-end":   generatedWith("animation-range-end", "normal | [ [ cover | contain | entry | exit | entry-crossing | exit-crossing ] <length-percentage>? | <length-percentage> ]#", "normal", false, "by computed value"),

	"hyphenate-limit-chars": generatedWith("hyphenate-limit-chars", "[ auto | <integer-nonnegative> ]{1,3}", "auto", true, "by computed value"),
	"hyphenate-limit-lines": generatedWith("hyphenate-limit-lines", "no-limit | <integer-nonnegative>", "no-limit", true, "by computed value"),
	"scroll-axis-lock":      generatedWith("scroll-axis-lock", "auto | none", "auto", false, "discrete"),
	"scroll-marker-group":   generatedWith("scroll-marker-group", "none | before | after", "none", false, "discrete"),
	"scroll-target-group":   generatedWith("scroll-target-group", "none | auto", "none", false, "discrete"),
	"size":                  generatedWith("size", "auto | <length>{1,2}", "auto", false, "by computed value"),

	"text-autospace":    generatedWith("text-autospace", "normal | no-autospace | auto | [ ideograph-alpha | ideograph-numeric | punctuation | insert | replace ]+", "normal", true, "discrete"),
	"text-fit":          generatedWith("text-fit", "none | [ grow | shrink ] [ consistent | per-line | per-line-all ]? <percentage-nonnegative>?", "none", true, "by computed value"),
	"text-spacing":      generatedWith("text-spacing", "[ normal | no-autospace | trim-start | space-all | none | auto ]{1,2}", "normal", true, "discrete"),
	"text-spacing-trim": generatedWith("text-spacing-trim", "normal | auto | space-all | trim-both | trim-all | trim-start | space-first", "normal", true, "discrete"),
	"text-wrap-mode":    generatedWith("text-wrap-mode", "wrap | nowrap", "wrap", true, "discrete"),
	"text-wrap-style":   generatedWith("text-wrap-style", "auto | balance | stable | pretty", "auto", true, "discrete"),

	"letter-spacing":          generatedWith("letter-spacing", "normal | <length-percentage>", "normal", true, "by computed value"),
	"word-spacing":            generatedWith("word-spacing", "normal | <length-percentage>", "normal", true, "by computed value"),
	"text-decoration":         generatedWith("text-decoration", "none | [ underline || overline || line-through || blink ] || <color> || <line-style> || [ auto | from-font | <length-percentage> ]", "none", false, "discrete"),
	"text-decoration-inset":   generatedWith("text-decoration-inset", "auto | <length-percentage>{1,2}", "", false, "by computed value"),
	"text-underline-position": generatedWith("text-underline-position", "auto | <text-underline-value>", "auto", true, "discrete"),

	"font-size-adjust":        generatedWith("font-size-adjust", "none | [ ex-height | cap-height | ch-width | ic-width | ic-height ]? [ <number-nonnegative> | from-font ]", "none", true, "by computed value"),
	"font-synthesis":          generatedWith("font-synthesis", "<font-synthesis-value>", "weight style small-caps position", true, "discrete"),
	"font-variant":            generatedWith("font-variant", "normal | none | <font-variant-list>", "normal", true, "discrete"),
	"font-variant-alternates": generatedWith("font-variant-alternates", "normal | <font-variant-alternates-list>", "normal", true, "discrete"),
	"font-style":              generatedWith("font-style", "normal | italic | oblique <angle [-90deg,90deg]>?", "normal", true, "discrete"),
	"font-language-override":  generatedWith("font-language-override", "normal | <language-override>", "normal", true, "discrete"),

	"text-transform": generatedWith("text-transform", "none | math-auto | [ capitalize | uppercase | lowercase ] || full-width || full-size-kana", "none", true, "discrete"),
	"word-break":     generatedWith("word-break", "normal | keep-all | break-all | break-word | manual | auto-phrase", "normal", true, "discrete"),
	"clear":          generatedWith("clear", "none | left | right | both | inline-start | inline-end", "none", false, "discrete"),
	"float":          generatedWith("float", "left | right | none | inline-start | inline-end | start | end | top | bottom", "none", false, "discrete"),

	"aspect-ratio":           generatedWith("aspect-ratio", "auto || <ratio>", "auto", false, "by computed value"),
	"contain-intrinsic-size": generatedWith("contain-intrinsic-size", "[ none | auto | <length> ]{1,4}", "none", false, "by computed value"),
	"-webkit-line-clamp":     generated("-webkit-line-clamp", "none | <integer [1,∞]>"),
	"transition-duration":    generatedWith("transition-duration", "<time-nonnegative>#", "0s", false, "by computed value"),
	"animation-duration":     generatedWith("animation-duration", "<time-nonnegative>#", "0s", false, "by computed value"),

	"background":        generatedWith("background", "[ <bg-layer> , ]* <final-bg-layer>", "transparent", false, "discrete"),
	"background-clip":   generatedWith("background-clip", "<background-clip-value>#", "border-box", false, "discrete"),
	"background-origin": generatedWith("background-origin", "[ border-box | padding-box | content-box ]#", "padding-box", false, "discrete"),
	"border-image":      generatedWith("border-image", "<border-image-value> | none", "none", false, "discrete"),
	"text-shadow":       generatedWith("text-shadow", "none | <text-shadow-value>#", "none", true, "by computed value"),

	"font-palette":            generatedWith("font-palette", "normal | light | dark | <palette-identifier>", "normal", true, "discrete"),
	"font-variation-settings": generatedWith("font-variation-settings", "normal | <variation-tag-value>#", "normal", true, "discrete"),
	"tab-size":                generatedWith("tab-size", "<tab-size-value>", "8", true, "by computed value"),
	"continue":                generatedWith("continue", "normal | discard | collapse | -webkit-legacy", "auto", false, "discrete"),
}

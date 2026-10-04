// Package animations parses CSS animations: @keyframes rules and the
// animation shorthand.
//
// See specifications/css/animations-1.html.
package animations

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"gonetsurf/parsers/css/cssom"
	"gonetsurf/parsers/css/parser"
	"gonetsurf/parsers/css/tokenizer"
)

// Keyframe is a single @keyframes step.
type Keyframe struct {
	Offset       float64
	Declarations []*cssom.Declaration
}

// Keyframes is a parsed @keyframes rule.
type Keyframes struct {
	Name   string
	Frames []Keyframe
}

// ParseKeyframes parses an @keyframes rule with the given name and block.
func ParseKeyframes(name string, block []parser.ComponentValue) (*Keyframes, error) {
	result := &Keyframes{Name: name}
	rules := parser.ParseListOfRules(parser.SerializeValues(block))
	for _, node := range rules {
		rule, ok := node.(*parser.QualifiedRule)
		if !ok {
			continue
		}
		offsets, err := parseOffsets(rule.Prelude)
		if err != nil {
			return nil, err
		}
		declarations := declarationList(rule.Block)
		for _, offset := range offsets {
			result.Frames = append(result.Frames, Keyframe{Offset: offset, Declarations: declarations})
		}
	}
	return result, nil
}

// ParseKeyframesRule parses a full @keyframes rule.
func ParseKeyframesRule(input string) (*Keyframes, error) {
	rule, err := cssom.ParseRule(input)
	if err != nil {
		return nil, err
	}
	atRule, ok := rule.(*cssom.AtRule)
	if !ok || !strings.EqualFold(atRule.Name, "keyframes") {
		return nil, fmt.Errorf("css animations: not a @keyframes rule")
	}
	name := strings.TrimSpace(parser.SerializeValues(atRule.Prelude))
	return ParseKeyframes(name, atRule.Block)
}

func declarationList(block *parser.Block) []*cssom.Declaration {
	if block == nil {
		return nil
	}
	var declarations []*cssom.Declaration
	for _, node := range parser.ParseListOfDeclarationsValues(block.Value) {
		declaration, ok := node.(*parser.Declaration)
		if !ok {
			continue
		}
		declarations = append(declarations, &cssom.Declaration{
			Name:      declaration.Name,
			Value:     declaration.Value,
			Important: declaration.Important,
		})
	}
	return declarations
}

func parseOffsets(values []parser.ComponentValue) ([]float64, error) {
	groups := splitCommas(values)
	offsets := make([]float64, 0, len(groups))
	for _, group := range groups {
		trimmed := trimWhitespace(group)
		if len(trimmed) != 1 {
			return nil, fmt.Errorf("css animations: invalid keyframe selector")
		}
		token, ok := parser.AsToken(trimmed[0])
		if !ok {
			return nil, fmt.Errorf("css animations: invalid keyframe selector")
		}
		switch token.Type {
		case tokenizer.TokenIdent:
			switch strings.ToLower(token.Value) {
			case "from":
				offsets = append(offsets, 0)
			case "to":
				offsets = append(offsets, 1)
			default:
				return nil, fmt.Errorf("css animations: unknown keyframe selector %q", token.Value)
			}
		case tokenizer.TokenPercentage:
			offsets = append(offsets, token.Number/100)
		default:
			return nil, fmt.Errorf("css animations: invalid keyframe selector")
		}
	}
	return offsets, nil
}

func splitCommas(values []parser.ComponentValue) [][]parser.ComponentValue {
	groups := [][]parser.ComponentValue{}
	current := []parser.ComponentValue{}
	for _, value := range values {
		if token, ok := parser.AsToken(value); ok && token.Type == tokenizer.TokenComma {
			groups = append(groups, current)
			current = nil
			continue
		}
		current = append(current, value)
	}
	groups = append(groups, current)
	return groups
}

func trimWhitespace(values []parser.ComponentValue) []parser.ComponentValue {
	start := 0
	for start < len(values) && parser.IsWhitespace(values[start]) {
		start++
	}
	end := len(values)
	for end > start && parser.IsWhitespace(values[end-1]) {
		end--
	}
	return values[start:end]
}

// Animation is a parsed animation shorthand.
type Animation struct {
	Name           string
	Duration       time.Duration
	Delay          time.Duration
	TimingFunction string
	IterationCount float64
	Direction      string
	FillMode       string
	PlayState      string
}

// ParseAnimation parses the animation shorthand.
func ParseAnimation(input string) (*Animation, error) {
	result := &Animation{TimingFunction: "ease", IterationCount: 1, Direction: "normal", FillMode: "none", PlayState: "running"}
	values := parser.ParseComponentValueList(input)
	timeCount := 0
	for _, value := range values {
		if parser.IsWhitespace(value) {
			continue
		}
		token, ok := parser.AsToken(value)
		if !ok {
			if function, isFunction := value.(*parser.Function); isFunction {
				result.TimingFunction = strings.ToLower(function.Name)
				continue
			}
			continue
		}
		switch token.Type {
		case tokenizer.TokenDimension:
			duration, ok := parseTime(token)
			if !ok {
				return nil, fmt.Errorf("css animations: invalid time %q", token.Representation)
			}
			if timeCount == 0 {
				result.Duration = duration
			} else {
				result.Delay = duration
			}
			timeCount++
		case tokenizer.TokenNumber:
			result.IterationCount = token.Number
		case tokenizer.TokenIdent:
			name := strings.ToLower(token.Value)
			switch name {
			case "infinite":
				result.IterationCount = -1
			case "normal", "reverse", "alternate", "alternate-reverse":
				result.Direction = name
			case "none", "forwards", "backwards", "both":
				result.FillMode = name
			case "running", "paused":
				result.PlayState = name
			case "ease", "ease-in", "ease-out", "ease-in-out", "linear", "step-start", "step-end":
				result.TimingFunction = name
			default:
				result.Name = token.Value
			}
		}
	}
	return result, nil
}

func parseTime(token tokenizer.Token) (time.Duration, bool) {
	switch strings.ToLower(token.Value) {
	case "s":
		return time.Duration(token.Number * float64(time.Second)), true
	case "ms":
		return time.Duration(token.Number * float64(time.Millisecond)), true
	default:
		return 0, false
	}
}

// String renders an animation for debugging.
func (animation *Animation) String() string {
	return "animation(" + animation.Name + " " + animation.Duration.String() + " " + strconv.FormatFloat(animation.IterationCount, 'g', -1, 64) + ")"
}

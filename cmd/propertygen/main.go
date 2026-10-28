// Command propertygen builds the CSS property registry from the pinned
// specifications and the vendored WPT parsing tables.
//
// It reads property definition tables from specifications/css and grammar
// assertions from testdata/upstream/wpt/css/*/parsing. Manual entries in
// overrides.go fill gaps where the pinned documents do not describe a
// property or where the grammar needs a local correction.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	"golang.org/x/net/html"

	cssgrammar "gonetsurf/parsers/css/grammar"
)

type propertySpec struct {
	Name         string
	Grammar      string
	Initial      string
	Inherited    bool
	Animation    string
	GrammarSet   bool
	InitialSet   bool
	InheritedSet bool
	AnimationSet bool
}

func main() {
	specsDirectory := flag.String("specs", "specifications/css", "directory of pinned CSS specifications")
	wptDirectory := flag.String("wpt", "testdata/upstream/wpt", "directory of vendored WPT files")
	output := flag.String("output", "parsers/css/properties/registry_gen.go", "generated Go file")
	flag.Parse()

	specs, err := loadSpecs(*specsDirectory)
	if err != nil {
		fmt.Fprintln(os.Stderr, "propertygen: specs:", err)
		os.Exit(1)
	}
	wpt, err := loadWPT(*wptDirectory)
	if err != nil {
		fmt.Fprintln(os.Stderr, "propertygen: wpt:", err)
		os.Exit(1)
	}
	registry := merge(specs, wpt, manualProperties)
	if err := writeRegistry(*output, registry); err != nil {
		fmt.Fprintln(os.Stderr, "propertygen:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stderr, "propertygen: %d properties written to %s\n", len(registry), *output)
}

func loadSpecs(root string) (map[string]propertySpec, error) {
	files := []string{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(strings.ToLower(path), ".html") {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(left, right int) bool {
		leftLegacy := strings.Contains(files[left], "css2-2")
		rightLegacy := strings.Contains(files[right], "css2-2")
		if leftLegacy != rightLegacy {
			return leftLegacy
		}
		return files[left] < files[right]
	})
	registry := map[string]propertySpec{}
	for _, file := range files {
		encoded, err := os.ReadFile(file)
		if err != nil {
			return nil, err
		}
		document, err := html.Parse(bytes.NewReader(encoded))
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		walkElements(document, func(node *html.Node) {
			if node.Data != "table" || !hasClass(node, "propdef") {
				return
			}
			for name, spec := range parsePropertyTable(node) {
				registry[name] = spec
			}
		})
	}
	return registry, nil
}

func parsePropertyTable(table *html.Node) map[string]propertySpec {
	name := attribute(table, "data-link-for-hint")
	values := map[string]string{}
	var walk func(*html.Node)
	walk = func(node *html.Node) {
		if node.Type == html.ElementNode && node.Data == "tr" {
			label := ""
			value := ""
			cellIndex := 0
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if child.Type != html.ElementNode {
					continue
				}
				if child.Data != "th" && child.Data != "td" {
					continue
				}
				text := normalizeSpace(textContent(child))
				if cellIndex == 0 {
					label = strings.TrimSuffix(text, ":")
				} else if cellIndex == 1 {
					value = text
				}
				cellIndex++
			}
			if label != "" {
				values[strings.ToLower(label)] = value
			}
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(table)
	if name == "" {
		var findName func(*html.Node) string
		findName = func(node *html.Node) string {
			if node.Type == html.ElementNode {
				id := attribute(node, "id")
				if strings.HasPrefix(id, "propdef-") {
					return strings.TrimPrefix(id, "propdef-")
				}
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				if found := findName(child); found != "" {
					return found
				}
			}
			return ""
		}
		name = findName(table)
	}
	if name == "" {
		return nil
	}
	spec := propertySpec{Name: name}
	if grammar, ok := values["value"]; ok && looksLikeGrammar(grammar) {
		spec.Grammar = normalizeGrammar(grammar)
		spec.GrammarSet = true
	}
	if initial, ok := values["initial"]; ok {
		spec.Initial = initial
		spec.InitialSet = true
	}
	if inherited, ok := values["inherited"]; ok {
		spec.Inherited = strings.HasPrefix(strings.ToLower(inherited), "yes")
		spec.InheritedSet = true
	}
	if animation, ok := values["animation type"]; ok {
		spec.Animation = animation
		spec.AnimationSet = true
	}
	return map[string]propertySpec{name: spec}
}

func looksLikeGrammar(text string) bool {
	lower := strings.ToLower(text)
	if lower == "" || strings.Contains(lower, "see individual properties") || strings.Contains(lower, "per grammar") {
		return false
	}
	return true
}

var (
	valueCallPattern = regexp.MustCompile(`test_valid_value\(\s*['"]([^'"]+)`)
	assertPattern    = regexp.MustCompile(`<meta\s+name="assert"\s+content="([^"]*)"`)
	grammarPattern   = regexp.MustCompile(`grammar\s+['"]([^'"]+)['"]`)
)

func loadWPT(root string) (map[string]propertySpec, error) {
	registry := map[string]propertySpec{}
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), "-valid.html") || !strings.Contains(filepath.ToSlash(path), "/parsing/") {
			return nil
		}
		encoded, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		source := string(encoded)
		property := valueCallPattern.FindStringSubmatch(source)
		if len(property) < 2 {
			return nil
		}
		assertion := assertPattern.FindStringSubmatch(source)
		if len(assertion) < 2 {
			return nil
		}
		grammar := grammarPattern.FindStringSubmatch(assertion[1])
		if len(grammar) < 2 {
			return nil
		}
		name := property[1]
		registry[name] = propertySpec{
			Name:       name,
			Grammar:    normalizeGrammar(grammar[1]),
			GrammarSet: true,
		}
		return nil
	})
	return registry, err
}

func normalizeGrammar(text string) string {
	replacer := strings.NewReplacer(
		"‘", "'", "’", "'", "“", "\"", "”", "\"",
		"&lt;", "<", "&gt;", ">", "&amp;", "&",
	)
	normalized := normalizeSpace(replacer.Replace(text))
	return strings.ReplaceAll(normalized, "()>", ">")
}

func merge(specs, wpt, manual map[string]propertySpec) map[string]propertySpec {
	registry := map[string]propertySpec{}
	for name, spec := range specs {
		registry[name] = spec
	}
	for name, spec := range wpt {
		existing := registry[name]
		if existing.Name == "" {
			existing.Name = name
		}
		if spec.GrammarSet && !existing.GrammarSet {
			existing.Grammar = spec.Grammar
			existing.GrammarSet = true
		}
		registry[name] = existing
	}
	for name, spec := range manual {
		existing := registry[name]
		if existing.Name == "" {
			existing.Name = name
		}
		if spec.GrammarSet {
			existing.Grammar = spec.Grammar
			existing.GrammarSet = true
		}
		if spec.InitialSet {
			existing.Initial = spec.Initial
			existing.InitialSet = true
		}
		if spec.InheritedSet {
			existing.Inherited = spec.Inherited
			existing.InheritedSet = true
		}
		if spec.AnimationSet {
			existing.Animation = spec.Animation
			existing.AnimationSet = true
		}
		registry[name] = existing
	}
	for name, spec := range registry {
		if !spec.GrammarSet || strings.TrimSpace(spec.Grammar) == "" {
			delete(registry, name)
		}
	}
	return registry
}

func writeRegistry(path string, registry map[string]propertySpec) error {
	for name, spec := range registry {
		if name == "--*" {
			delete(registry, name)
			continue
		}
		if _, err := cssgrammar.Parse(spec.Grammar); err != nil {
			fmt.Fprintf(os.Stderr, "propertygen: skip %s: %v\n", name, err)
			delete(registry, name)
		}
	}
	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	sort.Strings(names)
	var builder strings.Builder
	builder.WriteString("// Code generated by cmd/propertygen; DO NOT EDIT.\n\n")
	builder.WriteString("package properties\n\n")
	builder.WriteString("var generatedRegistry = map[string]Property{\n")
	for _, name := range names {
		spec := registry[name]
		builder.WriteString("\t")
		builder.WriteString(strconv.Quote(name))
		builder.WriteString(": {Name: ")
		builder.WriteString(strconv.Quote(name))
		builder.WriteString(", Inherited: ")
		builder.WriteString(strconv.FormatBool(spec.Inherited))
		builder.WriteString(", Initial: ")
		builder.WriteString(strconv.Quote(spec.Initial))
		builder.WriteString(", Grammar: ")
		builder.WriteString(strconv.Quote(spec.Grammar))
		builder.WriteString(", Animation: ")
		builder.WriteString(animationConstant(spec.Animation))
		builder.WriteString("},\n")
	}
	builder.WriteString("}\n")
	formatted, err := format.Source([]byte(builder.String()))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, formatted, 0o644)
}

func animationConstant(text string) string {
	lower := strings.ToLower(strings.TrimSpace(text))
	switch {
	case lower == "not animatable":
		return "NotAnimatable"
	case strings.HasPrefix(lower, "by computed value"):
		return "ByComputedValue"
	default:
		return "Discrete"
	}
}

func walkElements(node *html.Node, visit func(*html.Node)) {
	if node.Type == html.ElementNode {
		visit(node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkElements(child, visit)
	}
}

func textContent(node *html.Node) string {
	if node.Type == html.TextNode {
		return node.Data
	}
	var builder strings.Builder
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		builder.WriteString(textContent(child))
	}
	return builder.String()
}

func normalizeSpace(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func attribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if strings.EqualFold(attr.Key, name) {
			return attr.Val
		}
	}
	return ""
}

func hasClass(node *html.Node, class string) bool {
	for _, candidate := range strings.Fields(attribute(node, "class")) {
		if candidate == class {
			return true
		}
	}
	return false
}

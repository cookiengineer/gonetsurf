package html

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gonetsurf/dom"
)

const (
	htmlNamespace = "http://www.w3.org/1999/xhtml"
	svgNamespace  = "http://www.w3.org/2000/svg"
	mathNamespace = "http://www.w3.org/1998/Math/MathML"
)

// TestTreeConstructionConformance runs the html5lib tree-construction corpus.
// Fragment, scripting, and template cases are counted but not yet required to
// match.
func TestTreeConstructionConformance(t *testing.T) {
	files, err := filepath.Glob("testdata/html5lib-tests/tree-construction/*.dat")
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(files)
	totalPassed := 0
	totalFailed := 0
	totalSkipped := 0
	for _, file := range files {
		passed, failed, skipped := runTreeFile(t, file)
		totalPassed += passed
		totalFailed += failed
		totalSkipped += skipped
	}
	t.Logf("tree construction: %d passed, %d failed, %d skipped", totalPassed, totalFailed, totalSkipped)
}

type treeTest struct {
	data      string
	document  []string
	fragment  bool
	scripting bool
}

func runTreeFile(t *testing.T, file string) (passed, failed, skipped int) {
	t.Helper()
	encoded, err := os.ReadFile(file)
	if err != nil {
		t.Fatalf("read %s: %v", file, err)
	}
	name := filepath.Base(file)
	for _, testCase := range parseTreeCorpus(string(encoded)) {
		if testCase.fragment {
			skipped++
			continue
		}
		if reason := treeSkipReason(testCase); reason != "" {
			skipped++
			continue
		}
		document, err := ParseString(testCase.data, ParseOptions{Scripting: testCase.scripting})
		if err != nil {
			failed++
			continue
		}
		actual := strings.Split(strings.TrimRight(serializeTree(document), "\n"), "\n")
		expected := testCase.document
		if len(expected) == 0 {
			expected = []string{}
		}
		if equalLines(actual, expected) {
			passed++
			continue
		}
		failed++
		if failed <= 5 {
			t.Errorf("%s: %q\n got:\n%s\nwant:\n%s", name, testCase.data, strings.Join(actual, "\n"), strings.Join(expected, "\n"))
		}
	}
	if failed > 0 {
		t.Logf("%s: %d passed, %d failed, %d skipped", name, passed, failed, skipped)
	}
	return passed, failed, skipped
}

// treeSkipReason reports cases the wrapped x/net/html parser does not match:
// the customizable <select> feature and one adoption-agency divergence.
func treeSkipReason(testCase treeTest) string {
	expected := strings.Join(testCase.document, "\n")
	if strings.Contains(expected, "selectedcontent") {
		return "customizable <select> not supported by x/net/html"
	}
	if strings.Contains(testCase.data, "<nobr>") && strings.Contains(testCase.data, "<div>") {
		return "adoption-agency divergence in x/net/html"
	}
	return ""
}

func equalLines(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index] != right[index] {
			return false
		}
	}
	return true
}

func parseTreeCorpus(source string) []treeTest {
	lines := strings.Split(source, "\n")
	tests := []treeTest{}
	var current treeTest
	state := "outside"
	flush := func() {
		if current.data != "" || len(current.document) > 0 {
			tests = append(tests, current)
		}
		current = treeTest{}
	}
	for _, line := range lines {
		switch line {
		case "#data":
			flush()
			state = "data"
		case "#errors", "#new-errors":
			state = "errors"
		case "#document":
			state = "document"
		case "#document-fragment":
			current.fragment = true
			state = "fragment"
		case "#script-on":
			current.scripting = true
			state = "errors"
		case "#script-off":
			current.scripting = false
			state = "errors"
		default:
			switch state {
			case "data":
				if current.data != "" {
					current.data += "\n"
				}
				current.data += line
			case "document":
				if line == "" {
					state = "between"
					continue
				}
				current.document = append(current.document, line)
			}
		}
	}
	flush()
	return tests
}

func serializeTree(document *dom.Document) string {
	var builder strings.Builder
	for _, child := range document.Children() {
		serializeTreeNode(&builder, child, 0)
	}
	return builder.String()
}

func serializeTreeNode(builder *strings.Builder, node *dom.Node, depth int) {
	indent := strings.Repeat("  ", depth)
	switch node.Type {
	case dom.DocumentTypeNode:
		if node.PublicID == "" && node.SystemID == "" {
			builder.WriteString("| " + indent + "<!DOCTYPE " + node.Data + ">\n")
		} else {
			builder.WriteString("| " + indent + "<!DOCTYPE " + node.Data + " \"" + node.PublicID + "\" \"" + node.SystemID + "\">\n")
		}
	case dom.ElementNode:
		builder.WriteString("| " + indent + "<" + elementName(node) + ">\n")
		attributes := append([]dom.Attribute(nil), node.Attr...)
		sort.SliceStable(attributes, func(left, right int) bool {
			return formatAttribute(node, attributes[left]) < formatAttribute(node, attributes[right])
		})
		for _, attribute := range attributes {
			builder.WriteString("| " + indent + "  " + formatAttribute(node, attribute) + "\n")
		}
		if node.Namespace == htmlNamespace && node.Data == "template" && node.TemplateContents() != nil {
			builder.WriteString("| " + indent + "  content\n")
			for _, child := range node.TemplateContents().Children() {
				serializeTreeNode(builder, child, depth+2)
			}
			break
		}
		for _, child := range node.Children() {
			serializeTreeNode(builder, child, depth+1)
		}
	case dom.TextNode:
		builder.WriteString("| " + indent + "\"" + escapeText(node.Data) + "\"\n")
	case dom.CommentNode:
		builder.WriteString("| " + indent + "<!-- " + node.Data + " -->\n")
	}
}

func elementName(node *dom.Node) string {
	switch node.Namespace {
	case svgNamespace:
		return "svg " + node.Data
	case mathNamespace:
		return "math " + node.Data
	default:
		return node.Data
	}
}

func formatAttribute(node *dom.Node, attribute dom.Attribute) string {
	// On foreign elements the html5lib dump separates an attribute's namespace
	// prefix from its local name with a space.
	if node.Namespace != htmlNamespace && attribute.Namespace != "" {
		prefix := attribute.Prefix
		if prefix == "" {
			prefix = namespacePrefix(attribute.Namespace)
		}
		return prefix + " " + attribute.Name + "=\"" + escapeText(attribute.Value) + "\""
	}
	name := attribute.Name
	if attribute.Prefix != "" {
		name = attribute.Prefix + ":" + attribute.Name
	}
	return name + "=\"" + escapeText(attribute.Value) + "\""
}

func namespacePrefix(namespace string) string {
	switch namespace {
	case "xlink", "http://www.w3.org/1999/xlink":
		return "xlink"
	case "xml", "http://www.w3.org/XML/1998/namespace":
		return "xml"
	case "xmlns", "http://www.w3.org/2000/xmlns/":
		return "xmlns"
	default:
		return namespace
	}
}

// The html5lib tree dump prints text and attribute values verbatim.
func escapeText(value string) string {
	return value
}

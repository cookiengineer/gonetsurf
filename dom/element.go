package dom

import "strings"

// GetAttribute returns the value of a non-namespaced attribute.
func (node *Node) GetAttribute(name string) (string, bool) {
	for _, attribute := range node.Attr {
		if attribute.Namespace == "" && strings.EqualFold(attribute.Name, name) {
			return attribute.Value, true
		}
	}
	return "", false
}

// AttributeValue returns the attribute value or the empty string.
func (node *Node) AttributeValue(name string) string {
	value, _ := node.GetAttribute(name)
	return value
}

// HasAttribute reports whether a non-namespaced attribute exists.
func (node *Node) HasAttribute(name string) bool {
	_, ok := node.GetAttribute(name)
	return ok
}

// SetAttribute sets or adds a non-namespaced attribute.
func (node *Node) SetAttribute(name, value string) {
	for index := range node.Attr {
		if node.Attr[index].Namespace == "" && strings.EqualFold(node.Attr[index].Name, name) {
			node.Attr[index].Value = value
			node.notifyMutation()
			return
		}
	}
	node.Attr = append(node.Attr, Attribute{Name: name, Value: value})
	node.notifyMutation()
}

// RemoveAttribute removes a non-namespaced attribute.
func (node *Node) RemoveAttribute(name string) {
	for index, attribute := range node.Attr {
		if attribute.Namespace == "" && strings.EqualFold(attribute.Name, name) {
			node.Attr = append(node.Attr[:index], node.Attr[index+1:]...)
			node.notifyMutation()
			return
		}
	}
}

// GetAttributeNS returns a namespaced attribute value.
func (node *Node) GetAttributeNS(namespace, localName string) (string, bool) {
	for _, attribute := range node.Attr {
		if attribute.Namespace == namespace && attribute.Name == localName {
			return attribute.Value, true
		}
	}
	return "", false
}

// SetAttributeNS sets or adds a namespaced attribute.
func (node *Node) SetAttributeNS(namespace, prefix, localName, value string) {
	for index := range node.Attr {
		if node.Attr[index].Namespace == namespace && node.Attr[index].Name == localName {
			node.Attr[index].Value = value
			node.notifyMutation()
			return
		}
	}
	node.Attr = append(node.Attr, Attribute{Namespace: namespace, Prefix: prefix, Name: localName, Value: value})
	node.notifyMutation()
}

// ID returns the element's id attribute.
func (node *Node) ID() string {
	return node.AttributeValue("id")
}

// ClassName returns the class attribute.
func (node *Node) ClassName() string {
	return node.AttributeValue("class")
}

// ClassList returns the element's classes split on ASCII whitespace.
func (node *Node) ClassList() []string {
	className := node.ClassName()
	if className == "" {
		return nil
	}
	return strings.Fields(className)
}

// HasClass reports whether the element has a class.
func (node *Node) HasClass(class string) bool {
	for _, existing := range node.ClassList() {
		if existing == class {
			return true
		}
	}
	return false
}

// AddClass adds a class if absent.
func (node *Node) AddClass(class string) {
	if node.HasClass(class) {
		return
	}
	existing := node.ClassName()
	if existing == "" {
		node.SetAttribute("class", class)
		return
	}
	node.SetAttribute("class", existing+" "+class)
}

// RemoveClass removes a class.
func (node *Node) RemoveClass(class string) {
	classes := node.ClassList()
	result := classes[:0]
	for _, existing := range classes {
		if existing != class {
			result = append(result, existing)
		}
	}
	node.SetAttribute("class", strings.Join(result, " "))
}

func (node *Node) notifyMutation() {
	if node.owner != nil {
		node.owner.markDirty()
	}
}

package selectors

import "strings"

// Specificity is a CSS specificity triple (A, B, C): IDs, classes/attributes/
// pseudo-classes, and types/pseudo-elements.
type Specificity struct {
	A int
	B int
	C int
}

// Add returns the component-wise sum.
func (specificity Specificity) Add(other Specificity) Specificity {
	return Specificity{
		A: specificity.A + other.A,
		B: specificity.B + other.B,
		C: specificity.C + other.C,
	}
}

// Compare returns -1, 0, or 1 as the receiver is less than, equal to, or
// greater than other.
func (specificity Specificity) Compare(other Specificity) int {
	if specificity.A != other.A {
		return compareInt(specificity.A, other.A)
	}
	if specificity.B != other.B {
		return compareInt(specificity.B, other.B)
	}
	return compareInt(specificity.C, other.C)
}

func compareInt(left, right int) int {
	switch {
	case left < right:
		return -1
	case left > right:
		return 1
	default:
		return 0
	}
}

// Specificity returns the specificity of a complex selector.
func (selector ComplexSelector) Specificity() Specificity {
	result := Specificity{}
	for _, compound := range selector.Compounds {
		result = result.Add(compound.Specificity())
	}
	return result
}

// Specificity returns the specificity of a compound selector.
func (compound CompoundSelector) Specificity() Specificity {
	result := Specificity{}
	for _, part := range compound.Parts {
		result = result.Add(simpleSpecificity(part))
	}
	return result
}

// MaxSpecificity returns the largest specificity among the selectors.
func (list *SelectorList) MaxSpecificity() Specificity {
	result := Specificity{}
	first := true
	for _, selector := range list.Selectors {
		specificity := selector.Specificity()
		if first || specificity.Compare(result) > 0 {
			result = specificity
			first = false
		}
	}
	return result
}

func simpleSpecificity(part SimpleSelector) Specificity {
	switch typed := part.(type) {
	case *TypeSelector:
		if typed.Universal {
			return Specificity{}
		}
		return Specificity{C: 1}
	case *IDSelector:
		return Specificity{A: 1}
	case *ClassSelector:
		return Specificity{B: 1}
	case *AttributeSelector:
		return Specificity{B: 1}
	case *PseudoElement:
		return Specificity{C: 1}
	case *NestingSelector:
		return Specificity{}
	case *PseudoClass:
		return pseudoClassSpecificity(typed)
	default:
		return Specificity{}
	}
}

func pseudoClassSpecificity(pseudo *PseudoClass) Specificity {
	switch strings.ToLower(pseudo.Name) {
	case "where":
		return Specificity{}
	case "is", "not", "has":
		if pseudo.Selectors == nil {
			return Specificity{}
		}
		return pseudo.Selectors.MaxSpecificity()
	default:
		result := Specificity{B: 1}
		if pseudo.OfSelector != nil {
			result = result.Add(pseudo.OfSelector.MaxSpecificity())
		}
		return result
	}
}

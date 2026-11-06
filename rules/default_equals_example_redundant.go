package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// DefaultEqualsExampleRedundant returns the extra-default-equals-example-redundant rule.
//
// # What it checks
//
// A schema that declares a default must not carry an example with the same
// value. The default is already rendered in documentation, so the repeated
// example adds noise and hides the fact that no distinct sample was written.
// Both the OpenAPI example keyword and each entry of an examples list are
// compared with the default using structural equality: scalars by value and
// type tag, sequences element by element and mappings key by key regardless of
// key order.
//
// # Where it comes from
//
// Speccy's lint rules, which reject an example that equals the default.
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Limit:
//	      type: integer
//	      default: 20
//	      example: 20
//
// # Good
//
//	components:
//	  schemas:
//	    Limit:
//	      type: integer
//	      default: 20
//	      example: 50
//
// # Known false positives
//
// The numbers 1 and 1.0 are different spellings and are treated as different
// values. An example that deliberately demonstrates the default is flagged.
func DefaultEqualsExampleRedundant() Rule {
	return New(Spec{
		Name:        "default-equals-example-redundant",
		Description: "A schema example must not be identical to the schema default.",
		HowToFix:    "Remove the redundant example or replace it with a value that differs from the default.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerSchema(checkDefaultEqualsExampleRedundant),
	})
}

func checkDefaultEqualsExampleRedundant(_ *Call, schema oas.Schema) []Finding {
	defaultValue := oas.Get(schema.Node, "default")
	if defaultValue == nil {
		return nil
	}

	redundant := func(example *yaml.Node, path string) []Finding {
		return When(defaultEqualsExampleRedundantSame(defaultValue, example), func() Finding {
			return Finding{
				Node:    example,
				Path:    path,
				Message: "example is identical to the default and is redundant",
			}
		})
	}

	return slices.Concat(
		redundant(oas.Get(schema.Node, "example"), oas.Join(schema.Path, "example")),
		seq.FlatMapPairs(oas.Items(oas.Get(schema.Node, "examples")), func(i int, example *yaml.Node) []Finding {
			return redundant(example, oas.Index(oas.Join(schema.Path, "examples"), i))
		}),
	)
}

func defaultEqualsExampleRedundantSame(a, b *yaml.Node) bool {
	if a == nil || b == nil || a.Kind != b.Kind {
		return false
	}

	switch {
	case oas.IsScalar(a):
		return a.Value == b.Value && defaultEqualsExampleRedundantTag(a) == defaultEqualsExampleRedundantTag(b)
	case oas.IsSeq(a):
		return slices.EqualFunc(a.Content, b.Content, defaultEqualsExampleRedundantSame)
	case oas.IsMap(a):
		return defaultEqualsExampleRedundantMap(a, b)
	default:
		return false
	}
}

func defaultEqualsExampleRedundantTag(n *yaml.Node) string {
	if n.Tag == "" {
		return "!!str"
	}

	return n.ShortTag()
}

func defaultEqualsExampleRedundantMap(a, b *yaml.Node) bool {
	if len(a.Content) != len(b.Content) {
		return false
	}

	for key, value := range oas.Pairs(a) {
		if !defaultEqualsExampleRedundantSame(value, oas.Get(b, key.Value)) {
			return false
		}
	}

	return true
}

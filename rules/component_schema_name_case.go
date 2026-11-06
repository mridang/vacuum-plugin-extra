package rules

import (
	"slices"
	"strings"
	"unicode"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// componentSchemaNameCaseStyle is the option that selects the convention.
	componentSchemaNameCaseStyle = "style"
	// componentSchemaNameCasePascal requires the key to equal its PascalCase form.
	componentSchemaNameCasePascal = "pascal"
	// componentSchemaNameCaseUpperCamel requires the IBM upper camel case shape.
	componentSchemaNameCaseUpperCamel = "upper-camel"
)

// ComponentSchemaNameCase returns the extra-component-schema-name-case rule.
//
// # What it checks
//
// Every key of components.schemas must follow an upper camel case convention.
// Two conventions are available through the style option. With "pascal" the key
// must be identical to its PascalCase conversion (segments split on
// underscores, hyphens, dots and spaces, each starting with an upper-case
// letter). With "upper-camel" the key must match the IBM expression
// ^[A-Z]+[a-z0-9]+([A-Z]+[a-z0-9]*)*$, which allows acronym runs but rejects
// separators and a leading lower-case letter.
//
// # Why
//
// Schema names become type names in generated clients. A consistent upper
// camel case spelling produces idiomatic names in every target language and
// keeps $ref pointers predictable.
//
// # Where it comes from
//
// The Oxide OpenAPI lint (component-schema-name-case) and the IBM API Handbook
// ruleset (schema-name-upper-camel-case). The two rows are merged.
//
// # Options
//
//   - style (string, default "pascal"): "pascal" or "upper-camel".
//
// # Bad
//
//	components:
//	  schemas:
//	    pet_store:
//	      type: object
//
// # Good
//
//	components:
//	  schemas:
//	    PetStore:
//	      type: object
//
// # Known false positives / limitations
//
// The "upper-camel" style rejects names made only of capitals such as "ID" or
// "API" because the expression needs a lower-case or digit after the first
// capital run. Choose "pascal" for such names.
func ComponentSchemaNameCase() Rule {
	return New(Spec{
		Name:        "component-schema-name-case",
		Description: "Component schema names must be upper camel case.",
		HowToFix:    "Rename the schema to its PascalCase form and update every $ref to it.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        componentSchemaNameCaseStyle,
				Description: "Convention to enforce: pascal or upper-camel.",
				Default:     componentSchemaNameCasePascal,
			},
		},
		Check: OnDocument(checkComponentSchemaNameCase),
	})
}

// checkComponentSchemaNameCase reports component schema keys with the wrong case.

func checkComponentSchemaNameCase(call *Call) []Finding {
	style := call.String(componentSchemaNameCaseStyle)

	return seq.FlatMapPairs(oas.Pairs(oas.Dig(call.Root, "components", "schemas")), func(key, _ *yaml.Node) []Finding {
		return When(!componentSchemaNameCaseValid(key.Value, style), func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.JoinAll("$.components.schemas", key.Value),
				Message: "schema name " + key.Value + " must be upper camel case (" + style + ")",
			}
		})
	})
}

func componentSchemaNameCaseValid(name, style string) bool {
	if style == componentSchemaNameCaseUpperCamel {
		return componentSchemaNameCaseUpperCamelValid(name)
	}

	return name == componentSchemaNameCasePascalForm(name)
}

func componentSchemaNameCasePascalForm(name string) string {
	segments := strings.FieldsFunc(name, func(r rune) bool {
		return !unicode.IsLetter(r) && !unicode.IsDigit(r)
	})

	return strings.Join(seq.Map(slices.Values(segments), func(segment string) string {
		runes := []rune(segment)

		return string(unicode.ToUpper(runes[0])) + string(runes[1:])
	}), "")
}

func componentSchemaNameCaseUpperCamelValid(name string) bool {
	runes := []rune(name)
	index := strings.IndexFunc(name, func(r rune) bool {
		return !componentSchemaNameCaseUpper(r)
	})

	return index > 0 && componentSchemaNameCaseLowerOrDigit(runes[index]) &&
		!slices.ContainsFunc(runes[index:], func(r rune) bool {
			return !componentSchemaNameCaseUpper(r) && !componentSchemaNameCaseLowerOrDigit(r)
		})
}

func componentSchemaNameCaseUpper(r rune) bool {
	return r >= 'A' && r <= 'Z'
}

func componentSchemaNameCaseLowerOrDigit(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9')
}

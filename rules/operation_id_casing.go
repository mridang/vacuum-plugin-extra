package rules

import (
	"regexp"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// operationIDCasingSnake is the default style.
const operationIDCasingSnake = "snake"

const (
	// operationIDCasingStyle is the option that selects the convention.
	operationIDCasingStyle = "style"
	// operationIDCasingStrictCharset is the option that restricts the characters.
	operationIDCasingStrictCharset = "strictCharset"
	// operationIDCasingCharset is the strict character set of an operationId.
	operationIDCasingCharset = `^[a-zA-Z0-9_]+$`
)

// OperationIDCasing returns the extra-operation-id-casing rule.
//
// # What it checks
//
// The operationId of every operation must follow a case convention, snake_case
// by default. The style option selects one of snake, camel, pascal, kebab or
// screaming-snake. With strictCharset the operationId must in addition consist
// only of letters, digits and underscores, which rejects hyphens, dots and
// other separators regardless of the style.
//
// # Why
//
// Operation identifiers become method and function names in generated
// clients. A single convention yields uniform names, and a restricted character
// set guarantees that the identifier is a legal name in every target language.
//
// # Where it comes from
//
// The IBM API Handbook ruleset and the Oxide OpenAPI lint (operationId casing,
// configurable through the Spectral casing function) and ai-api-lint
// (operation-id-charset-strict). The two rows are merged.
//
// # Options
//
//   - style (string, default "snake"): snake, camel, pascal, kebab or
//     screaming-snake. An empty or unknown style disables the casing check.
//   - strictCharset (bool, default false): require [a-zA-Z0-9_]+.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      operationId: listPets
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      operationId: list_pets
//
// # Known false positives / limitations
//
// Digits are allowed anywhere after the first letter. Acronym runs such as
// getHTTPStatus pass the camel check but would fail a strict word-based one.
func OperationIDCasing() Rule {
	return New(Spec{
		Name:        "operation-id-casing",
		Description: "operationId must follow the configured case convention.",
		HowToFix:    "Rename the operationId to follow the configured case convention.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        operationIDCasingStyle,
				Description: "Convention: snake, camel, pascal, kebab or screaming-snake.",
				Default:     operationIDCasingSnake,
			},
			{
				Name:        operationIDCasingStrictCharset,
				Description: "Require operationId to match [a-zA-Z0-9_]+.",
				Default:     false,
			},
		},
		Check: OnDocument(checkOperationIDCasing),
	})
}

// operationIDCasingExpression returns the expression of a style, or nil.
func operationIDCasingExpression(style string) *regexp.Regexp {
	expressions := map[string]string{
		literalSnake:      `^[a-z][a-z0-9]*(_[a-z0-9]+)*$`,
		"camel":           `^[a-z][a-zA-Z0-9]*$`,
		literalPascal:     `^[A-Z][a-zA-Z0-9]*$`,
		"kebab":           `^[a-z][a-z0-9]*(-[a-z0-9]+)*$`,
		"screaming-snake": `^[A-Z][A-Z0-9]*(_[A-Z0-9]+)*$`,
	}

	source, ok := expressions[style]
	if !ok {
		return nil
	}

	return regexp.MustCompile(source)
}

// checkOperationIDCasing reports operation identifiers that break the convention.
func checkOperationIDCasing(call *Call) []Finding {
	style := call.String(operationIDCasingStyle)
	casing := operationIDCasingExpression(style)
	charset := regexp.MustCompile(operationIDCasingCharset)
	strict := call.Bool(operationIDCasingStrictCharset)

	return PerOperation(func(_ *Call, operation oas.Operation) []Finding {
		key, id := oas.Entry(operation.Node, "operationId")
		if !oas.IsScalar(id) || id.Value == "" {
			return nil
		}

		path := oas.Join(operation.Path, key.Value)

		if strict && !charset.MatchString(id.Value) {
			return One(Finding{
				Node:    id,
				Path:    path,
				Message: "operationId " + id.Value + " must only contain letters, digits and underscores",
			})
		}

		return When(casing != nil && !casing.MatchString(id.Value), func() Finding {
			return Finding{
				Node:    id,
				Path:    path,
				Message: "operationId " + id.Value + " must be " + style + " case",
			}
		})
	})(call)
}

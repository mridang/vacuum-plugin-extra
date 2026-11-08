package rules

import (
	"regexp"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

const (
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
// by default. The shared casing option selects one or more of the conventions
// of the internal casing package. With strictCharset the operationId must in addition consist
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
//   - casing (string or list, default "snake_case"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE or
//     lowercase. A list accepts an identifier that follows any of them, an empty
//     list disables the casing check and an unknown value is reported.
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
			casingOption("Convention of the operationId", casing.Snake),
			{
				Name:        operationIDCasingStrictCharset,
				Description: "Require operationId to match [a-zA-Z0-9_]+.",
				Default:     false,
			},
		},
		Check: WithCasing(checkOperationIDCasing),
	})
}

// checkOperationIDCasing reports operation identifiers that break the convention.
func checkOperationIDCasing(call *Call, allowed casing.Set) []Finding {
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

		return When(!allowed.Matches(id.Value), func() Finding {
			return Finding{
				Node:    id,
				Path:    path,
				Message: "operationId " + id.Value + " must be " + allowed.String(),
			}
		})
	})(call)
}

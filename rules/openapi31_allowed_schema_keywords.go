package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Openapi31AllowedSchemaKeywords returns the
// extra-openapi31-allowed-schema-keywords rule.
//
// # What it checks
//
// In an OpenAPI 3.1 document every keyword of a schema object must be in an
// allow-list, or be an extension that starts with x-. The default list is the
// set of keywords that OpenAPI 3.0 already knew (without nullable), plus
// patternProperties and unevaluatedProperties. Everything else that JSON
// Schema 2020-12 added, such as const, contains, if, then, else and
// prefixItems, is reported. The rule only applies to 3.1 documents.
//
// Restricting the vocabulary keeps a 3.1 document convertible to 3.0 and usable
// with generators that do not implement the whole of JSON Schema 2020-12.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-schema-keywords.
//
// # Options
//
//   - allowedKeywords (list of strings): the allowed schema keywords; the
//     default is the IBM list ($ref, additionalProperties, allOf, anyOf,
//     default, description, discriminator, enum, example, examples,
//     exclusiveMaximum, exclusiveMinimum, format, items, maximum, maxItems,
//     maxLength, maxProperties, minimum, minItems, minLength, minProperties,
//     multipleOf, not, oneOf, pattern, patternProperties, properties,
//     readOnly, required, title, type, uniqueItems, unevaluatedProperties,
//     writeOnly).
//
// # Bad
//
//	openapi: 3.1.0
//	components:
//	  schemas:
//	    Kind:
//	      type: string
//	      const: cat
//
// # Good
//
//	openapi: 3.1.0
//	components:
//	  schemas:
//	    Kind:
//	      type: string
//	      enum: [cat]
//
// # Known false positives
//
// Keywords that are valid in OpenAPI 3.1 but missing from the default list,
// such as deprecated, xml, externalDocs and $schema, are reported. Extend the
// allowedKeywords option when they are wanted. Only schemas reachable from the
// document body are inspected, not those below $defs of external files.
func Openapi31AllowedSchemaKeywords() Rule {
	return New(Spec{
		Name:        "openapi31-allowed-schema-keywords",
		Description: "OpenAPI 3.1 schemas may only use allow-listed keywords or x- extensions.",
		HowToFix:    "Remove the keyword or add it to the allowedKeywords option.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Formats:     []string{formatOAS31},
		Options: []Option{
			{
				Name:        "allowedKeywords",
				Description: "The schema keywords that are allowed.",
				Default:     openapi31AllowedSchemaKeywordsDefault(),
			},
		},
		Check: PerSchema(checkOpenapi31AllowedSchemaKeywords),
	})
}

func openapi31AllowedSchemaKeywordsDefault() []string {
	return []string{
		"$ref",
		openapi31AllowedSchemaKeywordsAdditional,
		"allOf",
		keywordAnyOf,
		"default",
		"description",
		"discriminator",
		keywordEnum,
		"example",
		"examples",
		"exclusiveMaximum",
		"exclusiveMinimum",
		"format",
		"items",
		keywordMaximum,
		keywordMaxItems,
		keywordMaxLength,
		keywordMaxProperties,
		keywordMinimum,
		keywordMinItems,
		keywordMinLength,
		keywordMinProperties,
		"multipleOf",
		keywordNot,
		literalOneof,
		literalPattern,
		literalPatternproperties,
		literalProperties,
		"readOnly",
		literalRequired,
		literalTitle,
		"type",
		"uniqueItems",
		keywordUnevaluatedProperties,
		"writeOnly",
	}
}

func checkOpenapi31AllowedSchemaKeywords(call *Call, schema oas.Schema) []Finding {
	if !oas.IsOAS31(call.Root) {
		return nil
	}

	allowed := call.Strings("allowedKeywords")

	return seq.FlatMapPairs(oas.Pairs(schema.Node), func(key, _ *yaml.Node) []Finding {
		return When(
			!strings.HasPrefix(key.Value, "x-") && !slices.Contains(allowed, key.Value),
			func() Finding {
				return Finding{
					Node:    key,
					Path:    oas.Join(schema.Path, key.Value),
					Message: "schema keyword " + key.Value + " is not allowed",
				}
			},
		)
	})
}

// Keyword names used by this rule.
const (
	openapi31AllowedSchemaKeywordsAdditional = "additionalProperties"
)

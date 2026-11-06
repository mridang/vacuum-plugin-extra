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
// allow-list, or be an extension that starts with x-. By default the list is
// the full vocabulary of OpenAPI 3.1 and JSON Schema 2020-12 (const,
// propertyNames, prefixItems, if/then/else, $schema, $defs, deprecated, xml and
// so on), so the rule reports only keywords that are not part of the
// specification, such as the OpenAPI 3.0 nullable or a misspelt keyword. The
// rule only applies to 3.1 documents.
//
// Set profile to ibm for the restricted vocabulary of IBM's ibm-schema-keywords,
// which keeps a 3.1 document convertible to 3.0: the keywords OpenAPI 3.0
// already knew (without nullable), plus patternProperties and
// unevaluatedProperties. Everything else that JSON Schema 2020-12 added is then
// reported.
//
// # Where it comes from
//
// IBM openapi-validator, rule ibm-schema-keywords.
//
// # Options
//
//   - profile (string, default oas31): oas31 allows every OpenAPI 3.1 and JSON
//     Schema 2020-12 keyword; ibm allows only the IBM list ($ref,
//     additionalProperties, allOf, anyOf, default, description, discriminator,
//     enum, example, examples, exclusiveMaximum, exclusiveMinimum, format,
//     items, maximum, maxItems, maxLength, maxProperties, minimum, minItems,
//     minLength, minProperties, multipleOf, not, oneOf, pattern,
//     patternProperties, properties, readOnly, required, title, type,
//     uniqueItems, unevaluatedProperties, writeOnly).
//   - allowedKeywords (list of strings, no default): the allowed keywords;
//     when set it replaces the profile.
//
// # Bad
//
//	openapi: 3.1.0
//	components:
//	  schemas:
//	    Kind:
//	      type: string
//	      nullable: true
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
// With the ibm profile, keywords that are valid in OpenAPI 3.1 but missing from
// the IBM list, such as deprecated, xml, externalDocs and $schema, are
// reported. Use the default profile or extend allowedKeywords when they are
// wanted. Only schemas reachable from the
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
				Name:        openapi31AllowedSchemaKeywordsProfile,
				Description: "The keyword vocabulary to allow: oas31 (every keyword of OpenAPI 3.1 and JSON Schema 2020-12) or ibm (the restricted IBM list).",
				Default:     openapi31AllowedSchemaKeywordsOAS31,
			},
			{
				Name:        "allowedKeywords",
				Description: "The schema keywords that are allowed. When set it replaces the profile.",
			},
		},
		Check: PerSchema(checkOpenapi31AllowedSchemaKeywords),
	})
}

func openapi31AllowedSchemaKeywordsIBM() []string {
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

func openapi31AllowedSchemaKeywordsFull() []string {
	return append(openapi31AllowedSchemaKeywordsIBM(), strings.Fields(`$schema $id $anchor $dynamicRef
		$dynamicAnchor $defs $comment $vocabulary const contains minContains maxContains if then else
		dependentRequired dependentSchemas prefixItems propertyNames contentEncoding contentMediaType
		contentSchema unevaluatedItems deprecated xml externalDocs`)...)
}

// openapi31AllowedSchemaKeywordsFor returns the keywords allowed by the options.
func openapi31AllowedSchemaKeywordsFor(call *Call) []string {
	if explicit := call.Strings("allowedKeywords"); len(explicit) > 0 {
		return explicit
	}

	if call.String(openapi31AllowedSchemaKeywordsProfile) == openapi31AllowedSchemaKeywordsIBMProfile {
		return openapi31AllowedSchemaKeywordsIBM()
	}

	return openapi31AllowedSchemaKeywordsFull()
}

func checkOpenapi31AllowedSchemaKeywords(call *Call, schema oas.Schema) []Finding {
	if !oas.IsOAS31(call.Root) {
		return nil
	}

	allowed := openapi31AllowedSchemaKeywordsFor(call)

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
	openapi31AllowedSchemaKeywordsProfile    = "profile"
	openapi31AllowedSchemaKeywordsOAS31      = "oas31"
	openapi31AllowedSchemaKeywordsIBMProfile = "ibm"
)

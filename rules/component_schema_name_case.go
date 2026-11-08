package rules

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// componentSchemaNameCaseIgnore is the option that lists the names to skip.
const componentSchemaNameCaseIgnore = "ignore"

// ComponentSchemaNameCase returns the extra-component-schema-name-case rule.
//
// # What it checks
//
// Every key of components.schemas must follow the configured case convention,
// PascalCase by default. The shared casing option selects another convention,
// for example kebab-case for an API whose schemas are named pet-owner, and the
// ignore option excludes generated names that follow no convention.
//
// # Why
//
// Schema names become type names in generated clients. A consistent spelling
// produces idiomatic names in every target language and keeps $ref pointers
// predictable.
//
// # Where it comes from
//
// The Oxide OpenAPI lint (component-schema-name-case) and the IBM API Handbook
// ruleset (schema-name-upper-camel-case). The two rows are merged.
//
// # Options
//
//   - casing (string or list, default "PascalCase"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE or
//     lowercase. A name is accepted when it follows any listed convention, an
//     empty list disables the casing check and an unknown value is reported.
//   - ignore (string or list of regular expressions, default none): schema
//     names that match any expression (unanchored, Go RE2 syntax) are not
//     checked. Generated error schemas such as auth-csrf_invalid, a kebab
//     prefix followed by a snake_case code, are excluded with
//     ignore: ["^auth-.*_"], or with "_" to skip every name that contains an
//     underscore. An expression that does not compile is reported.
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
// With casing: kebab-case and ignore: ["^auth-.*_"], pet-store and
// auth-csrf_invalid are accepted and PetStore is reported.
//
// # Known false positives / limitations
//
// PascalCase is checked loosely: an upper-case first letter followed by
// letters and digits, so acronym runs such as HTTPError and ID are accepted.
// The stricter IBM expression that rejects such names is not offered.
func ComponentSchemaNameCase() Rule {
	return New(Spec{
		Name:        "component-schema-name-case",
		Description: "Component schema names must follow the configured case convention.",
		HowToFix:    "Rename the schema to the configured case convention and update every $ref to it.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			casingOption("Case convention of the schema names", casing.Pascal),
			{
				Name: componentSchemaNameCaseIgnore,
				Description: "Regular expressions; schema names that match any of them are not checked, " +
					"for example ^auth-.*_ for generated names such as auth-csrf_invalid.",
				Default: []string{},
			},
		},
		Check: WithCasing(checkComponentSchemaNameCase),
	})
}

// checkComponentSchemaNameCase reports component schema keys with the wrong case.
func checkComponentSchemaNameCase(call *Call, allowed casing.Set) []Finding {
	ignored, err := componentSchemaNameCaseIgnored(call.Strings(componentSchemaNameCaseIgnore))
	if err != nil {
		return One(Finding{Path: "$", Message: "option " + componentSchemaNameCaseIgnore + ": " + err.Error()})
	}

	return seq.FlatMapPairs(oas.Pairs(oas.Dig(call.Root, "components", "schemas")), func(key, _ *yaml.Node) []Finding {
		skipped := slices.ContainsFunc(ignored, func(pattern *regexp.Regexp) bool {
			return pattern.MatchString(key.Value)
		})

		return When(!skipped && !allowed.Matches(key.Value), func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.JoinAll("$.components.schemas", key.Value),
				Message: "schema name " + key.Value + " must be " + allowed.String(),
			}
		})
	})
}

// componentSchemaNameCaseIgnored compiles the ignore patterns.
func componentSchemaNameCaseIgnored(sources []string) ([]*regexp.Regexp, error) {
	patterns := make([]*regexp.Regexp, 0, len(sources))

	for _, source := range sources {
		pattern, err := regexp.Compile(source)
		if err != nil {
			return nil, fmt.Errorf("invalid pattern %q: %w", source, err)
		}

		patterns = append(patterns, pattern)
	}

	return patterns, nil
}

package rules

import (
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// patternPropertiesRestrictionsMaxContent is the node count of a mapping with one entry.
const patternPropertiesRestrictionsMaxContent = 2

// PatternPropertiesRestrictions returns the extra-pattern-properties-restrictions rule.
//
// # What it checks
//
// OpenAPI 3.1 only. Where a schema defines patternProperties, the rule requires
// that it does not also define additionalProperties, that patternProperties is
// an object with at least one entry and at most one entry, and that the single
// pattern is anchored with a leading ^ and a trailing $.
//
// # Why
//
// Several patterns, or a pattern next to additionalProperties, make it
// ambiguous which schema applies to a given property name, and many code
// generators and validators support only a single anchored pattern. Anchoring
// avoids matching property names by accident.
//
// # Where it comes from
//
// The IBM API Handbook ruleset (pattern-properties).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Labels:
//	      type: object
//	      patternProperties:
//	        "^a": {type: string}
//	        "^b": {type: string}
//
// # Good
//
//	components:
//	  schemas:
//	    Labels:
//	      type: object
//	      patternProperties:
//	        "^x-.*$":
//	          type: string
//
// # Known false positives / limitations
//
// The anchoring test is textual (the key starts with ^ and ends with $); an
// escaped final dollar sign is not distinguished from an anchor.
func PatternPropertiesRestrictions() Rule {
	return New(Spec{
		Name:        "pattern-properties-restrictions",
		Description: "patternProperties must have one anchored entry and no additionalProperties.",
		HowToFix:    "Use a single anchored pattern and remove additionalProperties from the same schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Formats:     []string{formatOAS31},
		Check:       PerSchema(checkPatternPropertiesRestrictions),
	})
}

// checkPatternPropertiesRestrictions reports unsupported patternProperties use.
func checkPatternPropertiesRestrictions(_ *Call, schema oas.Schema) []Finding {
	key, patterns := oas.Entry(schema.Node, "patternProperties")
	if key == nil {
		return nil
	}

	return patternPropertiesRestrictionsSchema(schema, patterns)
}

// patternPropertiesRestrictionsSchema applies the restrictions to one schema.
func patternPropertiesRestrictionsSchema(schema oas.Schema, patterns *yaml.Node) []Finding {
	path := oas.Join(schema.Path, "patternProperties")

	switch {
	case oas.Has(schema.Node, "additionalProperties"):
		return []Finding{{
			Node:    schema.Node,
			Path:    schema.Path,
			Message: "patternProperties and additionalProperties are mutually exclusive",
		}}
	case !oas.IsMap(patterns):
		return []Finding{{Node: patterns, Path: path, Message: "patternProperties must be an object"}}
	case len(patterns.Content) == 0:
		return []Finding{{Node: patterns, Path: path, Message: "patternProperties must be a non-empty object"}}
	case len(patterns.Content) > patternPropertiesRestrictionsMaxContent:
		return []Finding{
			{Node: patterns, Path: path, Message: "patternProperties must be an object with at most one entry"},
		}
	}

	pattern := patterns.Content[0]
	if strings.HasPrefix(pattern.Value, "^") && strings.HasSuffix(pattern.Value, "$") {
		return nil
	}

	return []Finding{{
		Node:    pattern,
		Path:    oas.Join(path, pattern.Value),
		Message: "patternProperties patterns should be anchored with ^ and $",
	}}
}

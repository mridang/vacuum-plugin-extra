package rules

import (
	"slices"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// SingleTypePlusNull returns the extra-single-type-plus-null rule.
//
// # What it checks
//
// In OpenAPI 3.1 documents, the type list of a schema may hold at most one type
// besides "null". Once null is ignored, ["string"] and ["string", "null"] are
// fine, while ["string", "integer"] and ["string", "integer", "null"] are
// reported.
//
// # Why
//
// A schema that accepts several unrelated types is hard to model in typed
// languages and in most code generators. A nullable value is the one union that
// maps cleanly everywhere, so IBM's guideline allows nullability and nothing
// else.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-avoid-multiple-types).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    Id:
//	      type: [string, integer]
//
// # Good
//
//	components:
//	  schemas:
//	    Id:
//	      type: [string, "null"]
//
// # Known false positives / limitations
//
// Only the type keyword of schemas is examined; unions modelled with oneOf are
// not. OpenAPI 3.0 documents are not checked since type is a single string
// there.
func SingleTypePlusNull() Rule {
	return New(Spec{
		Name:        "single-type-plus-null",
		Description: "A schema type list must not hold more than one type besides null.",
		HowToFix:    "Keep one type, optionally with null, or model the union with oneOf.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Formats:     []string{formatOAS31},
		Check:       PerSchema(checkSingleTypePlusNull),
	})
}

func checkSingleTypePlusNull(_ *Call, schema oas.Schema) []Finding {
	typeNode := oas.Get(schema.Node, "type")
	if !oas.IsSeq(typeNode) {
		return nil
	}

	others := seq.Filter(slices.Values(oas.Types(schema.Node)), func(name string) bool {
		return name != keywordNull
	})

	return When(len(others) > 1, func() Finding {
		return Finding{
			Node:    typeNode,
			Path:    oas.Join(schema.Path, "type"),
			Message: "type list must hold a single type besides null",
		}
	})
}

package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// NoPathItemLevelParameters returns the extra-no-path-item-level-parameters rule.
//
// # What it checks
//
// A path item must not define a parameters list. Parameters declared on the
// path item apply to every operation of the path, which hides them from the
// operations that use them and surprises tools that only look at the
// operation. Every parameter must be declared on the operation itself, which
// keeps each operation self-describing.
//
// # Where it comes from
//
// openapilint, which forbids path-item level parameters.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    parameters:
//	      - {name: id, in: path, required: true, schema: {type: string}}
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Good
//
//	paths:
//	  /pets/{id}:
//	    get:
//	      parameters:
//	        - {name: id, in: path, required: true, schema: {type: string}}
//	      responses:
//	        "200":
//	          description: ok
//
// # Known false positives
//
// Path parameters repeated on every operation of a path item are the common
// reason to use this feature; the rule intentionally asks for the repetition.
func NoPathItemLevelParameters() Rule {
	return New(Spec{
		Name:        "no-path-item-level-parameters",
		Description: "Path items must not define a parameters list.",
		HowToFix:    "Move the parameters into each operation of the path item.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkNoPathItemLevelParameters),
	})
}

func checkNoPathItemLevelParameters(call *Call) []Finding {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(call.Root, "paths")), func(route, item *yaml.Node) []Finding {
		key, parameters := oas.Entry(item, "parameters")

		return When(parameters != nil, func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.Join(oas.Join("$.paths", route.Value), key.Value),
				Message: "path item " + route.Value + " must not define parameters",
			}
		})
	})
}

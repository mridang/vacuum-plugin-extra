package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ParameterSchemaTypeRequired returns the extra-parameter-schema-type-required
// rule.
//
// # What it checks
//
// Every parameter of a path item or an operation must have a schema whose type
// is declared. A parameter is reported when it has no schema at all (which is
// the case for parameters that use content instead of schema), or when the
// schema, after following a local $ref, has no type or an empty type. Path
// item parameters are checked once, and operation parameters are checked on
// their own, so a parameter is never reported twice.
//
// Clients and tool descriptions are generated from the type of a parameter.
// Without it, agents and code generators have to guess whether a value is a
// string, a number or a list.
//
// # Where it comes from
//
// ai-api-lint, which requires every operation parameter to have schema.type.
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: limit
//	          in: query
//	          schema:
//	            minimum: 1
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: limit
//	          in: query
//	          schema:
//	            type: integer
//
// # Known false positives
//
// A schema that is a composition (oneOf, anyOf, allOf) without its own type is
// reported, and so is every parameter that uses content. Parameters under
// components/parameters that no operation uses are not checked.
func ParameterSchemaTypeRequired() Rule {
	return New(Spec{
		Name:        "parameter-schema-type-required",
		Description: "Every parameter must have a schema with a declared type.",
		HowToFix:    "Add a schema with a type to the parameter.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check: Concat(
			PerPathItem(checkParameterSchemaTypeRequiredItem),
			PerOperation(checkParameterSchemaTypeRequiredOperation),
		),
	})
}

func checkParameterSchemaTypeRequiredItem(call *Call, item oas.PathItem) []Finding {
	return parameterSchemaTypeRequiredList(call.Root, item.Node, item.Path)
}

func checkParameterSchemaTypeRequiredOperation(call *Call, operation oas.Operation) []Finding {
	return parameterSchemaTypeRequiredList(call.Root, operation.Node, operation.Path)
}

func parameterSchemaTypeRequiredList(root, owner *yaml.Node, ownerPath string) []Finding {
	listPath := oas.Join(ownerPath, "parameters")

	return seq.FlatMapPairs(oas.Items(oas.Get(owner, "parameters")), func(i int, item *yaml.Node) []Finding {
		parameter := oas.Resolve(root, item)
		schema := oas.Resolve(root, oas.Get(parameter, "schema"))
		path := parameterSchemaTypeRequiredPath(oas.Index(listPath, i), schema, item)

		return When(oas.IsMap(parameter) && len(oas.Types(schema)) == 0, func() Finding {
			return Finding{
				Node:    parameter,
				Path:    path,
				Message: "parameter " + oas.StringAt(parameter, "name") + " must have a schema with a type",
			}
		})
	})
}

// parameterSchemaTypeRequiredPath points at the inline schema when there is one,
// and at the parameter otherwise.
func parameterSchemaTypeRequiredPath(parameterPath string, schema, item *yaml.Node) string {
	if oas.IsMap(schema) && oas.Ref(item) == "" {
		return oas.Join(parameterPath, "schema")
	}

	return parameterPath
}

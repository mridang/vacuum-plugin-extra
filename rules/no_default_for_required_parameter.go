package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const noDefaultForRequiredParameterTruthyOption = "onlyTruthy"

// NoDefaultForRequiredParameter returns the extra-no-default-for-required-parameter rule.
//
// # What it checks
//
// A parameter with required: true must not have a default value in its schema.
// Both operation parameters (including those declared on the path item) and
// reusable parameters under components.parameters are checked, and the schema
// is resolved through local $ref. The finding is reported on the default
// keyword when the schema is inline, and on the schema otherwise.
//
// # Why
//
// A default is the value a server assumes when the client omits the parameter.
// A required parameter can never be omitted, so the default is dead
// documentation that suggests the parameter is optional and misleads generated
// clients and mock servers.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-no-default-for-required-parameter) and the
// openapilint rule required-parameter-must-not-have-default. The two are merged;
// IBM flags any default, openapilint only a truthy one, which is the onlyTruthy
// option.
//
// # Options
//
//   - onlyTruthy (boolean, default false): when true, only a truthy default is
//     reported, so false, 0, the empty string and null are tolerated.
//
// # Bad
//
//	parameters:
//	  - name: limit
//	    in: query
//	    required: true
//	    schema:
//	      type: integer
//	      default: 10
//
// # Good
//
//	parameters:
//	  - name: limit
//	    in: query
//	    required: false
//	    schema:
//	      type: integer
//	      default: 10
//
// # Known false positives
//
// None known. A parameter whose schema lives in content rather than schema is
// not inspected.
func NoDefaultForRequiredParameter() Rule {
	return New(Spec{
		Name:        "no-default-for-required-parameter",
		Description: "A required parameter must not have a default value.",
		HowToFix:    "Remove the default, or make the parameter optional.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        noDefaultForRequiredParameterTruthyOption,
			Description: "Only report truthy defaults.",
			Default:     false,
		}},
		Check: OnDocument(checkNoDefaultForRequiredParameter),
	})
}

// checkNoDefaultForRequiredParameter inspects every distinct parameter of the
// operations and of components.parameters once.
func checkNoDefaultForRequiredParameter(call *Call) []Finding {
	fromOperations := seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []oas.Parameter {
		return oas.Parameters(call.Root, operation)
	})
	fromComponents := seq.FlatMapPairs(
		oas.Pairs(oas.Dig(call.Root, "components", "parameters")),
		func(name, node *yaml.Node) []oas.Parameter {
			return []oas.Parameter{{
				Node: oas.Resolve(call.Root, node),
				Path: oas.JoinAll("$.components.parameters", name.Value),
			}}
		},
	)
	parameters := slices.Concat(fromOperations, fromComponents)

	return seq.FlatMapIndexed(parameters, func(i int, parameter oas.Parameter) []Finding {
		first := slices.IndexFunc(parameters, func(other oas.Parameter) bool {
			return other.Path == parameter.Path
		})
		if first != i {
			return nil
		}

		return noDefaultForRequiredParameterOne(call, parameter)
	})
}

// noDefaultForRequiredParameterOne inspects a single parameter.
func noDefaultForRequiredParameterOne(call *Call, parameter oas.Parameter) []Finding {
	if !parameter.Required() {
		return nil
	}

	schema := oas.Get(parameter.Node, "schema")
	key, value := oas.Entry(oas.Resolve(call.Root, schema), "default")

	return When(
		key != nil &&
			(!call.Bool(noDefaultForRequiredParameterTruthyOption) || noDefaultForRequiredParameterTruthy(value)),
		func() Finding {
			return Finding{
				Node:    value,
				Path:    noDefaultForRequiredParameterPath(parameter, schema),
				Message: "required parameter '" + parameter.Name() + "' must not have a default value",
			}
		},
	)
}

// noDefaultForRequiredParameterPath returns the path of the default, or the
// path of the schema when it is a reference and the default lives elsewhere.
func noDefaultForRequiredParameterPath(parameter oas.Parameter, schema *yaml.Node) string {
	schemaPath := oas.Join(parameter.Path, "schema")
	if oas.Ref(schema) != "" {
		return schemaPath
	}

	return oas.Join(schemaPath, "default")
}

// noDefaultForRequiredParameterTruthy reports whether a default value is truthy.
func noDefaultForRequiredParameterTruthy(value *yaml.Node) bool {
	if !oas.IsScalar(value) {
		return len(value.Content) > 0
	}

	return !slices.Contains(strings.Fields("false 0 null ~ 0.0"), value.Value) && value.Value != ""
}

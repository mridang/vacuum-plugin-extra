package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	// queryParamWireKeyCollisionQuery is the parameter location checked.
	queryParamWireKeyCollisionQuery = "query"
	// queryParamWireKeyCollisionFalse is the value that turns explode off.
	queryParamWireKeyCollisionFalse = "false"
)

// QueryParamWireKeyCollision returns the extra-query-param-wire-key-collision rule.
//
// # What it checks
//
// Query parameters of one operation are serialised to wire keys. A plain
// parameter produces one key, its own name. An object parameter with style form
// and explode true (the defaults) produces one key per property of the object.
// A deepObject parameter produces name[property] keys. When two parameters of
// an operation produce the same key, their schemas must be compatible, which
// means that they declare the same type and format. Incompatible collisions are
// reported at the parameter that comes later.
//
// # Why
//
// On the wire, the server only sees key=value pairs. If an exploded object
// property and another parameter share a key but disagree on the type, the
// server cannot know which definition to apply to the value, so validation and
// generated clients behave unpredictably.
//
// # Where it comes from
//
// Specmatic (query parameters serialising to the same wire key).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /search:
//	    get:
//	      parameters:
//	        - name: color
//	          in: query
//	          schema:
//	            type: string
//	        - name: filter
//	          in: query
//	          schema:
//	            type: object
//	            properties:
//	              color:
//	                type: integer
//
// # Good
//
//	paths:
//	  /search:
//	    get:
//	      parameters:
//	        - name: color
//	          in: query
//	          schema:
//	            type: string
//	        - name: filter
//	          in: query
//	          schema:
//	            type: object
//	            properties:
//	              size:
//	                type: integer
//
// # Known false positives / limitations
//
// Array, spaceDelimited and pipeDelimited serialisation produce the parameter
// name as the key and are handled as plain parameters. Compatibility compares
// only type and format of $ref-resolved schemas, not constraints.
func QueryParamWireKeyCollision() Rule {
	return New(Spec{
		Name:        "query-param-wire-key-collision",
		Description: "Query parameters that serialise to the same wire key must have compatible schemas.",
		HowToFix:    "Rename one of the parameters or properties, or give the colliding schemas the same type.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerOperation(checkQueryParamWireKeyCollision),
	})
}

// queryParamWireKeyCollisionKey is one wire key with the shape of its schema.
type queryParamWireKeyCollisionKey struct {
	key   string
	shape string
}

// queryParamWireKeyCollisionEntry is a wire key together with the parameter that produces it.
type queryParamWireKeyCollisionEntry struct {
	parameter oas.Parameter
	wire      queryParamWireKeyCollisionKey
}

// checkQueryParamWireKeyCollision reports incompatible wire key collisions.
func checkQueryParamWireKeyCollision(call *Call, operation oas.Operation) []Finding {
	entries := seq.FlatMapSlice(
		oas.Parameters(call.Root, operation),
		func(parameter oas.Parameter) []queryParamWireKeyCollisionEntry {
			if parameter.In() != queryParamWireKeyCollisionQuery {
				return nil
			}

			return seq.FlatMapSlice(
				queryParamWireKeyCollisionKeys(call.Root, parameter),
				func(wire queryParamWireKeyCollisionKey) []queryParamWireKeyCollisionEntry {
					return []queryParamWireKeyCollisionEntry{{parameter: parameter, wire: wire}}
				},
			)
		},
	)

	return seq.FlatMapIndexed(entries, func(i int, entry queryParamWireKeyCollisionEntry) []Finding {
		first := slices.IndexFunc(entries, func(other queryParamWireKeyCollisionEntry) bool {
			return other.wire.key == entry.wire.key
		})

		return When(first != i && entries[first].wire.shape != entry.wire.shape, func() Finding {
			return Finding{
				Node: entry.parameter.Node,
				Path: entry.parameter.Path,
				Message: "query parameter " + entry.parameter.Name() + " serialises to wire key " +
					entry.wire.key + " with an incompatible schema",
			}
		})
	})
}

// queryParamWireKeyCollisionKeys lists the wire keys of a query parameter.
func queryParamWireKeyCollisionKeys(root *yaml.Node, parameter oas.Parameter) []queryParamWireKeyCollisionKey {
	schema := oas.Resolve(root, oas.Get(parameter.Node, "schema"))
	style := oas.StringAt(parameter.Node, "style")

	if oas.HasType(schema, "object") && style == "deepObject" {
		return queryParamWireKeyCollisionProperties(root, schema, parameter.Name()+"[", "]")
	}

	exploded := oas.StringAt(parameter.Node, "explode") != queryParamWireKeyCollisionFalse
	if oas.HasType(schema, "object") && (style == "" || style == keywordForm) && exploded {
		return queryParamWireKeyCollisionProperties(root, schema, "", "")
	}

	return []queryParamWireKeyCollisionKey{{key: parameter.Name(), shape: queryParamWireKeyCollisionShape(schema)}}
}

// queryParamWireKeyCollisionProperties lists one key per object property.
func queryParamWireKeyCollisionProperties(
	root, schema *yaml.Node,
	prefix, suffix string,
) []queryParamWireKeyCollisionKey {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Get(schema, "properties")),
		func(name, property *yaml.Node) []queryParamWireKeyCollisionKey {
			return []queryParamWireKeyCollisionKey{{
				key:   prefix + name.Value + suffix,
				shape: queryParamWireKeyCollisionShape(oas.Resolve(root, property)),
			}}
		},
	)
}

// queryParamWireKeyCollisionShape summarises the type and format of a schema.
func queryParamWireKeyCollisionShape(schema *yaml.Node) string {
	types := slices.Sorted(slices.Values(oas.Types(schema)))

	return strings.Join(types, ",") + "/" + oas.StringAt(schema, "format")
}

package rules

import (
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// RefPatternComponentLocation returns the extra-ref-pattern-component-location rule.
//
// # What it checks
//
// A port of IBM's ibm-ref-pattern. Every $ref whose value is a string that
// starts with # is checked against the location of the object that holds it.
// The location (the keys and array indexes from the document root to the
// holder, joined with commas) decides which kind of object is expected, and the
// value must start with the matching prefix. The patterns are tried in this
// order and the first one that matches decides:
//
//   - schemas: the holder is a schema, properties,<name>, items,
//     additionalProperties or an allOf, anyOf or oneOf entry; the prefix is
//     #/components/schemas/;
//   - parameters: holder at parameters,<index>; #/components/parameters/;
//   - responses: holder at responses,<code>; #/components/responses/;
//   - requestBodies: holder at requestBody; #/components/requestBodies/;
//   - links: holder at links,<name>; #/components/links/;
//   - examples: holder at examples,<name> (media types, parameters and
//     components alike); #/components/examples/;
//   - headers: holder at headers,<name>; #/components/headers/;
//   - securitySchemes: holder at securitySchemes,<name>; #/components/securitySchemes/;
//   - callbacks: holder at callbacks,<name>; #/components/callbacks/.
//
// The message is "$refs to <kind> should start with '<prefix>'" and the finding
// is placed on the $ref. Only the prefix is compared, like in IBM's rule, so a
// reference such as #/components/schemas/Pet/properties/name is accepted. A
// $ref that is not a string (a property named $ref) and a $ref to another file
// or URL are not inspected. Every $ref in the document is visited, including
// the ones that sit in example values and vendor extensions, as IBM's
// $..$ref does.
//
// # Why
//
// Pointing a schema at #/paths/~1pets/get/responses/200/content/... works for a
// resolver but defeats reuse, breaks bundlers and makes generators invent
// unstable names. Components are the supported place for reusable objects.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-ref-pattern, severity warn).
//
// # Options
//
// None.
//
// # Bad
//
//	parameters:
//	  - $ref: "#/components/schemas/PetId"
//
// # Good
//
//	parameters:
//	  - $ref: "#/components/parameters/PetId"
//
// # Known false positives and limitations
//
// A $ref held by a path item, a security requirement or other objects that the
// patterns do not list is not checked, as in the original. Keys that contain a
// comma can confuse the location patterns, as in the original.
func RefPatternComponentLocation() Rule {
	return New(Spec{
		Name:        "ref-pattern-component-location",
		Description: "A $ref must start with the components prefix that matches the kind of object it sits in.",
		HowToFix:    "Move the target into components under the matching section and reference it as #/components/<kind>/<name>.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkRefPatternComponentLocation),
	})
}

// refPatternComponentLocationKind pairs a location pattern with the components
// section the reference must point into.
type refPatternComponentLocationKind struct {
	name    string
	pattern *regexp.Regexp
}

// refPatternComponentLocationKinds returns the IBM location patterns in the
// order in which they are tried. Each entry of the table is name=pattern.
func refPatternComponentLocationKinds() []refPatternComponentLocationKind {
	table := []string{
		literalSchemas + `=,((schema)|(properties,[^,]+)|(items)|(additionalProperties)|((allOf|anyOf|oneOf),\d+))$`,
		`parameters=,parameters,\d+$`,
		`responses=,responses,[^,]+$`,
		`requestBodies=,requestBody$`,
		`links=,links,[^,]+$`,
		`examples=,examples,[^,]+$`,
		`headers=,headers,[^,]+$`,
		`securitySchemes=,securitySchemes,[^,]+$`,
		`callbacks=,callbacks,[^,]+$`,
	}

	return seq.Map(slices.Values(table), func(entry string) refPatternComponentLocationKind {
		name, pattern, _ := strings.Cut(entry, "=")

		return refPatternComponentLocationKind{name: name, pattern: regexp.MustCompile(pattern)}
	})
}

func checkRefPatternComponentLocation(call *Call) []Finding {
	return refPatternComponentLocationWalk(
		refPatternComponentLocationKinds(),
		oas.Unwrap(call.Root),
		nil,
		"$",
	)
}

func refPatternComponentLocationWalk(
	kinds []refPatternComponentLocationKind,
	node *yaml.Node,
	segments []string,
	path string,
) []Finding {
	switch {
	case oas.IsMap(node):
		return slices.Concat(
			refPatternComponentLocationCheck(kinds, node, segments, path),
			seq.FlatMapPairs(oas.Pairs(node), func(key, value *yaml.Node) []Finding {
				return refPatternComponentLocationWalk(
					kinds,
					value,
					append(slices.Clone(segments), key.Value),
					oas.Join(path, key.Value),
				)
			}),
		)
	case oas.IsSeq(node):
		return seq.FlatMapPairs(oas.Items(node), func(i int, item *yaml.Node) []Finding {
			return refPatternComponentLocationWalk(
				kinds,
				item,
				append(slices.Clone(segments), strconv.Itoa(i)),
				oas.Index(path, i),
			)
		})
	}

	return nil
}

func refPatternComponentLocationCheck(
	kinds []refPatternComponentLocationKind,
	node *yaml.Node,
	segments []string,
	path string,
) []Finding {
	key, ref := oas.Entry(node, "$ref")
	if !oas.IsScalar(ref) || ref.Tag == refPatternComponentLocationNullTag || !strings.HasPrefix(ref.Value, "#") ||
		len(segments) == 0 {
		return nil
	}

	location := "," + strings.Join(segments, ",")

	matched := slices.IndexFunc(kinds, func(kind refPatternComponentLocationKind) bool {
		return kind.pattern.MatchString(location)
	})
	if matched < 0 {
		return nil
	}

	kind := kinds[matched]
	prefix := "#/components/" + kind.name + "/"

	return When(!strings.HasPrefix(ref.Value, prefix), func() Finding {
		return Finding{
			Node:    ref,
			Path:    oas.Join(path, key.Value),
			Message: "$refs to " + kind.name + " should start with '" + prefix + "'",
		}
	})
}

const refPatternComponentLocationNullTag = "!!null"

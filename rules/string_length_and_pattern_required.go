package rules

import (
	"cmp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/formats"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	stringLengthAndPatternRequiredPatternOption = "requirePattern"
	stringLengthAndPatternRequiredMinOption     = "requireMinLength"
	stringLengthAndPatternRequiredMaxOption     = "requireMaxLength"
	stringLengthAndPatternRequiredFormats       = "shapedFormats"
	stringLengthAndPatternRequiredString        = "string"
	stringLengthAndPatternRequiredEnum          = "enum"
	stringLengthAndPatternRequiredDate          = "date"
	stringLengthAndPatternRequiredURL           = "url"
	stringLengthAndPatternRequiredMaxDepth      = 16
)

// StringLengthAndPatternRequired returns the extra-string-length-and-pattern-required rule.
//
// # What it checks
//
// A faithful port of IBM's ibm-string-attributes. Every string schema must
// define pattern, minLength and maxLength so the set of valid values is clearly
// bounded; every non-string schema must not define them.
//
// The schemas examined are the root schemas (components, parameters, request
// and response bodies, headers) and, below them, the properties, items,
// additionalProperties and patternProperties schemas plus the members of allOf,
// oneOf and anyOf (the nested schemas of those members are checked; the
// composed member itself is not, exactly as in IBM's validateNestedSchemas).
// Schemas below not, if, then and else are skipped.
//
// Whether a schema is a string schema, and what its format, pattern, minLength,
// maxLength and enum are, is decided in a composition-aware way like IBM's
// schemaHasConstraint: a keyword counts when the schema declares it, when any
// allOf member does, or when every oneOf or anyOf member does ($refs to local
// components are followed).
//
// For a string schema without an enum or const (either one anywhere in the
// composition bypasses every check, because the values are already bounded; the
// const keyword is an addition to IBM's rule, which predates OpenAPI 3.1):
//
//   - pattern is required unless format is binary, byte, date, date-time or url,
//     or one of the shapedFormats;
//   - minLength is required unless format is date, identifier or url, or one of
//     the shapedFormats;
//   - maxLength is required unless format is date (the shapedFormats do not
//     exempt it, since formats such as uri and email are unbounded);
//   - minLength must not be greater than maxLength.
//
// For a schema that declares a type other than string, a declared pattern,
// minLength or maxLength is reported at the keyword. A schema without any type
// (a mixin fragment meant for allOf) is not reported.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-string-attributes, severity warn).
//
// # Options
//
//   - requirePattern: require pattern. Default: true.
//
//   - requireMinLength: require minLength. Default: true.
//
//   - requireMaxLength: require maxLength. Default: true.
//
//   - shapedFormats (list of strings): the formats whose values are never empty
//     and already shaped, which satisfy the pattern and minLength demands.
//     Default: date, date-time, time, duration, uuid, uri, uri-reference, iri,
//     url, email, idn-email, hostname, idn-hostname, ipv4, ipv6, json-pointer,
//     regex, byte. Set it to an empty list to require both on every string.
//
// The requirePattern, requireMinLength and requireMaxLength options switch off
// the "missing" findings only; the not-a-string and
// minLength-greater-than-maxLength checks always run.
//
// # Bad
//
//	components:
//	  schemas:
//	    Thing:
//	      type: object
//	      properties:
//	        thing_id:
//	          type: string
//	        count:
//	          type: integer
//	          minLength: 1
//
// # Good
//
//	components:
//	  schemas:
//	    Thing:
//	      type: object
//	      properties:
//	        thing_id:
//	          type: string
//	          pattern: "^[a-z0-9]+$"
//	          minLength: 1
//	          maxLength: 64
//
// # Known false positives and limitations
//
// IBM validates only schemas that are used by operations (after resolving
// $refs); this rule also validates every components.schemas entry, since usage
// cannot be traced without resolving the document, so an unused component is
// reported too. A $ref property is not followed into its target for the
// finding (the target is checked where it is defined). Free text such as
// descriptions legitimately has no pattern; set requirePattern to false.
func StringLengthAndPatternRequired() Rule {
	return New(Spec{
		Name:        "string-length-and-pattern-required",
		Description: "String schemas must define pattern, minLength and maxLength, and other schemas must not.",
		HowToFix:    "Add the missing pattern, minLength and maxLength keywords to the string schema, or remove them from non-string schemas.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        stringLengthAndPatternRequiredPatternOption,
				Description: "Require the pattern keyword.",
				Default:     true,
			},
			{
				Name:        stringLengthAndPatternRequiredMinOption,
				Description: "Require the minLength keyword.",
				Default:     true,
			},
			{
				Name:        stringLengthAndPatternRequiredMaxOption,
				Description: "Require the maxLength keyword.",
				Default:     true,
			},
			{
				Name:        stringLengthAndPatternRequiredFormats,
				Description: "Formats that satisfy the pattern and minLength demands because their values are never empty.",
				Default:     formats.NonEmpty(),
			},
		},
		Check: OnDocument(checkStringLengthAndPatternRequired),
	})
}

// stringLengthAndPatternRequiredBypass returns the formats that exempt a
// string schema from requiring the given keyword.
func stringLengthAndPatternRequiredBypass(call *Call, keyword string) []string {
	switch keyword {
	case literalPattern:
		return slices.Concat(
			[]string{
				"binary",
				"byte",
				stringLengthAndPatternRequiredDate,
				formatDateTime,
				stringLengthAndPatternRequiredURL,
			},
			call.Strings(stringLengthAndPatternRequiredFormats),
		)
	case keywordMinLength:
		return slices.Concat(
			[]string{stringLengthAndPatternRequiredDate, "identifier", stringLengthAndPatternRequiredURL},
			call.Strings(stringLengthAndPatternRequiredFormats),
		)
	default:
		return []string{stringLengthAndPatternRequiredDate}
	}
}

// checkStringLengthAndPatternRequired validates every schema that is not nested
// below another schema, and descends into the nested ones.
func checkStringLengthAndPatternRequired(call *Call) []Finding {
	return seq.FlatMapSlice(stringLengthAndPatternRequiredRoots(call.Root), func(root oas.Schema) []Finding {
		return stringLengthAndPatternRequiredWalk(call, root.Node, root.Path, true, 0)
	})
}

// stringLengthAndPatternRequiredRoots returns the schemas that are not nested
// below another schema.
func stringLengthAndPatternRequiredRoots(root *yaml.Node) []oas.Schema {
	all := slices.Collect(oas.Schemas(root))
	children := seq.FlatMapSlice(all, func(schema oas.Schema) []*yaml.Node {
		return seq.Map(
			slices.Values(stringLengthAndPatternRequiredChildren(schema.Node, schema.Path)),
			func(child stringLengthAndPatternRequiredChild) *yaml.Node {
				return child.node
			},
		)
	})

	return seq.Filter(slices.Values(all), func(schema oas.Schema) bool {
		return !slices.Contains(children, schema.Node)
	})
}

// stringLengthAndPatternRequiredChild is a sub-schema of a schema.
type stringLengthAndPatternRequiredChild struct {
	node *yaml.Node
	path string
	self bool
}

// stringLengthAndPatternRequiredChildOf returns the sub-schema when it is a
// mapping, and nothing otherwise.
func stringLengthAndPatternRequiredChildOf(
	node *yaml.Node,
	path string,
	self bool,
) []stringLengthAndPatternRequiredChild {
	if !oas.IsMap(node) {
		return nil
	}

	return []stringLengthAndPatternRequiredChild{{node: node, path: path, self: self}}
}

// stringLengthAndPatternRequiredChildren lists every sub-schema of a schema
// that the walker knows about; self marks the ones that are validated
// themselves (not the composed members).
func stringLengthAndPatternRequiredChildren(schema *yaml.Node, path string) []stringLengthAndPatternRequiredChild {
	properties := seq.FlatMapPairs(
		oas.Pairs(oas.Get(schema, "properties")),
		func(name, value *yaml.Node) []stringLengthAndPatternRequiredChild {
			return stringLengthAndPatternRequiredChildOf(value, oas.JoinAll(path, "properties", name.Value), true)
		},
	)
	patterns := seq.FlatMapPairs(
		oas.Pairs(oas.Get(schema, literalPatternproperties)),
		func(name, value *yaml.Node) []stringLengthAndPatternRequiredChild {
			return stringLengthAndPatternRequiredChildOf(
				value,
				oas.JoinAll(path, literalPatternproperties, name.Value),
				true,
			)
		},
	)
	members := seq.FlatMapSlice(
		[]string{stringLengthAndPatternRequiredAllOf, literalOneof, stringLengthAndPatternRequiredAnyOf},
		func(keyword string) []stringLengthAndPatternRequiredChild {
			return seq.FlatMapPairs(
				oas.Items(oas.Get(schema, keyword)),
				func(i int, member *yaml.Node) []stringLengthAndPatternRequiredChild {
					return stringLengthAndPatternRequiredChildOf(member, oas.Index(oas.Join(path, keyword), i), false)
				},
			)
		},
	)
	keywords := seq.FlatMap(
		strings.FieldsSeq("not if then else prefixItems contains propertyNames"),
		func(keyword string) []stringLengthAndPatternRequiredChild {
			return stringLengthAndPatternRequiredChildOf(oas.Get(schema, keyword), oas.Join(path, keyword), false)
		},
	)

	return slices.Concat(
		properties,
		stringLengthAndPatternRequiredChildOf(oas.Get(schema, "items"), oas.Join(path, "items"), true),
		stringLengthAndPatternRequiredChildOf(
			oas.Get(schema, "additionalProperties"), oas.Join(path, "additionalProperties"), true),
		patterns,
		members,
		keywords,
	)
}

// stringLengthAndPatternRequiredWalk validates a schema when self is set and
// then every nested schema, mirroring IBM's validateNestedSchemas.
func stringLengthAndPatternRequiredWalk(call *Call, schema *yaml.Node, path string, self bool, depth int) []Finding {
	if !oas.IsMap(schema) || oas.Ref(schema) != "" || depth > stringLengthAndPatternRequiredMaxDepth {
		return nil
	}

	nested := seq.FlatMapSlice(
		stringLengthAndPatternRequiredChildren(schema, path),
		func(child stringLengthAndPatternRequiredChild) []Finding {
			if stringLengthAndPatternRequiredSkipped(path, child.path) {
				return nil
			}

			return stringLengthAndPatternRequiredWalk(call, child.node, child.path, child.self, depth+1)
		},
	)
	if !self {
		return nested
	}

	return slices.Concat(stringLengthAndPatternRequiredCheck(call, schema, path), nested)
}

// stringLengthAndPatternRequiredSkipped reports whether the child is below a
// keyword that IBM's walker does not descend into.
func stringLengthAndPatternRequiredSkipped(parent, child string) bool {
	suffix := child[len(parent):]

	return slices.ContainsFunc(
		[]string{".not", ".if", ".then", ".else", ".prefixItems", ".contains", ".propertyNames"},
		func(keyword string) bool {
			return strings.HasPrefix(suffix, keyword)
		},
	)
}

// stringLengthAndPatternRequiredHit is the outcome of a composition-aware
// search. ok tells whether the constraint holds. found is the value of the
// last probe that matched while searching, which can be set even when ok is
// false because a failed oneOf or anyOf keeps the values of its earlier members.
type stringLengthAndPatternRequiredHit struct {
	ok    bool
	found *yaml.Node
}

// stringLengthAndPatternRequiredSearch reports whether the schema, or its
// composition, satisfies the probe, like IBM's schemaHasConstraint: the probe
// matches the schema itself, any allOf member, or every oneOf or anyOf member.
// The probe returns the matched value, or nil.
func stringLengthAndPatternRequiredSearch(
	root, schema *yaml.Node,
	probe func(*yaml.Node) *yaml.Node,
	depth int,
) stringLengthAndPatternRequiredHit {
	resolved := oas.Resolve(root, schema)
	if !oas.IsMap(resolved) || depth > stringLengthAndPatternRequiredMaxDepth {
		return stringLengthAndPatternRequiredHit{}
	}

	if value := probe(resolved); value != nil {
		return stringLengthAndPatternRequiredHit{ok: true, found: value}
	}

	members := func(keyword string) []*yaml.Node {
		return stringLengthAndPatternRequiredMembers(oas.Get(resolved, keyword))
	}
	allOf := stringLengthAndPatternRequiredAny(root, members(stringLengthAndPatternRequiredAllOf), probe, depth+1)
	if allOf.ok {
		return allOf
	}

	oneOf := stringLengthAndPatternRequiredEvery(root, members(literalOneof), probe, depth+1)
	if oneOf.ok {
		return stringLengthAndPatternRequiredHit{ok: true, found: cmp.Or(oneOf.found, allOf.found)}
	}

	anyOf := stringLengthAndPatternRequiredEvery(root, members(stringLengthAndPatternRequiredAnyOf), probe, depth+1)

	return stringLengthAndPatternRequiredHit{ok: anyOf.ok, found: cmp.Or(anyOf.found, oneOf.found, allOf.found)}
}

// stringLengthAndPatternRequiredAny searches the members in order and stops at
// the first one that satisfies the probe.
func stringLengthAndPatternRequiredAny(
	root *yaml.Node,
	members []*yaml.Node,
	probe func(*yaml.Node) *yaml.Node,
	depth int,
) stringLengthAndPatternRequiredHit {
	if len(members) == 0 {
		return stringLengthAndPatternRequiredHit{}
	}

	head := stringLengthAndPatternRequiredSearch(root, members[0], probe, depth)
	if head.ok {
		return head
	}

	rest := stringLengthAndPatternRequiredAny(root, members[1:], probe, depth)

	return stringLengthAndPatternRequiredHit{ok: rest.ok, found: cmp.Or(rest.found, head.found)}
}

// stringLengthAndPatternRequiredEvery searches the members in order and stops at
// the first one that does not satisfy the probe. An empty list never holds.
func stringLengthAndPatternRequiredEvery(
	root *yaml.Node,
	members []*yaml.Node,
	probe func(*yaml.Node) *yaml.Node,
	depth int,
) stringLengthAndPatternRequiredHit {
	if len(members) == 0 {
		return stringLengthAndPatternRequiredHit{}
	}

	head := stringLengthAndPatternRequiredSearch(root, members[0], probe, depth)
	if !head.ok || len(members) == 1 {
		return head
	}

	rest := stringLengthAndPatternRequiredEvery(root, members[1:], probe, depth)

	return stringLengthAndPatternRequiredHit{ok: rest.ok, found: cmp.Or(rest.found, head.found)}
}

// stringLengthAndPatternRequiredMembers returns the members of a sequence node.
func stringLengthAndPatternRequiredMembers(list *yaml.Node) []*yaml.Node {
	return seq.FlatMapPairs(oas.Items(list), func(_ int, member *yaml.Node) []*yaml.Node {
		return []*yaml.Node{member}
	})
}

// stringLengthAndPatternRequiredAttribute returns the value of a keyword found
// by the composition-aware search, or nil.
func stringLengthAndPatternRequiredAttribute(root, schema *yaml.Node, keyword string) *yaml.Node {
	return stringLengthAndPatternRequiredSearch(root, schema, func(node *yaml.Node) *yaml.Node {
		value := oas.Get(node, keyword)
		if oas.IsNull(value) {
			return nil
		}

		return value
	}, 0).found
}

// stringLengthAndPatternRequiredCheck validates one schema.
func stringLengthAndPatternRequiredCheck(call *Call, schema *yaml.Node, path string) []Finding {
	isString := stringLengthAndPatternRequiredSearch(call.Root, schema, func(node *yaml.Node) *yaml.Node {
		if slices.Contains(oas.Types(node), stringLengthAndPatternRequiredString) {
			return node
		}

		return nil
	}, 0).ok

	if !isString {
		if !oas.Has(schema, "type") {
			return nil
		}

		return stringLengthAndPatternRequiredNonString(call, schema, path)
	}

	for _, keyword := range []string{stringLengthAndPatternRequiredEnum, "const"} {
		if stringLengthAndPatternRequiredAttribute(call.Root, schema, keyword) != nil {
			return nil
		}
	}

	return stringLengthAndPatternRequiredStringChecks(call, schema, path)
}

// stringLengthAndPatternRequiredNonString reports string keywords on a schema
// that is not a string.
func stringLengthAndPatternRequiredNonString(call *Call, schema *yaml.Node, path string) []Finding {
	return seq.FlatMapSlice(
		[]string{literalPattern, keywordMinLength, keywordMaxLength},
		func(keyword string) []Finding {
			return When(stringLengthAndPatternRequiredAttribute(call.Root, schema, keyword) != nil, func() Finding {
				return Finding{
					Node:    schema,
					Path:    oas.Join(path, keyword),
					Message: "'" + keyword + "' should not be defined for non-string schemas",
				}
			})
		},
	)
}

// stringLengthAndPatternRequiredEnabled returns the value of the option that
// requires the keyword.
func stringLengthAndPatternRequiredEnabled(call *Call, keyword string) bool {
	switch keyword {
	case literalPattern:
		return call.Bool(stringLengthAndPatternRequiredPatternOption)
	case keywordMinLength:
		return call.Bool(stringLengthAndPatternRequiredMinOption)
	default:
		return call.Bool(stringLengthAndPatternRequiredMaxOption)
	}
}

// stringLengthAndPatternRequiredStringChecks reports the missing keywords of a
// string schema, and a minLength above the maxLength.
func stringLengthAndPatternRequiredStringChecks(call *Call, schema *yaml.Node, path string) []Finding {
	format := oas.Text(stringLengthAndPatternRequiredAttribute(call.Root, schema, "format"))
	missing := seq.FlatMapSlice(
		[]string{literalPattern, keywordMinLength, keywordMaxLength},
		func(keyword string) []Finding {
			return When(
				stringLengthAndPatternRequiredEnabled(call, keyword) &&
					!slices.Contains(stringLengthAndPatternRequiredBypass(call, keyword), format) &&
					stringLengthAndPatternRequiredAttribute(call.Root, schema, keyword) == nil,
				func() Finding {
					return Finding{
						Node:    schema,
						Path:    path,
						Message: "String schemas should define property '" + keyword + "'",
					}
				},
			)
		},
	)
	minimum, minOK := oas.Number(stringLengthAndPatternRequiredAttribute(call.Root, schema, keywordMinLength))
	maximum, maxOK := oas.Number(stringLengthAndPatternRequiredAttribute(call.Root, schema, keywordMaxLength))
	inverted := When(minOK && maxOK && minimum > maximum, func() Finding {
		return Finding{
			Node:    schema,
			Path:    path,
			Message: "'minLength' cannot be greater than 'maxLength'",
		}
	})

	return slices.Concat(missing, inverted)
}

const (
	stringLengthAndPatternRequiredAllOf = "allOf"
	stringLengthAndPatternRequiredAnyOf = "anyOf"
)

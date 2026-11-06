package rules

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// dateNamedPropertyMustUseDateFormatNames is the option with extra property names.
const dateNamedPropertyMustUseDateFormatNames = "names"

const (
	dateNamedPropertyMustUseDateFormatAllOf      = "allOf"
	dateNamedPropertyMustUseDateFormatNullTag    = "!!null"
	dateNamedPropertyMustUseDateFormatMaxDepth   = 32
	dateNamedPropertyMustUseDateFormatString     = "string"
	dateNamedPropertyMustUseDateFormatItems      = "[]"
	dateNamedPropertyMustUseDateFormatWild       = "*"
	dateNamedPropertyMustUseDateFormatContentKey = "content"
	dateNamedPropertyMustUseDateFormatExamples   = "examples"
	dateNamedPropertyMustUseDateFormatProps      = "properties"
	dateNamedPropertyMustUseDateFormatNameMsg    = `According to its name, this property should use type "string" and format "date" or "date-time"`
	dateNamedPropertyMustUseDateFormatValueMsg   = `According to its example value, this schema should use type "string" and format "date" or "date-time"`
)

// DateNamedPropertyMustUseDateFormat returns the
// extra-date-named-property-must-use-date-format rule.
//
// # What it checks
//
// A port of IBM's ibm-use-date-based-format. Schemas that model dates must be
// strings with format date or date-time; whether a schema models a date is
// decided heuristically, in two ways, for every schema reached from a request
// body, a response body, a parameter or header (and every nested property,
// items, additionalProperties and patternProperties schema below them) as well
// as every components.schemas entry. A schema that is already a date or
// date-time schema (composition-aware: its own type string and format, any
// allOf member, or every oneOf and anyOf member) is accepted.
//
//  1. By name. A schema that is the value of a property (properties.<name>)
//     whose name is date-based and that is a string or integer schema is
//     reported with "According to its name, this property should use type
//     "string" and format "date" or "date-time"". Date-based names are:
//     created, updated, modified, expired, expires; names ending in _at or
//     _on; names starting with date_ or time_; names containing _date_ or
//     _time_; names ending in _date; the names start_time, end_time,
//     create_time, created_time, modify_time, modified_time and update_time;
//     and names containing timestamp. Names ending in depends_on are exempt.
//  2. By example. A string schema whose example value looks like a date or a
//     time is reported with "According to its example value, this schema
//     should use type "string" and format "date" or "date-time"". The example
//     is looked up the way IBM does: the examples of the schema itself and of
//     its allOf, oneOf and anyOf members (example, or the examples array); the
//     matching value inside the example of an enclosing schema (following
//     properties, array items and dictionary values); and, for the schema of a
//     request body, response body or parameter, the example or examples of the
//     media type or parameter object that holds it. The first example found
//     is used; null examples are ignored. A value looks like a date when, for
//     a string, it contains no lowercase letters other than month and day names
//     and matches a month name, YYYY-MM-DD, DD-MM-YYYY, MM-DD-YYYY (with ".",
//     "/" or "-") or hh:mm:ss.
//
// # Why
//
// A logical date typed as a bare string or integer loses its meaning. Date
// formats let generators emit native date types and let validators reject
// malformed values.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-use-date-based-format, severity warn; name and
// value heuristics from utils/date-based-utils.js).
//
// # Options
//
//   - names (list of strings, default empty): additional exact property names
//     that are treated as date-based (an extension of this port).
//
// # Bad
//
//	components:
//	  schemas:
//	    Resource:
//	      type: object
//	      properties:
//	        created_at:
//	          type: integer
//	        stamp:
//	          type: string
//	          example: "1990-12-31T23:59:60Z"
//
// # Good
//
//	components:
//	  schemas:
//	    Resource:
//	      type: object
//	      properties:
//	        created_at:
//	          type: string
//	          format: date-time
//	        stamp:
//	          type: string
//	          format: date-time
//
// # Known false positives and limitations
//
// The name heuristic matches names such as start_date_range that are not
// dates. IBM validates only schemas that an operation uses; this rule also
// validates every components.schemas entry and does not follow a $ref below a
// property into the target's own nested schemas (the target is checked where it
// is defined, but a $ref property is judged by its name and by the target
// schema).
func DateNamedPropertyMustUseDateFormat() Rule {
	return New(Spec{
		Name:        "date-named-property-must-use-date-format",
		Description: "Date-based schemas and properties must be strings with format date or date-time.",
		HowToFix:    "Use type string with format date or date-time for the schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        dateNamedPropertyMustUseDateFormatNames,
				Description: "Additional property names that must use a date format.",
				Default:     []string{},
			},
		},
		Check: OnDocument(checkDateNamedPropertyMustUseDateFormat),
	})
}

// dateNamedPropertyMustUseDateFormatWalker validates the schemas below one
// root. It holds the settings only; the examples gathered for each logical path
// are passed down the walk.
type dateNamedPropertyMustUseDateFormatWalker struct {
	root        *yaml.Node
	extra       []string
	expressions dateNamedPropertyMustUseDateFormatExpressions
}

// dateNamedPropertyMustUseDateFormatRoot is a root schema with the examples of
// the object that holds it.
type dateNamedPropertyMustUseDateFormatRoot struct {
	schema   *yaml.Node
	path     string
	indirect []any
}

// checkDateNamedPropertyMustUseDateFormat reports date-like schemas lacking a date format.
func checkDateNamedPropertyMustUseDateFormat(call *Call) []Finding {
	extra := call.Strings(dateNamedPropertyMustUseDateFormatNames)
	expressions := dateNamedPropertyMustUseDateFormatCompile()
	findings := seq.FlatMapSlice(
		dateNamedPropertyMustUseDateFormatRoots(call.Root),
		func(root dateNamedPropertyMustUseDateFormatRoot) []Finding {
			schema, path := dateNamedPropertyMustUseDateFormatResolve(call.Root, root.schema, root.path, 0)
			walker := dateNamedPropertyMustUseDateFormatWalker{root: call.Root, extra: extra, expressions: expressions}

			return walker.visit(schema, path, nil, "", true, 0, root.indirect)
		},
	)

	return dateNamedPropertyMustUseDateFormatUnique(findings)
}

// dateNamedPropertyMustUseDateFormatUnique keeps the first finding of every
// path and message.
func dateNamedPropertyMustUseDateFormatUnique(findings []Finding) []Finding {
	identity := func(finding Finding) string {
		return finding.Path + "|" + finding.Message
	}
	first := maps.Collect(func(yield func(string, int) bool) {
		for i, finding := range slices.Backward(findings) {
			if !yield(identity(finding), i) {
				return
			}
		}
	})

	return seq.FlatMapIndexed(findings, func(i int, finding Finding) []Finding {
		return When(first[identity(finding)] == i, func() Finding {
			return finding
		})
	})
}

// dateNamedPropertyMustUseDateFormatResolve follows $ref and tracks the path of the target.
func dateNamedPropertyMustUseDateFormatResolve(root, node *yaml.Node, path string, depth int) (*yaml.Node, string) {
	ref := oas.Ref(node)
	target := oas.Pointer(root, ref)

	if depth >= dateNamedPropertyMustUseDateFormatMaxDepth || ref == "" || target == nil {
		return node, path
	}

	tokens := seq.Map(
		slices.Values(strings.Split(strings.TrimPrefix(strings.TrimPrefix(ref, "#"), "/"), "/")),
		func(token string) string {
			return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		},
	)

	return dateNamedPropertyMustUseDateFormatResolve(root, target, oas.JoinAll("$", tokens...), depth+1)
}

// dateNamedPropertyMustUseDateFormatChild extends a logical path by one segment.
func dateNamedPropertyMustUseDateFormatChild(logical []string, segment string) []string {
	return slices.Concat(logical, []string{segment})
}

// visit validates one schema and descends into its nested schemas, like IBM's
// validateNestedSchemas. self is false for the members of allOf, oneOf and
// anyOf, which are not validated themselves. inherited holds the examples of
// the nearest validated ancestor at the logical parent path.
func (w dateNamedPropertyMustUseDateFormatWalker) visit(
	schema *yaml.Node, path string, logical []string, name string, self bool, depth int, inherited []any,
) []Finding {
	if !oas.IsMap(schema) || depth > dateNamedPropertyMustUseDateFormatMaxDepth {
		return nil
	}

	if oas.Ref(schema) != "" {
		if !self {
			return nil
		}

		resolved := oas.Resolve(w.root, schema)

		return w.validate(resolved, path, logical, name, w.combine(resolved, logical, inherited))
	}

	own := w.ownExamples(self, schema, logical, inherited)
	validation := w.validateOwn(self, schema, path, logical, name, own)

	return slices.Concat(
		validation,
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(schema, dateNamedPropertyMustUseDateFormatProps)),
			func(key, value *yaml.Node) []Finding {
				return w.visit(value, oas.JoinAll(path, dateNamedPropertyMustUseDateFormatProps, key.Value),
					dateNamedPropertyMustUseDateFormatChild(logical, key.Value), key.Value, true, depth+1, own)
			},
		),
		w.visit(
			oas.Get(schema, "items"),
			oas.Join(path, "items"),
			dateNamedPropertyMustUseDateFormatChild(
				logical,
				dateNamedPropertyMustUseDateFormatItems,
			),
			"",
			true,
			depth+1,
			own,
		),
		w.visitAdditional(schema, path, logical, depth, own),
		seq.FlatMapPairs(oas.Pairs(oas.Get(schema, literalPatternproperties)), func(key, value *yaml.Node) []Finding {
			return w.visit(
				value,
				oas.JoinAll(path, literalPatternproperties, key.Value),
				dateNamedPropertyMustUseDateFormatChild(
					logical,
					dateNamedPropertyMustUseDateFormatWild,
				),
				"",
				true,
				depth+1,
				own,
			)
		}),
		seq.FlatMapSlice(
			[]string{dateNamedPropertyMustUseDateFormatAllOf, literalOneof, dateNamedPropertyMustUseDateFormatAnyOf},
			func(keyword string) []Finding {
				return seq.FlatMapPairs(oas.Items(oas.Get(schema, keyword)), func(i int, member *yaml.Node) []Finding {
					return w.visit(member, oas.Index(oas.Join(path, keyword), i), logical, name, false, depth+1, own)
				})
			},
		),
	)
}

// visitAdditional descends into additionalProperties when it is a schema.
func (w dateNamedPropertyMustUseDateFormatWalker) visitAdditional(
	schema *yaml.Node, path string, logical []string, depth int, own []any,
) []Finding {
	additional := oas.Get(schema, "additionalProperties")
	if !oas.IsMap(additional) {
		return nil
	}

	return w.visit(
		additional,
		oas.Join(path, "additionalProperties"),
		dateNamedPropertyMustUseDateFormatChild(
			logical,
			dateNamedPropertyMustUseDateFormatWild,
		),
		"",
		true,
		depth+1,
		own,
	)
}

// ownExamples returns the combined examples of a schema that is validated, or
// the inherited ones when it is only traversed.
func (w dateNamedPropertyMustUseDateFormatWalker) ownExamples(
	self bool,
	schema *yaml.Node,
	logical []string,
	inherited []any,
) []any {
	if !self {
		return inherited
	}

	return w.combine(schema, logical, inherited)
}

// validateOwn validates the schema only when it takes part in validation.
func (w dateNamedPropertyMustUseDateFormatWalker) validateOwn(
	self bool,
	schema *yaml.Node,
	path string,
	logical []string,
	name string,
	own []any,
) []Finding {
	if !self {
		return nil
	}

	return w.validate(schema, path, logical, name, own)
}

// dateNamedPropertyMustUseDateFormatInheritedFor selects the examples a schema
// inherits from its logical parent.
func dateNamedPropertyMustUseDateFormatInheritedFor(inherited []any, logical []string) []any {
	if len(logical) == 0 {
		return inherited
	}

	return dateNamedPropertyMustUseDateFormatParental(inherited, logical[len(logical)-1])
}

// dateNamedPropertyMustUseDateFormatFirst returns the first example, or nil.
func dateNamedPropertyMustUseDateFormatFirst(combined []any) any {
	if len(combined) == 0 {
		return nil
	}

	return combined[0]
}

// combine gathers the examples for the schema: its own examples, followed by
// the matching values of the examples of its logical parent (or, for a root,
// those of the object that holds it), without the null ones.
func (w dateNamedPropertyMustUseDateFormatWalker) combine(schema *yaml.Node, logical []string, inherited []any) []any {
	parental := dateNamedPropertyMustUseDateFormatInheritedFor(inherited, logical)

	return slices.DeleteFunc(slices.Concat(w.collectExamples(schema, 0), parental), func(value any) bool {
		return value == nil
	})
}

// validate runs the two checks on the schema with its combined examples.
func (w dateNamedPropertyMustUseDateFormatWalker) validate(
	schema *yaml.Node,
	path string,
	logical []string,
	name string,
	combined []any,
) []Finding {
	message := w.message(
		schema,
		name,
		len(logical) > 0 && name != "",
		dateNamedPropertyMustUseDateFormatFirst(combined),
	)

	return When(message != "", func() Finding {
		return Finding{Node: schema, Path: path, Message: message}
	})
}

// message performs the name and example checks of IBM's performValidation and
// returns the violation message, or the empty string.
func (w dateNamedPropertyMustUseDateFormatWalker) message(
	schema *yaml.Node,
	name string,
	property bool,
	example any,
) string {
	if w.has(schema, dateNamedPropertyMustUseDateFormatDateFormat) {
		return ""
	}

	isString := w.has(schema, func(node *yaml.Node) bool {
		return oas.HasType(node, dateNamedPropertyMustUseDateFormatString)
	})
	isInteger := w.has(schema, func(node *yaml.Node) bool { return oas.HasType(node, "integer") })

	switch {
	case property && dateNamedPropertyMustUseDateFormatName(name, w.extra) && (isString || isInteger):
		return dateNamedPropertyMustUseDateFormatNameMsg
	case isString && example != nil && dateNamedPropertyMustUseDateFormatValue(example, w.expressions):
		return dateNamedPropertyMustUseDateFormatValueMsg
	default:
		return ""
	}
}

// dateNamedPropertyMustUseDateFormatDateFormat reports a string schema with format date or date-time.
func dateNamedPropertyMustUseDateFormatDateFormat(node *yaml.Node) bool {
	format := oas.StringAt(node, "format")

	return oas.HasType(node, dateNamedPropertyMustUseDateFormatString) &&
		(format == formatDate || format == formatDateTime)
}

// has reports whether the schema, or its composition, satisfies the predicate,
// like IBM's schemaHasConstraint.
func (w dateNamedPropertyMustUseDateFormatWalker) has(schema *yaml.Node, predicate func(*yaml.Node) bool) bool {
	return dateNamedPropertyMustUseDateFormatHas(w.root, schema, predicate, 0)
}

func dateNamedPropertyMustUseDateFormatHas(root, schema *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	resolved := oas.Resolve(root, schema)
	if !oas.IsMap(resolved) || depth > dateNamedPropertyMustUseDateFormatMaxDepth {
		return false
	}

	allOf := oas.Get(resolved, dateNamedPropertyMustUseDateFormatAllOf)

	return predicate(resolved) ||
		(oas.IsSeq(allOf) && slices.ContainsFunc(allOf.Content, func(member *yaml.Node) bool {
			return dateNamedPropertyMustUseDateFormatHas(root, member, predicate, depth+1)
		})) ||
		slices.ContainsFunc([]string{literalOneof, dateNamedPropertyMustUseDateFormatAnyOf}, func(keyword string) bool {
			members := oas.Get(resolved, keyword)

			return oas.IsSeq(members) && len(members.Content) > 0 &&
				!slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
					return !dateNamedPropertyMustUseDateFormatHas(root, member, predicate, depth+1)
				})
		})
}

// collectExamples gathers the examples of the schema and of its composed members.
func (w dateNamedPropertyMustUseDateFormatWalker) collectExamples(schema *yaml.Node, depth int) []any {
	resolved := oas.Resolve(w.root, schema)
	if !oas.IsMap(resolved) || depth > dateNamedPropertyMustUseDateFormatMaxDepth {
		return nil
	}

	return slices.Concat(
		dateNamedPropertyMustUseDateFormatDeclared(resolved),
		seq.FlatMapSlice(
			[]string{dateNamedPropertyMustUseDateFormatAllOf, literalOneof, dateNamedPropertyMustUseDateFormatAnyOf},
			func(keyword string) []any {
				return seq.FlatMapPairs(oas.Items(oas.Get(resolved, keyword)), func(_ int, member *yaml.Node) []any {
					return w.collectExamples(member, depth+1)
				})
			},
		),
	)
}

// dateNamedPropertyMustUseDateFormatDeclared returns the examples array of a
// schema, or its example when there is no examples array.
func dateNamedPropertyMustUseDateFormatDeclared(schema *yaml.Node) []any {
	list := oas.Get(schema, dateNamedPropertyMustUseDateFormatExamples)

	switch {
	case oas.IsSeq(list):
		return seq.Map(slices.Values(list.Content), dateNamedPropertyMustUseDateFormatConvert)
	case oas.Has(schema, "example"):
		return []any{dateNamedPropertyMustUseDateFormatConvert(oas.Get(schema, "example"))}
	default:
		return nil
	}
}

// dateNamedPropertyMustUseDateFormatParental finds the values of the child in
// the examples of its logical parent.
func dateNamedPropertyMustUseDateFormatParental(parents []any, child string) []any {
	return seq.FlatMapSlice(parents, func(parent any) []any {
		switch child {
		case dateNamedPropertyMustUseDateFormatItems:
			list, _ := parent.([]any)

			return list
		case dateNamedPropertyMustUseDateFormatWild:
			object, _ := parent.(map[string]any)

			return slices.Collect(maps.Values(object))
		default:
			object, _ := parent.(map[string]any)
			value, found := object[child]

			if !found {
				return nil
			}

			return []any{value}
		}
	})
}

// dateNamedPropertyMustUseDateFormatConvert turns a YAML node into plain values.
func dateNamedPropertyMustUseDateFormatConvert(node *yaml.Node) any {
	unwrapped := oas.Unwrap(node)

	switch {
	case unwrapped == nil:
		return nil
	case oas.IsMap(unwrapped):
		return maps.Collect(func(yield func(string, any) bool) {
			for key, value := range oas.Pairs(unwrapped) {
				if !yield(key.Value, dateNamedPropertyMustUseDateFormatConvert(value)) {
					return
				}
			}
		})
	case oas.IsSeq(unwrapped):
		return seq.Map(slices.Values(unwrapped.Content), dateNamedPropertyMustUseDateFormatConvert)
	case unwrapped.Tag == dateNamedPropertyMustUseDateFormatNullTag:
		return nil
	default:
		return unwrapped.Value
	}
}

// dateNamedPropertyMustUseDateFormatRoots lists the root schemas the rule validates.
func dateNamedPropertyMustUseDateFormatRoots(root *yaml.Node) []dateNamedPropertyMustUseDateFormatRoot {
	return slices.Concat(
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "schemas")),
			func(key, schema *yaml.Node) []dateNamedPropertyMustUseDateFormatRoot {
				return []dateNamedPropertyMustUseDateFormatRoot{{
					schema: schema,
					path:   oas.JoinAll("$.components.schemas", key.Value),
				}}
			},
		),
		seq.FlatMap(oas.PathItems(root), func(item oas.PathItem) []dateNamedPropertyMustUseDateFormatRoot {
			return seq.FlatMapPairs(
				oas.Items(oas.Get(item.Node, "parameters")),
				func(i int, parameter *yaml.Node) []dateNamedPropertyMustUseDateFormatRoot {
					return dateNamedPropertyMustUseDateFormatParameter(
						root,
						parameter,
						oas.Index(oas.Join(item.Path, "parameters"), i),
					)
				},
			)
		}),
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []dateNamedPropertyMustUseDateFormatRoot {
			body := oas.Resolve(root, oas.Get(operation.Node, "requestBody"))

			return slices.Concat(
				seq.FlatMapSlice(
					oas.OwnParameters(root, operation),
					func(parameter oas.Parameter) []dateNamedPropertyMustUseDateFormatRoot {
						return dateNamedPropertyMustUseDateFormatParameter(root, parameter.Node, parameter.Path)
					},
				),
				dateNamedPropertyMustUseDateFormatContent(root, body, oas.Join(operation.Path, "requestBody"), nil),
				seq.FlatMapSlice(
					oas.Responses(root, operation),
					func(response oas.Response) []dateNamedPropertyMustUseDateFormatRoot {
						return slices.Concat(
							dateNamedPropertyMustUseDateFormatContent(root, response.Node, response.Path, nil),
							dateNamedPropertyMustUseDateFormatHeaders(root, response),
						)
					},
				),
			)
		}),
	)
}

// dateNamedPropertyMustUseDateFormatParameter returns the schema roots of a parameter.
func dateNamedPropertyMustUseDateFormatParameter(
	root, parameter *yaml.Node,
	path string,
) []dateNamedPropertyMustUseDateFormatRoot {
	resolved := oas.Resolve(root, parameter)
	own := dateNamedPropertyMustUseDateFormatOpenAPIExamples(root, resolved)

	return slices.Concat(
		dateNamedPropertyMustUseDateFormatSchemaRoot(oas.Get(resolved, "schema"), oas.Join(path, "schema"), own),
		dateNamedPropertyMustUseDateFormatContent(root, resolved, path, own),
	)
}

// dateNamedPropertyMustUseDateFormatSchemaRoot returns the root of a schema, or
// nothing when the schema is absent.
func dateNamedPropertyMustUseDateFormatSchemaRoot(
	schema *yaml.Node,
	path string,
	indirect []any,
) []dateNamedPropertyMustUseDateFormatRoot {
	if schema == nil {
		return nil
	}

	return []dateNamedPropertyMustUseDateFormatRoot{{schema: schema, path: path, indirect: indirect}}
}

// dateNamedPropertyMustUseDateFormatContent returns the schema roots of the
// media types of an object; extra holds the examples of the owning parameter.
func dateNamedPropertyMustUseDateFormatContent(
	root, owner *yaml.Node,
	path string,
	extra []any,
) []dateNamedPropertyMustUseDateFormatRoot {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Get(owner, dateNamedPropertyMustUseDateFormatContentKey)),
		func(name, media *yaml.Node) []dateNamedPropertyMustUseDateFormatRoot {
			return dateNamedPropertyMustUseDateFormatSchemaRoot(
				oas.Get(media, "schema"),
				oas.JoinAll(path, dateNamedPropertyMustUseDateFormatContentKey, name.Value, "schema"),
				slices.Concat(dateNamedPropertyMustUseDateFormatOpenAPIExamples(root, media), extra),
			)
		},
	)
}

// dateNamedPropertyMustUseDateFormatHeaders returns the schema roots of response headers.
func dateNamedPropertyMustUseDateFormatHeaders(
	root *yaml.Node,
	response oas.Response,
) []dateNamedPropertyMustUseDateFormatRoot {
	return seq.FlatMapPairs(
		oas.Pairs(oas.Get(response.Node, "headers")),
		func(name, value *yaml.Node) []dateNamedPropertyMustUseDateFormatRoot {
			header := oas.Resolve(root, value)
			path := oas.JoinAll(response.Path, "headers", name.Value)
			contents := dateNamedPropertyMustUseDateFormatContent(root, header, path, nil)

			return slices.Concat(
				dateNamedPropertyMustUseDateFormatSchemaRoot(oas.Get(header, "schema"), oas.Join(path, "schema"), nil),
				seq.Map(
					slices.Values(contents),
					func(content dateNamedPropertyMustUseDateFormatRoot) dateNamedPropertyMustUseDateFormatRoot {
						return dateNamedPropertyMustUseDateFormatRoot{schema: content.schema, path: content.path}
					},
				),
			)
		},
	)
}

// dateNamedPropertyMustUseDateFormatOpenAPIExamples returns the example or the
// examples of a parameter or media type object.
func dateNamedPropertyMustUseDateFormatOpenAPIExamples(root, artifact *yaml.Node) []any {
	if !oas.IsMap(artifact) {
		return nil
	}

	if example := oas.Get(artifact, "example"); example != nil && !oas.IsNull(example) {
		return []any{dateNamedPropertyMustUseDateFormatConvert(example)}
	}

	return seq.FlatMapPairs(
		oas.Pairs(oas.Get(artifact, dateNamedPropertyMustUseDateFormatExamples)),
		func(_, entry *yaml.Node) []any {
			return []any{dateNamedPropertyMustUseDateFormatConvert(oas.Get(oas.Resolve(root, entry), "value"))}
		},
	)
}

// dateNamedPropertyMustUseDateFormatName applies the IBM name heuristic.
func dateNamedPropertyMustUseDateFormatName(name string, extra []string) bool {
	return !strings.HasSuffix(name, "depends_on") &&
		(slices.Contains(extra, name) ||
			dateNamedPropertyMustUseDateFormatExact(name) ||
			dateNamedPropertyMustUseDateFormatPattern(name))
}

// dateNamedPropertyMustUseDateFormatExact matches the fixed names.
func dateNamedPropertyMustUseDateFormatExact(name string) bool {
	return slices.Contains([]string{
		"created", "updated", "modified", "expired", "expires",
		"start_time", "end_time", "create_time", "created_time",
		"modify_time", "modified_time", "update_time",
	}, name)
}

// dateNamedPropertyMustUseDateFormatPattern matches the prefix, suffix and infix names.
func dateNamedPropertyMustUseDateFormatPattern(name string) bool {
	return strings.HasSuffix(name, "_at") ||
		strings.HasSuffix(name, "_on") ||
		strings.HasPrefix(name, "date_") ||
		strings.Contains(name, "_date_") ||
		strings.HasSuffix(name, "_date") ||
		strings.HasPrefix(name, "time_") ||
		strings.Contains(name, "_time_") ||
		strings.Contains(name, "timestamp")
}

// dateNamedPropertyMustUseDateFormatMonths matches full and abbreviated month names.
const dateNamedPropertyMustUseDateFormatMonths = `\b(Jan(uary)?|Feb(ruary)?|Mar(ch)?|Apr(il)?|May|Jun(e)?|Jul(y)?|Aug(ust)?|Sep(tember)?|Oct(ober)?|Nov(ember)?|Dec(ember)?)\b`

// dateNamedPropertyMustUseDateFormatDays matches full and abbreviated day names.
const dateNamedPropertyMustUseDateFormatDays = `\b(Mon(day)?|Tue(sday)?|Wed(nesday)?|Thu(rsday)?|Fri(day)?|Sat(urday)?|Sun(day)?)\b`

// dateNamedPropertyMustUseDateFormatExpressions are the compiled expressions of
// IBM's date value check. They are compiled once per run and passed down.
type dateNamedPropertyMustUseDateFormatExpressions struct {
	patterns []*regexp.Regexp
	months   *regexp.Regexp
	days     *regexp.Regexp
}

// dateNamedPropertyMustUseDateFormatCompile compiles IBM's date value expressions.
func dateNamedPropertyMustUseDateFormatCompile() dateNamedPropertyMustUseDateFormatExpressions {
	return dateNamedPropertyMustUseDateFormatExpressions{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(dateNamedPropertyMustUseDateFormatMonths),
			regexp.MustCompile(`\b\d{4}[./-](0?[1-9]|1[012])[./-]([012]?[1-9]|3[01])(\b|T)`),
			regexp.MustCompile(`\b([012]?[1-9]|3[01])[./-](0?[1-9]|1[012])[./-]\d{4}\b`),
			regexp.MustCompile(`\b(0?[1-9]|1[012])[./-]([012]?[1-9]|3[01])[./-]\d{4}\b`),
			regexp.MustCompile(`(\b|T)\d\d:\d\d:\d\d(\b[^:])`),
		},
		months: regexp.MustCompile(dateNamedPropertyMustUseDateFormatMonths),
		days:   regexp.MustCompile(dateNamedPropertyMustUseDateFormatDays),
	}
}

// dateNamedPropertyMustUseDateFormatValue is IBM's isDateBasedValue: it reports
// whether an example value looks like a date or a time.
func dateNamedPropertyMustUseDateFormatValue(
	value any,
	expressions dateNamedPropertyMustUseDateFormatExpressions,
) bool {
	text := dateNamedPropertyMustUseDateFormatText(value)
	_, isString := value.(string)
	stripped := expressions.days.ReplaceAllString(
		expressions.months.ReplaceAllString(text, ""),
		"",
	)

	lettered := isString && strings.ContainsAny(stripped, "abcdefghijklmnopqrstuvwxyz")

	return !lettered &&
		slices.ContainsFunc(expressions.patterns, func(pattern *regexp.Regexp) bool {
			return pattern.MatchString(text)
		})
}

// dateNamedPropertyMustUseDateFormatText converts a value to the text that
// JavaScript's RegExp.test would see.
func dateNamedPropertyMustUseDateFormatText(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case []any:
		return strings.Join(seq.Map(slices.Values(typed), dateNamedPropertyMustUseDateFormatText), ",")
	case map[string]any:
		return "[object Object]"
	default:
		return ""
	}
}

const (
	dateNamedPropertyMustUseDateFormatAnyOf = "anyOf"
)

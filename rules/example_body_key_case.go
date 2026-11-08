package rules

import (
	pathpkg "path"
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ExampleBodyKeyCase returns the extra-example-body-key-case rule.
//
// # What it checks
//
// Property names inside example bodies must follow the configured case style.
// The examples checked are the example and examples[*].value of every JSON media
// type in request bodies and responses, inline or under components. Nested
// objects and arrays are walked, and every key of an object whose schema
// declares properties must follow the configured convention. The schema of the
// media type is followed alongside the example, through $ref, items,
// properties, additionalProperties and the allOf, oneOf and anyOf members.
//
// # Why
//
// Examples are copied into client code, tests and documentation. An example
// that spells a property differently from the schema, or from the rest of the
// API, teaches the wrong convention and hides inconsistencies that
// schema-only naming rules cannot see.
//
// # Where it comes from
//
// apilint (body-capitalization).
//
// # Options
//
//   - casing (string or list, default "snake_case"): kebab-case, snake_case,
//     camelCase, PascalCase, SCREAMING_SNAKE_CASE, SCREAMING-KEBAB-CASE, dot.case or
//     lowercase. A key is accepted when it follows any listed convention, an
//     empty list disables the rule and an unknown value is reported.
//   - ignoreKeys (string or list, default none): exact keys or glob patterns
//     (path.Match syntax, for example givenName or x-*) that are never
//     reported. A malformed pattern is reported.
//
// # Bad
//
//	responses:
//	  "200":
//	    description: ok
//	    content:
//	      application/json:
//	        example:
//	          firstName: Ada
//
// # Good
//
//	example:
//	  first_name: Ada
//
// # Known false positives and limitations
//
// Keys of free-form maps (additionalProperties without properties), of objects
// with no schema, and of embedded documents are data rather than property
// names and are not checked. Names defined outside the API, such as OIDC claims
// like givenName, are excluded with ignoreKeys. Examples given as a string
// holding JSON, and $ref examples, are not inspected.
func ExampleBodyKeyCase() Rule {
	return New(Spec{
		Name:        "example-body-key-case",
		Description: "Property names in example bodies must follow the configured case style.",
		HowToFix:    "Rename the example property so that it follows the configured case style.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			casingOption("Case convention of the example property names", casing.Snake),
			{
				Name: exampleBodyKeyCaseIgnoreKeys,
				Description: "Keys, or glob patterns such as x-*, that are never reported, " +
					"for names defined outside the API such as OIDC claims.",
				Default: []string{},
			},
		},
		Check: WithCasing(checkExampleBodyKeyCase),
	})
}

type exampleBodyKeyCaseExample struct {
	node   *yaml.Node
	path   string
	schema *yaml.Node
}

// exampleBodyKeyCaseIgnoreKeys is the option that lists externally defined keys.
const exampleBodyKeyCaseIgnoreKeys = "ignoreKeys"

// exampleBodyKeyCaseMaxDepth bounds the walk through composed schemas.
const exampleBodyKeyCaseMaxDepth = 8

// exampleBodyKeyCaseContext is what a walk of one example needs.
type exampleBodyKeyCaseContext struct {
	root    *yaml.Node
	allowed casing.Set
	ignored []string
}

// checkExampleBodyKeyCase walks every example body and compares key casing.
func checkExampleBodyKeyCase(call *Call, allowed casing.Set) []Finding {
	ignored := call.Strings(exampleBodyKeyCaseIgnoreKeys)

	for _, pattern := range ignored {
		if _, err := pathpkg.Match(pattern, ""); err != nil {
			return One(Finding{
				Path:    "$",
				Message: "option " + exampleBodyKeyCaseIgnoreKeys + ": invalid pattern " + strconv.Quote(pattern),
			})
		}
	}

	walk := exampleBodyKeyCaseContext{root: call.Root, allowed: allowed, ignored: ignored}

	return seq.FlatMapSlice(exampleBodyKeyCaseExamples(call.Root), func(example exampleBodyKeyCaseExample) []Finding {
		return walk.value(example.node, example.path, example.schema, 0)
	})
}

// value reports the keys of one example value, following the schema alongside
// it. Keys are only checked on objects whose schema declares properties, so
// free-form maps and embedded documents are left alone.
func (c exampleBodyKeyCaseContext) value(node *yaml.Node, path string, schema *yaml.Node, depth int) []Finding {
	schema = oas.Resolve(c.root, schema)
	if schema == nil || depth > exampleBodyKeyCaseMaxDepth*exampleBodyKeyCaseMaxDepth {
		return nil
	}

	switch {
	case oas.IsMap(node):
		return c.object(node, path, schema, depth)
	case oas.IsSeq(node):
		return seq.FlatMapPairs(oas.Items(node), func(i int, item *yaml.Node) []Finding {
			return c.value(item, oas.Index(path, i), oas.Get(schema, "items"), depth+1)
		})
	default:
		return nil
	}
}

// object reports the keys of a mapping and walks into its values.
func (c exampleBodyKeyCaseContext) object(node *yaml.Node, path string, schema *yaml.Node, depth int) []Finding {
	properties := c.properties(schema, 0)
	additional := oas.Get(schema, "additionalProperties")

	return seq.FlatMapPairs(oas.Pairs(node), func(key, value *yaml.Node) []Finding {
		child, declared := properties[key.Value]
		if !declared {
			child = additional
		}

		own := When(len(properties) > 0 && !c.skipped(key.Value), func() Finding {
			return Finding{
				Node: key,
				Path: oas.Join(path, key.Value),
				Message: "example body property '" + key.Value + "' must be " + c.allowed.String() +
					" (for example '" + c.allowed.Convert(key.Value) + "')",
			}
		})
		if !oas.IsMap(child) {
			return own
		}

		return slices.Concat(own, c.value(value, oas.Join(path, key.Value), child, depth+1))
	})
}

// skipped reports whether the key is accepted whatever its casing.
func (c exampleBodyKeyCaseContext) skipped(key string) bool {
	return c.allowed.Matches(key) || slices.ContainsFunc(c.ignored, func(pattern string) bool {
		matched, err := pathpkg.Match(pattern, key)

		return err == nil && matched
	})
}

// properties returns the declared properties of a schema, including those of
// the members of allOf, oneOf and anyOf.
func (c exampleBodyKeyCaseContext) properties(schema *yaml.Node, depth int) map[string]*yaml.Node {
	merged := map[string]*yaml.Node{}
	schema = oas.Resolve(c.root, schema)

	if depth > exampleBodyKeyCaseMaxDepth || !oas.IsMap(schema) {
		return merged
	}

	for key, value := range oas.Pairs(oas.Get(schema, "properties")) {
		merged[key.Value] = value
	}

	for _, keyword := range []string{"allOf", "oneOf", "anyOf"} {
		for _, member := range oas.Items(oas.Get(schema, keyword)) {
			for name, value := range c.properties(member, depth+1) {
				if _, exists := merged[name]; !exists {
					merged[name] = value
				}
			}
		}
	}

	return merged
}

// exampleBodyKeyCaseExamples collects the example values of all JSON media types.
func exampleBodyKeyCaseExamples(root *yaml.Node) []exampleBodyKeyCaseExample {
	media := func(owner *yaml.Node, path string) []exampleBodyKeyCaseExample {
		return seq.FlatMapSlice(
			oas.MediaTypes(oas.Get(owner, "content"), oas.Join(path, "content")),
			func(media oas.MediaType) []exampleBodyKeyCaseExample {
				if !strings.Contains(strings.ToLower(media.Name), "json") {
					return nil
				}

				return exampleBodyKeyCaseMedia(media)
			},
		)
	}

	return slices.Concat(
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []exampleBodyKeyCaseExample {
			return slices.Concat(
				media(oas.Get(operation.Node, "requestBody"), oas.Join(operation.Path, "requestBody")),
				seq.FlatMapPairs(
					oas.Pairs(oas.Get(operation.Node, "responses")),
					func(key, response *yaml.Node) []exampleBodyKeyCaseExample {
						return media(response, oas.JoinAll(operation.Path, "responses", key.Value))
					},
				),
			)
		}),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "requestBodies")),
			func(key, body *yaml.Node) []exampleBodyKeyCaseExample {
				return media(body, oas.JoinAll("$.components.requestBodies", key.Value))
			},
		),
		seq.FlatMapPairs(
			oas.Pairs(oas.Dig(root, "components", "responses")),
			func(key, response *yaml.Node) []exampleBodyKeyCaseExample {
				return media(response, oas.JoinAll("$.components.responses", key.Value))
			},
		),
	)
}

// exampleBodyKeyCaseMedia returns the example and examples values of one media type.
func exampleBodyKeyCaseMedia(media oas.MediaType) []exampleBodyKeyCaseExample {
	schema := oas.Get(media.Node, "schema")

	return slices.Concat(
		exampleBodyKeyCaseOf(oas.Get(media.Node, "example"), oas.Join(media.Path, "example"), schema),
		seq.FlatMapPairs(
			oas.Pairs(oas.Get(media.Node, "examples")),
			func(name, entry *yaml.Node) []exampleBodyKeyCaseExample {
				return exampleBodyKeyCaseOf(
					oas.Get(entry, "value"),
					oas.Join(oas.JoinAll(media.Path, "examples", name.Value), "value"),
					schema,
				)
			},
		),
	)
}

// exampleBodyKeyCaseOf wraps an example value, or returns nothing when it is absent.
func exampleBodyKeyCaseOf(node *yaml.Node, path string, schema *yaml.Node) []exampleBodyKeyCaseExample {
	if node == nil {
		return nil
	}

	return []exampleBodyKeyCaseExample{{node: node, path: path, schema: schema}}
}

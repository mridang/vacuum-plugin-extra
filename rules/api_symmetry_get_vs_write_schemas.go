package rules

import (
	"cmp"
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Names, prefixes and limits used by this rule.
const (
	apiSymmetryGetVsWriteSchemasMaxDepth = 24
	apiSymmetryGetVsWriteSchemasSchemas  = "#/components/schemas/"
	apiSymmetryGetVsWriteSchemasRefSufix = "Reference"
	apiSymmetryGetVsWriteSchemasPrototyp = "Prototype"
	apiSymmetryGetVsWriteSchemasOKCode   = "200"
	apiSymmetryGetVsWriteSchemasObject   = "object"
	apiSymmetryGetVsWriteSchemasArray    = "array"
	apiSymmetryGetVsWriteSchemasString   = "string"
	apiSymmetryGetVsWriteSchemasNumber   = "number"
	apiSymmetryGetVsWriteSchemasInteger  = "integer"
	apiSymmetryGetVsWriteSchemasBoolean  = "boolean"
	apiSymmetryGetVsWriteSchemasFormat   = "format"
	apiSymmetryGetVsWriteSchemasAllOf    = "allOf"
	apiSymmetryGetVsWriteSchemasAnyOf    = "anyOf"
	apiSymmetryGetVsWriteSchemasJSON     = "application/json"
	apiSymmetryGetVsWriteSchemasAddProps = "additionalProperties"
	apiSymmetryGetVsWriteSchemasPatProps = "patternProperties"
	apiSymmetryGetVsWriteSchemasProps    = "properties"
	apiSymmetryGetVsWriteSchemasItems    = "items"
	apiSymmetryGetVsWriteSchemasUnknown  = "unknown"
	apiSymmetryGetVsWriteSchemasBinary   = "binary"
	apiSymmetryGetVsWriteSchemasByte     = "byte"
	apiSymmetryGetVsWriteSchemasFloat    = "float"
	apiSymmetryGetVsWriteSchemasDouble   = "double"
)

// apiSymmetryGetVsWriteSchemasSiblingRE matches the parameter segment of an item path.
func apiSymmetryGetVsWriteSchemasSiblingRE() *regexp.Regexp {
	return regexp.MustCompile(`^/\{[^{}/]+\}$`)
}

// APISymmetryGetVsWriteSchemas returns the extra-api-symmetry-get-vs-write-schemas rule.
//
// # What it checks
//
// For every resource-oriented path, a generic path that has a sibling made of
// the same path plus one trailing path parameter (/pets and /pets/{petId}), the
// canonical schema of the resource is the schema referenced by the JSON success
// response of the GET operation of the resource-specific sibling. The success
// response is the 200 response when present, otherwise the first 2xx response
// that has a JSON (application/json) schema; the first application/json media
// type is used. The canonical schema is found by the name of the
// #/components/schemas reference that the response schema is.
//
// When components.schemas also defines schemas named after the canonical name
// plus Summary, Prototype or Patch, each of these variants must be a graph
// fragment of the canonical schema. A graph fragment has the structure of the
// canonical schema but may omit properties, at any nesting level. The variant is
// reported (at its place in components.schemas) when:
//
//   - its schema type (array, object, boolean, string and its formats,
//     integer and number and their formats, otherwise unknown) differs from the
//     canonical type, unless the variant is checked as an allOf/oneOf/anyOf
//     member;
//   - it defines a property that does not exist on the canonical schema (exists
//     means present with the same type), or that exists but whose own schema is
//     not a graph fragment of the canonical property; properties marked
//     writeOnly are exempt in Prototype variants;
//   - it is an array whose items are not a graph fragment of the canonical
//     items, or has additionalProperties or patternProperties that are not a
//     graph fragment of the canonical ones (every variant pattern must exist on
//     the canonical schema);
//   - any allOf, oneOf or anyOf member of the variant is not a graph fragment of
//     the canonical schema.
//
// Looking things up in the canonical schema follows the IBM semantics: a
// property satisfies a constraint when the canonical schema itself, or any one
// of its allOf, oneOf or anyOf members, does. A canonical schema that is a
// reference to a schema named Xxx Reference (for instance WidgetReference) is
// replaced by the canonical schema named Xxx when that schema exists and does
// not contain a nested reference to the Reference schema.
//
// # Why
//
// A summary, a prototype used to create a resource, and a patch used to update
// it describe the same resource as the canonical schema. When a variant invents
// properties or changes the type of one, the same concept is modelled twice and
// the models drift apart.
//
// # Where it comes from
//
// IBM openapi-validator, ibm-api-symmetry (Variations of a resource schema
// should be graph fragments of the canonical schema).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get: {}
//	  /pets/{id}:
//	    get:
//	      responses:
//	        "200":
//	          content:
//	            application/json:
//	              schema:
//	                $ref: "#/components/schemas/Pet"
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        name: {type: string}
//	    PetPrototype:
//	      type: object
//	      properties:
//	        nickname: {type: string}
//
// # Good
//
//	components:
//	  schemas:
//	    PetPrototype:
//	      type: object
//	      properties:
//	        name: {type: string}
//
// # Known false positives and limitations
//
// The original runs on a fully dereferenced document and tracks references
// through the Spectral graph; this port follows local $ref values on demand,
// with a recursion bound for circular schemas (a cycle is assumed to be a graph
// fragment). External references are not followed. Writing a reference to the
// canonical schema under a different name than components.schemas keys, or
// inlining it, makes the canonical schema impossible to find and the resource
// is skipped, as in the original.
func APISymmetryGetVsWriteSchemas() Rule {
	return New(Spec{
		Name:        "api-symmetry-get-vs-write-schemas",
		Description: "Summary, Prototype and Patch variants of a resource schema must be graph fragments of its canonical schema.",
		HowToFix:    "Remove the property from the variant, add it to the canonical schema, or give it the canonical type.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkAPISymmetryGetVsWriteSchemas),
	})
}

// apiSymmetryGetVsWriteSchemasState carries the document while comparing schemas.
type apiSymmetryGetVsWriteSchemasState struct {
	root      *yaml.Node
	schemas   *yaml.Node
	writeOnly bool
}

// apiSymmetryGetVsWriteSchemasEntry is a key and value of a mapping.
type apiSymmetryGetVsWriteSchemasEntry struct {
	key   *yaml.Node
	value *yaml.Node
}

// apiSymmetryGetVsWriteSchemasEntries returns the entries of a mapping.
func apiSymmetryGetVsWriteSchemasEntries(mapping *yaml.Node) []apiSymmetryGetVsWriteSchemasEntry {
	return seq.FlatMapPairs(oas.Pairs(mapping), func(key, value *yaml.Node) []apiSymmetryGetVsWriteSchemasEntry {
		return []apiSymmetryGetVsWriteSchemasEntry{{key: key, value: value}}
	})
}

// apiSymmetryGetVsWriteSchemasMembers returns the members of the sequences
// under the given keywords of a schema.
func apiSymmetryGetVsWriteSchemasMembers(schema *yaml.Node, keywords ...string) []*yaml.Node {
	return seq.FlatMapSlice(keywords, func(keyword string) []*yaml.Node {
		return seq.FlatMapPairs(oas.Items(oas.Get(schema, keyword)), func(_ int, member *yaml.Node) []*yaml.Node {
			return []*yaml.Node{member}
		})
	})
}

// checkAPISymmetryGetVsWriteSchemas compares every variant with its canonical schema.
func checkAPISymmetryGetVsWriteSchemas(call *Call) []Finding {
	schemas := oas.Dig(call.Root, "components", "schemas")
	if !oas.IsMap(schemas) {
		return nil
	}

	return seq.FlatMapSlice(
		apiSymmetryGetVsWriteSchemasCanonicalNames(call.Root, schemas),
		func(name string) []Finding {
			canonical := oas.Get(schemas, name)

			return seq.FlatMapSlice(
				[]string{"Summary", apiSymmetryGetVsWriteSchemasPrototyp, "Patch"},
				func(variant string) []Finding {
					node := oas.Get(schemas, name+variant)
					if !oas.IsMap(node) {
						return nil
					}

					state := &apiSymmetryGetVsWriteSchemasState{
						root:      call.Root,
						schemas:   schemas,
						writeOnly: variant == apiSymmetryGetVsWriteSchemasPrototyp,
					}

					return When(!state.fragment(node, canonical, false, 0), func() Finding {
						return Finding{
							Node:    node,
							Path:    oas.JoinAll("$", "components", "schemas", name+variant),
							Message: "variant schema should be a graph fragment of the canonical schema " + name,
						}
					})
				},
			)
		},
	)
}

// apiSymmetryGetVsWriteSchemasCanonicalNames lists the canonical schema names of all resources.
func apiSymmetryGetVsWriteSchemasCanonicalNames(root, schemas *yaml.Node) []string {
	routes := apiSymmetryGetVsWriteSchemasRoutes(root)

	return seq.FlatMapSlice(routes, func(route string) []string {
		if strings.HasSuffix(strings.TrimSpace(route), "}") {
			return nil
		}

		specific := apiSymmetryGetVsWriteSchemasSibling(route, routes)
		if specific == "" {
			return nil
		}

		schema := apiSymmetryGetVsWriteSchemasSuccessSchema(
			root,
			oas.Get(oas.Get(oas.Get(root, "paths"), specific), "get"),
		)
		ref, ok := strings.CutPrefix(oas.Ref(schema), apiSymmetryGetVsWriteSchemasSchemas)
		if !ok || !oas.Has(schemas, ref) {
			return nil
		}

		return []string{ref}
	})
}

// apiSymmetryGetVsWriteSchemasRoutes returns the route keys of the paths object.
func apiSymmetryGetVsWriteSchemasRoutes(root *yaml.Node) []string {
	return seq.FlatMapPairs(oas.Pairs(oas.Get(root, "paths")), func(key, _ *yaml.Node) []string {
		return []string{key.Value}
	})
}

// apiSymmetryGetVsWriteSchemasSibling finds the resource-specific sibling of a generic route.
func apiSymmetryGetVsWriteSchemasSibling(route string, routes []string) string {
	suffix := apiSymmetryGetVsWriteSchemasSiblingRE()
	found := slices.IndexFunc(routes, func(candidate string) bool {
		rest, ok := strings.CutPrefix(candidate, route)

		return ok && suffix.MatchString(rest)
	})
	if found < 0 {
		return ""
	}

	return routes[found]
}

// apiSymmetryGetVsWriteSchemasSuccessSchema returns the JSON schema of the success response of an operation.
func apiSymmetryGetVsWriteSchemasSuccessSchema(root, operation *yaml.Node) *yaml.Node {
	responses := oas.Get(operation, "responses")
	if oas.Has(responses, apiSymmetryGetVsWriteSchemasOKCode) {
		return apiSymmetryGetVsWriteSchemasJSONSchema(
			oas.Resolve(root, oas.Get(responses, apiSymmetryGetVsWriteSchemasOKCode)),
		)
	}

	return cmp.Or(seq.FlatMapPairs(oas.Pairs(responses), func(code, response *yaml.Node) []*yaml.Node {
		if len(code.Value) != len(apiSymmetryGetVsWriteSchemasOKCode) || !oas.IsSuccess(code.Value) {
			return nil
		}

		return []*yaml.Node{apiSymmetryGetVsWriteSchemasJSONSchema(oas.Resolve(root, response))}
	})...)
}

// apiSymmetryGetVsWriteSchemasJSONSchema returns the schema of the first application/json media type.
func apiSymmetryGetVsWriteSchemasJSONSchema(response *yaml.Node) *yaml.Node {
	schemas := seq.FlatMapPairs(oas.Pairs(oas.Get(response, "content")), func(name, media *yaml.Node) []*yaml.Node {
		base, _, _ := strings.Cut(name.Value, ";")
		if Fold(strings.TrimSpace(base)) != apiSymmetryGetVsWriteSchemasJSON {
			return nil
		}

		return []*yaml.Node{oas.Get(media, "schema")}
	})
	if len(schemas) == 0 {
		return nil
	}

	return schemas[0]
}

// fragment reports whether variant is a graph fragment of canonical.
func (s *apiSymmetryGetVsWriteSchemasState) fragment(
	variantNode, canonical *yaml.Node,
	fromApplicator bool,
	depth int,
) bool {
	variant := oas.Resolve(s.root, variantNode)
	if depth > apiSymmetryGetVsWriteSchemasMaxDepth || !oas.IsMap(variant) {
		return true
	}

	sameType := fromApplicator || s.meets(canonical, func(schema *yaml.Node) bool {
		return s.schemaType(variant) == s.schemaType(schema)
	}, depth)

	return sameType &&
		s.fragmentNested(variant, canonical, depth) &&
		s.fragmentProperties(variant, canonical, depth) &&
		s.fragmentApplicators(variant, canonical, depth)
}

// fragmentNested checks items, additionalProperties and patternProperties.
func (s *apiSymmetryGetVsWriteSchemasState) fragmentNested(variant, canonical *yaml.Node, depth int) bool {
	items := oas.Get(variant, apiSymmetryGetVsWriteSchemasItems)
	additional := oas.Get(variant, apiSymmetryGetVsWriteSchemasAddProps)
	patterns := oas.Get(variant, apiSymmetryGetVsWriteSchemasPatProps)

	itemsFit := !oas.IsMap(items) || !s.has(variant, s.isArray, depth) ||
		s.meets(canonical, func(schema *yaml.Node) bool {
			return oas.IsMap(oas.Get(schema, apiSymmetryGetVsWriteSchemasItems)) &&
				s.fragment(items, oas.Get(schema, apiSymmetryGetVsWriteSchemasItems), false, depth+1)
		}, depth)
	additionalFits := !apiSymmetryGetVsWriteSchemasTruthy(additional) ||
		s.meets(canonical, func(schema *yaml.Node) bool {
			return s.additionalFragment(additional, oas.Get(schema, apiSymmetryGetVsWriteSchemasAddProps), depth)
		}, depth)
	patternsFit := !oas.IsMap(patterns) || s.meets(canonical, func(schema *yaml.Node) bool {
		return s.patternsFragment(patterns, oas.Get(schema, apiSymmetryGetVsWriteSchemasPatProps), depth)
	}, depth)

	return itemsFit && additionalFits && patternsFit
}

// apiSymmetryGetVsWriteSchemasTruthy reports whether an additionalProperties value is enabled.
func apiSymmetryGetVsWriteSchemasTruthy(n *yaml.Node) bool {
	return oas.IsMap(n) || oas.IsTrue(n)
}

// additionalFragment compares two additionalProperties values.
func (s *apiSymmetryGetVsWriteSchemasState) additionalFragment(variant, canonical *yaml.Node, depth int) bool {
	if !apiSymmetryGetVsWriteSchemasTruthy(canonical) {
		return false
	}

	if !oas.IsMap(variant) || !oas.IsMap(canonical) {
		return true
	}

	return s.fragment(variant, canonical, false, depth+1)
}

// patternsFragment compares two patternProperties maps.
func (s *apiSymmetryGetVsWriteSchemasState) patternsFragment(variant, canonical *yaml.Node, depth int) bool {
	if !oas.IsMap(canonical) {
		return false
	}

	canonicalEntries := apiSymmetryGetVsWriteSchemasEntries(canonical)

	return !slices.ContainsFunc(
		apiSymmetryGetVsWriteSchemasEntries(variant),
		func(entry apiSymmetryGetVsWriteSchemasEntry) bool {
			return !oas.Has(canonical, entry.key.Value) ||
				!slices.ContainsFunc(canonicalEntries, func(other apiSymmetryGetVsWriteSchemasEntry) bool {
					return s.fragment(entry.value, other.value, false, depth+1)
				})
		},
	)
}

// fragmentProperties checks that every variant property exists on the canonical schema.
func (s *apiSymmetryGetVsWriteSchemasState) fragmentProperties(variant, canonical *yaml.Node, depth int) bool {
	entries := apiSymmetryGetVsWriteSchemasEntries(oas.Get(variant, apiSymmetryGetVsWriteSchemasProps))

	return !slices.ContainsFunc(entries, func(entry apiSymmetryGetVsWriteSchemasEntry) bool {
		valid := s.meets(canonical, func(schema *yaml.Node) bool {
			existing := oas.Get(oas.Get(schema, apiSymmetryGetVsWriteSchemasProps), entry.key.Value)

			return oas.IsMap(oas.Resolve(s.root, existing)) && s.fragment(entry.value, existing, false, depth+1)
		}, depth)
		prop := oas.Resolve(s.root, entry.value)

		return !valid && (!s.writeOnly || !oas.IsTrue(oas.Get(prop, "writeOnly")))
	})
}

// fragmentApplicators checks the allOf, oneOf and anyOf members of the variant.
func (s *apiSymmetryGetVsWriteSchemasState) fragmentApplicators(variant, canonical *yaml.Node, depth int) bool {
	members := apiSymmetryGetVsWriteSchemasMembers(
		variant, apiSymmetryGetVsWriteSchemasAllOf, literalOneof, apiSymmetryGetVsWriteSchemasAnyOf)

	return !slices.ContainsFunc(members, func(member *yaml.Node) bool {
		return !s.fragment(member, canonical, true, depth+1)
	})
}

// meets reports whether the canonical schema, or any composed member, satisfies the predicate.
func (s *apiSymmetryGetVsWriteSchemasState) meets(
	canonical *yaml.Node,
	predicate func(*yaml.Node) bool,
	depth int,
) bool {
	if depth > apiSymmetryGetVsWriteSchemasMaxDepth {
		return true
	}

	schema := s.replaceReference(canonical)
	if !oas.IsMap(schema) {
		return false
	}

	members := apiSymmetryGetVsWriteSchemasMembers(
		schema, apiSymmetryGetVsWriteSchemasAllOf, literalOneof, apiSymmetryGetVsWriteSchemasAnyOf)

	return predicate(schema) || slices.ContainsFunc(members, func(member *yaml.Node) bool {
		return s.meets(member, predicate, depth+1)
	})
}

// replaceReference resolves a node, swapping a Reference schema for its canonical version.
func (s *apiSymmetryGetVsWriteSchemasState) replaceReference(node *yaml.Node) *yaml.Node {
	name, ok := strings.CutPrefix(oas.Ref(node), apiSymmetryGetVsWriteSchemasSchemas)
	if ok {
		base, found := strings.CutSuffix(name, apiSymmetryGetVsWriteSchemasRefSufix)
		candidate := oas.Get(s.schemas, base)

		if found && oas.IsMap(candidate) && !s.includesReference(candidate, name) {
			return candidate
		}
	}

	return oas.Resolve(s.root, node)
}

// includesReference reports whether the schema contains a reference to the named schema.
func (s *apiSymmetryGetVsWriteSchemasState) includesReference(node *yaml.Node, name string) bool {
	found, _ := s.reaches(node, name, nil)

	return found
}

// reaches is a depth-first search that threads the sorted list of followed references
// forward as an immutable value, so that cycles end. Content below a node is a tree,
// so only references need remembering. It returns the updated list.
func (s *apiSymmetryGetVsWriteSchemasState) reaches(
	node *yaml.Node,
	name string,
	followed []string,
) (bool, []string) {
	if node == nil {
		return false, followed
	}

	ref := oas.Ref(node)
	if ref == "" {
		return s.reachesAny(node.Content, name, followed)
	}

	if ref == apiSymmetryGetVsWriteSchemasSchemas+name {
		return true, followed
	}

	at, seen := slices.BinarySearch(followed, ref)
	if seen {
		return false, followed
	}

	return s.reaches(
		oas.Pointer(s.root, ref),
		name,
		slices.Concat(followed[:at], []string{ref}, followed[at:]),
	)
}

// reachesAny searches the nodes in order, threading the followed references through.
func (s *apiSymmetryGetVsWriteSchemasState) reachesAny(
	nodes []*yaml.Node,
	name string,
	followed []string,
) (bool, []string) {
	if len(nodes) == 0 {
		return false, followed
	}

	found, next := s.reaches(nodes[0], name, followed)
	if found {
		return true, next
	}

	return s.reachesAny(nodes[1:], name, next)
}

// has applies the IBM schemaHasConstraint composition semantics to a predicate.
func (s *apiSymmetryGetVsWriteSchemasState) has(node *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	schema := oas.Resolve(s.root, node)
	if depth > apiSymmetryGetVsWriteSchemasMaxDepth || !oas.IsMap(schema) {
		return false
	}

	return predicate(schema) ||
		slices.ContainsFunc(
			apiSymmetryGetVsWriteSchemasMembers(schema, apiSymmetryGetVsWriteSchemasAllOf),
			func(member *yaml.Node) bool {
				return s.has(member, predicate, depth+1)
			},
		) ||
		slices.ContainsFunc([]string{literalOneof, apiSymmetryGetVsWriteSchemasAnyOf}, func(keyword string) bool {
			return s.hasAll(oas.Get(schema, keyword), predicate, depth)
		})
}

// hasAll reports whether a non-empty list has only members that satisfy the predicate.
func (s *apiSymmetryGetVsWriteSchemasState) hasAll(
	members *yaml.Node,
	predicate func(*yaml.Node) bool,
	depth int,
) bool {
	return oas.IsSeq(members) && len(members.Content) > 0 &&
		!slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
			return !s.has(member, predicate, depth+1)
		})
}

// isArray reports whether a simple schema is an array schema.
func (s *apiSymmetryGetVsWriteSchemasState) isArray(schema *yaml.Node) bool {
	return oas.HasType(schema, apiSymmetryGetVsWriteSchemasArray) ||
		(oas.IsMap(oas.Get(schema, apiSymmetryGetVsWriteSchemasItems)) && !oas.Has(schema, "type"))
}

// isObjectLike reports whether a simple schema is an object schema.
func (s *apiSymmetryGetVsWriteSchemasState) isObjectLike(schema *yaml.Node) bool {
	if oas.HasType(schema, apiSymmetryGetVsWriteSchemasObject) {
		return true
	}

	return !oas.Has(schema, "type") &&
		(oas.IsMap(oas.Get(schema, apiSymmetryGetVsWriteSchemasProps)) ||
			apiSymmetryGetVsWriteSchemasTruthy(oas.Get(schema, apiSymmetryGetVsWriteSchemasAddProps)) ||
			oas.IsMap(oas.Get(schema, apiSymmetryGetVsWriteSchemasPatProps)))
}

// schemaType returns the logical type of a schema, as IBM getSchemaType does.
func (s *apiSymmetryGetVsWriteSchemasState) schemaType(schema *yaml.Node) string {
	switch {
	case s.has(schema, s.isArray, 0):
		return apiSymmetryGetVsWriteSchemasArray
	case s.has(schema, s.isObjectLike, 0):
		return apiSymmetryGetVsWriteSchemasObject
	case s.typed(schema, apiSymmetryGetVsWriteSchemasBoolean, ""):
		return apiSymmetryGetVsWriteSchemasBoolean
	case s.typed(schema, apiSymmetryGetVsWriteSchemasString, ""):
		return s.refine(
			schema,
			apiSymmetryGetVsWriteSchemasString,
			[]string{apiSymmetryGetVsWriteSchemasBinary, apiSymmetryGetVsWriteSchemasByte, "date", formatDateTime},
		)
	case s.typed(schema, apiSymmetryGetVsWriteSchemasInteger, ""):
		return s.refine(schema, apiSymmetryGetVsWriteSchemasInteger, []string{"int32", "int64"})
	case s.typed(schema, apiSymmetryGetVsWriteSchemasNumber, ""):
		return s.refine(
			schema,
			apiSymmetryGetVsWriteSchemasNumber,
			[]string{apiSymmetryGetVsWriteSchemasFloat, apiSymmetryGetVsWriteSchemasDouble},
		)
	default:
		return apiSymmetryGetVsWriteSchemasUnknown
	}
}

// refine narrows a base type by its format, or by enum for strings.
func (s *apiSymmetryGetVsWriteSchemasState) refine(schema *yaml.Node, base string, formats []string) string {
	found := slices.IndexFunc(formats, func(format string) bool {
		return s.typed(schema, base, format)
	})
	if found >= 0 {
		return base + ":" + formats[found]
	}

	if base == apiSymmetryGetVsWriteSchemasString && s.has(schema, func(n *yaml.Node) bool {
		return oas.HasType(n, base) && oas.IsSeq(oas.Get(n, "enum"))
	}, 0) {
		return "enumeration"
	}

	return base
}

// typed reports whether a schema has the type and, when given, the format.
func (s *apiSymmetryGetVsWriteSchemasState) typed(schema *yaml.Node, kind, format string) bool {
	return s.has(schema, func(n *yaml.Node) bool {
		return oas.HasType(n, kind) && (format == "" || oas.StringAt(n, apiSymmetryGetVsWriteSchemasFormat) == format)
	}, 0)
}

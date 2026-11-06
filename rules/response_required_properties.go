package rules

import (
	"hash/maphash"
	"iter"
	"reflect"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	responseRequiredPropertiesAnyOf = "anyOf"
	responseRequiredPropertiesEnum  = "enum"
)

const (
	responseRequiredPropertiesArrays = "arrays"
	responseRequiredPropertiesEnums  = "enums"
)

// ResponseRequiredProperties returns the extra-response-required-properties rule.
//
// # What it checks
//
// In the JSON response schemas of operations, and in components.responses,
// every property that is an array, and every property that is an enumeration
// (it has an enum), must be listed in the required list of the object that
// declares it. The check descends through nested objects, array items,
// additionalProperties and the allOf, anyOf and oneOf members, following local
// $refs. A property is also considered required when a sibling allOf member
// requires it.
//
// # Why
//
// A response should never leave a client guessing. An absent array forces every
// consumer to distinguish between "not sent" and "empty", and an absent
// enumeration value forces handling of a state the server never intends to
// produce. Returning an empty list and always returning the enumeration makes
// responses simpler to consume.
//
// # Where it comes from
//
// IBM openapi-validator, merging ibm-required-array-properties-in-response and
// ibm-required-enum-properties-in-response.
//
// # Options
//
//   - arrays (bool, default true): require array properties.
//   - enums (bool, default true): require enumeration properties.
//
// # Bad
//
//	components:
//	  schemas:
//	    ThingCollection:
//	      type: object
//	      properties:
//	        things:
//	          type: array
//	          items: {type: string}
//	        status:
//	          type: string
//	          enum: [on, off]
//
// # Good
//
//	components:
//	  schemas:
//	    ThingCollection:
//	      type: object
//	      required: [things, status]
//	      properties:
//	        things:
//	          type: array
//	          items: {type: string}
//	        status:
//	          type: string
//	          enum: [on, off]
//
// # Known false positives and limitations
//
// Only schemas reachable from a response are examined, but the finding points
// at the property where it is declared, which may be in components.schemas and
// shared with request bodies. Properties declared read-only or write-only are
// not treated specially.
func ResponseRequiredProperties() Rule {
	return New(Spec{
		Name:        "response-required-properties",
		Description: "Array and enum properties of response schemas must be listed as required.",
		HowToFix:    "Add the property to the required list of the schema, and always return it.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{Name: responseRequiredPropertiesArrays, Description: "Require array properties.", Default: true},
			{Name: responseRequiredPropertiesEnums, Description: "Require enum properties.", Default: true},
		},
		Check: OnDocument(checkResponseRequiredProperties),
	})
}

// responseRequiredPropertiesFollowLimit bounds how many references are followed in a row.
const responseRequiredPropertiesFollowLimit = 32

const (
	// responseRequiredPropertiesLastBit is the index of the last bit of a 64 bit hash.
	responseRequiredPropertiesLastBit = 63

	// responseRequiredPropertiesSpread is an odd multiplier that spreads pointer bits evenly.
	responseRequiredPropertiesSpread = 0x9E3779B97F4A7C15
)

// responseRequiredPropertiesEntry is one key and value of a mapping, or one index and member of a sequence.
type responseRequiredPropertiesEntry[K any] struct {
	key   K
	value *yaml.Node
}

// responseRequiredPropertiesEntries collects the entries of an iterator in order.
func responseRequiredPropertiesEntries[K any](pairs iter.Seq2[K, *yaml.Node]) []responseRequiredPropertiesEntry[K] {
	return slices.Collect(func(yield func(responseRequiredPropertiesEntry[K]) bool) {
		for key, value := range pairs {
			if !yield(responseRequiredPropertiesEntry[K]{key: key, value: value}) {
				return
			}
		}
	})
}

// responseRequiredPropertiesFold threads a state through every item and returns the final state.
func responseRequiredPropertiesFold[T, S any](items []T, state S, step func(S, T) S) S {
	if len(items) == 0 {
		return state
	}

	return responseRequiredPropertiesFold(items[1:], step(state, items[0]), step)
}

// responseRequiredPropertiesNode is a node of a persistent binary trie keyed by hash bits.
type responseRequiredPropertiesNode[K comparable] struct {
	hash  uint64
	keys  []K
	left  *responseRequiredPropertiesNode[K]
	right *responseRequiredPropertiesNode[K]
}

// responseRequiredPropertiesSet is an immutable set: adding returns a new set that shares structure.
type responseRequiredPropertiesSet[K comparable] struct {
	hash func(K) uint64
	root *responseRequiredPropertiesNode[K]
}

// has reports whether key is in the set.
func (s responseRequiredPropertiesSet[K]) has(key K) bool {
	return responseRequiredPropertiesNodeHas(s.root, s.hash(key), key, 0)
}

// with returns a set that also holds key.
func (s responseRequiredPropertiesSet[K]) with(key K) responseRequiredPropertiesSet[K] {
	return responseRequiredPropertiesSet[K]{
		hash: s.hash,
		root: responseRequiredPropertiesNodeAdd(s.root, s.hash(key), key, 0),
	}
}

// responseRequiredPropertiesGoesRight picks the branch for a hash at a depth, most significant bit first.
func responseRequiredPropertiesGoesRight(hash uint64, depth uint) bool {
	return hash>>(responseRequiredPropertiesLastBit-depth)&1 == 1
}

// responseRequiredPropertiesNodeHas looks key up below node.
func responseRequiredPropertiesNodeHas[K comparable](
	node *responseRequiredPropertiesNode[K],
	hash uint64,
	key K,
	depth uint,
) bool {
	switch {
	case node == nil:
		return false
	case node.hash == hash:
		return slices.Contains(node.keys, key)
	case responseRequiredPropertiesGoesRight(hash, depth):
		return responseRequiredPropertiesNodeHas(node.right, hash, key, depth+1)
	default:
		return responseRequiredPropertiesNodeHas(node.left, hash, key, depth+1)
	}
}

// responseRequiredPropertiesNodeAdd returns a copy of the path to key, sharing everything else.
func responseRequiredPropertiesNodeAdd[K comparable](
	node *responseRequiredPropertiesNode[K],
	hash uint64,
	key K,
	depth uint,
) *responseRequiredPropertiesNode[K] {
	switch {
	case node == nil:
		return &responseRequiredPropertiesNode[K]{hash: hash, keys: []K{key}}
	case node.hash == hash && slices.Contains(node.keys, key):
		return node
	case node.hash == hash:
		return &responseRequiredPropertiesNode[K]{
			hash: hash, keys: slices.Concat(node.keys, []K{key}), left: node.left, right: node.right,
		}
	case responseRequiredPropertiesGoesRight(hash, depth):
		return &responseRequiredPropertiesNode[K]{
			hash: node.hash, keys: node.keys, left: node.left,
			right: responseRequiredPropertiesNodeAdd(node.right, hash, key, depth+1),
		}
	default:
		return &responseRequiredPropertiesNode[K]{
			hash: node.hash, keys: node.keys, right: node.right,
			left: responseRequiredPropertiesNodeAdd(node.left, hash, key, depth+1),
		}
	}
}

// responseRequiredPropertiesList is an immutable list of findings, newest first.
type responseRequiredPropertiesList struct {
	head Finding
	tail *responseRequiredPropertiesList
}

// walk yields the findings newest first.
func (l *responseRequiredPropertiesList) walk(yield func(Finding) bool) bool {
	if l == nil {
		return true
	}

	return yield(l.head) && l.tail.walk(yield)
}

// ordered returns the findings in the order they were found.
func (l *responseRequiredPropertiesList) ordered() []Finding {
	newestFirst := slices.Collect(func(yield func(Finding) bool) {
		l.walk(yield)
	})

	return slices.Collect(func(yield func(Finding) bool) {
		for _, finding := range slices.Backward(newestFirst) {
			if !yield(finding) {
				return
			}
		}
	})
}

// responseRequiredPropertiesState is what a walk carries forward: the schemas already walked, the
// property paths already reported and the findings so far. A schema reachable along several paths
// must be walked and reported once, which is why the walk threads state.
type responseRequiredPropertiesState struct {
	visited  responseRequiredPropertiesSet[*yaml.Node]
	reported responseRequiredPropertiesSet[string]
	findings *responseRequiredPropertiesList
}

// withVisited returns the state after walking schema.
func (s responseRequiredPropertiesState) withVisited(schema *yaml.Node) responseRequiredPropertiesState {
	return responseRequiredPropertiesState{
		visited: s.visited.with(schema), reported: s.reported, findings: s.findings,
	}
}

// withFinding returns the state after reporting finding.
func (s responseRequiredPropertiesState) withFinding(finding Finding) responseRequiredPropertiesState {
	return responseRequiredPropertiesState{
		visited:  s.visited,
		reported: s.reported.with(finding.Path),
		findings: &responseRequiredPropertiesList{head: finding, tail: s.findings},
	}
}

// responseRequiredPropertiesWalker walks the response schemas of a document.
type responseRequiredPropertiesWalker struct {
	root   *yaml.Node
	arrays bool
	enums  bool
}

// checkResponseRequiredProperties walks every response schema.
func checkResponseRequiredProperties(call *Call) []Finding {
	seed := maphash.MakeSeed()
	walker := responseRequiredPropertiesWalker{
		root:   call.Root,
		arrays: call.Bool(responseRequiredPropertiesArrays),
		enums:  call.Bool(responseRequiredPropertiesEnums),
	}
	empty := responseRequiredPropertiesState{
		visited: responseRequiredPropertiesSet[*yaml.Node]{
			hash: func(node *yaml.Node) uint64 {
				return uint64(reflect.ValueOf(node).Pointer()) * responseRequiredPropertiesSpread
			},
		},
		reported: responseRequiredPropertiesSet[string]{
			hash: func(path string) uint64 {
				return maphash.String(seed, path)
			},
		},
	}

	afterOperations := responseRequiredPropertiesFold(
		slices.Collect(oas.Operations(call.Root)),
		empty,
		func(state responseRequiredPropertiesState, operation oas.Operation) responseRequiredPropertiesState {
			return responseRequiredPropertiesFold(
				responseRequiredPropertiesEntries(oas.Pairs(oas.Get(operation.Node, "responses"))),
				state,
				func(inner responseRequiredPropertiesState, code responseRequiredPropertiesEntry[*yaml.Node]) responseRequiredPropertiesState {
					return walker.response(inner, code.value, oas.JoinAll(operation.Path, "responses", code.key.Value))
				},
			)
		},
	)

	final := responseRequiredPropertiesFold(
		responseRequiredPropertiesEntries(oas.Pairs(oas.Dig(call.Root, "components", "responses"))),
		afterOperations,
		func(state responseRequiredPropertiesState, named responseRequiredPropertiesEntry[*yaml.Node]) responseRequiredPropertiesState {
			return walker.response(state, named.value, oas.JoinAll("$.components.responses", named.key.Value))
		},
	)

	return final.findings.ordered()
}

// response walks the JSON schemas of one response object.
func (w responseRequiredPropertiesWalker) response(
	state responseRequiredPropertiesState,
	response *yaml.Node,
	path string,
) responseRequiredPropertiesState {
	resolved := oas.Resolve(w.root, response)

	return responseRequiredPropertiesFold(
		oas.MediaTypes(oas.Get(resolved, "content"), oas.Join(path, "content")),
		state,
		func(inner responseRequiredPropertiesState, media oas.MediaType) responseRequiredPropertiesState {
			if !strings.Contains(strings.ToLower(media.Name), "json") {
				return inner
			}

			return w.schema(inner, oas.Get(media.Node, "schema"), oas.Join(media.Path, "schema"), nil)
		},
	)
}

// follow resolves references and returns the target with its own JSONPath.
func (w responseRequiredPropertiesWalker) follow(node *yaml.Node, path string) (*yaml.Node, string) {
	return w.followWithin(node, path, responseRequiredPropertiesFollowLimit)
}

// followWithin follows at most remaining references.
func (w responseRequiredPropertiesWalker) followWithin(
	node *yaml.Node,
	path string,
	remaining int,
) (*yaml.Node, string) {
	ref := oas.Ref(node)
	target := oas.Pointer(w.root, ref)

	if remaining == 0 || ref == "" || target == nil {
		return node, path
	}

	return w.followWithin(target, responseRequiredPropertiesRefPath(ref), remaining-1)
}

// schema walks a schema; inherited holds names required by enclosing allOf siblings.
func (w responseRequiredPropertiesWalker) schema(
	state responseRequiredPropertiesState,
	node *yaml.Node,
	path string,
	inherited []string,
) responseRequiredPropertiesState {
	schema, at := w.follow(node, path)
	if !oas.IsMap(schema) || state.visited.has(schema) {
		return state
	}

	required := slices.Concat(
		inherited,
		oas.Strings(oas.Get(schema, "required")),
		seq.FlatMapPairs(oas.Items(oas.Get(schema, "allOf")), func(_ int, member *yaml.Node) []string {
			resolved, _ := w.follow(member, "")

			return oas.Strings(oas.Get(resolved, "required"))
		}),
	)

	return w.nested(w.properties(state.withVisited(schema), schema, at, required), schema, at, required)
}

// nested walks the sub schemas of a schema.
func (w responseRequiredPropertiesWalker) nested(
	state responseRequiredPropertiesState,
	schema *yaml.Node,
	at string,
	required []string,
) responseRequiredPropertiesState {
	afterItems := w.schema(state, oas.Get(schema, "items"), oas.Join(at, "items"), nil)
	afterAdditional := w.schema(
		afterItems, oas.Get(schema, "additionalProperties"), oas.Join(at, "additionalProperties"), nil,
	)

	afterAllOf := responseRequiredPropertiesFold(
		responseRequiredPropertiesEntries(oas.Items(oas.Get(schema, "allOf"))),
		afterAdditional,
		func(inner responseRequiredPropertiesState, member responseRequiredPropertiesEntry[int]) responseRequiredPropertiesState {
			return w.schema(inner, member.value, oas.Index(oas.Join(at, "allOf"), member.key), required)
		},
	)

	return responseRequiredPropertiesFold(
		[]string{responseRequiredPropertiesAnyOf, literalOneof},
		afterAllOf,
		func(inner responseRequiredPropertiesState, key string) responseRequiredPropertiesState {
			return responseRequiredPropertiesFold(
				responseRequiredPropertiesEntries(oas.Items(oas.Get(schema, key))),
				inner,
				func(deep responseRequiredPropertiesState, member responseRequiredPropertiesEntry[int]) responseRequiredPropertiesState {
					return w.schema(deep, member.value, oas.Index(oas.Join(at, key), member.key), nil)
				},
			)
		},
	)
}

// properties checks the properties of a schema and walks into them.
func (w responseRequiredPropertiesWalker) properties(
	state responseRequiredPropertiesState,
	schema *yaml.Node,
	at string,
	required []string,
) responseRequiredPropertiesState {
	base := oas.Join(at, "properties")

	return responseRequiredPropertiesFold(
		responseRequiredPropertiesEntries(oas.Pairs(oas.Get(schema, "properties"))),
		state,
		func(inner responseRequiredPropertiesState, property responseRequiredPropertiesEntry[*yaml.Node]) responseRequiredPropertiesState {
			path := oas.Join(base, property.key.Value)
			target, _ := w.follow(property.value, path)
			kind := w.kind(target)

			if kind == "" || slices.Contains(required, property.key.Value) || inner.reported.has(path) {
				return w.schema(inner, property.value, path, nil)
			}

			return w.schema(inner.withFinding(Finding{
				Node:    property.key,
				Path:    path,
				Message: kind + " property '" + property.key.Value + "' in a response schema must be listed in required",
			}), property.value, path, nil)
		},
	)
}

// kind names the kind of property that must be required, or returns an empty string.
func (w responseRequiredPropertiesWalker) kind(property *yaml.Node) string {
	switch {
	case w.arrays && oas.HasType(property, "array"):
		return "array"
	case w.enums && oas.Has(property, responseRequiredPropertiesEnum):
		return responseRequiredPropertiesEnum
	default:
		return ""
	}
}

// responseRequiredPropertiesRefPath converts a local reference into a JSONPath.
func responseRequiredPropertiesRefPath(ref string) string {
	tokens := seq.Map(strings.SplitSeq(strings.TrimPrefix(ref, "#/"), "/"), func(token string) string {
		return strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
	})

	return oas.JoinAll("$", tokens...)
}

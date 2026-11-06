package rules

import (
	"encoding/json"
	"maps"
	"math"
	"reflect"
	"slices"
	"strconv"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// illogicalCompositionKeywordsMinMembers is the smallest number of members a
// composition needs to be meaningful.
const illogicalCompositionKeywordsMinMembers = 2

// IllogicalCompositionKeywords returns the extra-illogical-composition-keywords
// rule.
//
// # What it checks
//
// A faithful port of Redocly CLI's no-illogical-composition-keywords. For every
// schema in the document (components, parameters, headers, request and response
// bodies and everything nested in them):
//
//   - allOf, anyOf and oneOf: a oneOf or anyOf with fewer than two members is
//     reported unless the schema declares a discriminator; an allOf with no
//     member, or with one member when the schema has no other keyword and the
//     member does not resolve to a schema that declares a discriminator (a
//     subtype declaration), is reported;
//   - members that are the empty schema {} (they match any value);
//   - members that are deep-equal to an earlier member of the same keyword
//     (the finding is on the later member and names the first position);
//   - anyOf and oneOf with a discriminator: every member must be a $ref (or
//     declare $id), because a discriminator selects a schema by its component
//     name and cannot select an inline member;
//   - oneOf: the schema is nullable (nullable: true, or null in type) and a
//     resolved member also accepts null;
//   - oneOf: every pair of members that a single value can match at once. The
//     comparison is deliberately narrow, exactly as in Redocly: it reads type,
//     nullable, enum, const, properties, required and additionalProperties
//     false, and gives up on a pair when either member uses any other
//     constraint keyword (not, pattern, minimum, nested allOf and so on; x-
//     keywords and annotations are ignored). It reports shared null, shared
//     enum or const values, a discriminator property that is not required in
//     both members (except on OpenAPI 3.2, where spec-discriminator-defaultMapping
//     covers it), optional properties that would tell the members apart, shared
//     properties without constraints that exclude each other, and shared types
//     (integer against number overlaps when the number side allows an
//     integer).
//
// An ambiguous oneOf has no single answer: validators, code generators and
// documentation disagree on which branch was meant.
//
// # Where it comes from
//
// Redocly CLI, rule no-illogical-composition-keywords
// (packages/core/src/rules/oas3/no-illogical-composition-keywords.ts).
//
// # Options
//
// None.
//
// # Bad
//
//	components:
//	  schemas:
//	    PhotoUrl:
//	      type: [string, "null"]
//	    MenuItemPhoto:
//	      oneOf:
//	        - $ref: "#/components/schemas/PhotoUrl"
//	        - type: "null"
//
// # Good
//
//	components:
//	  schemas:
//	    PhotoUrl:
//	      type: string
//	    MenuItemPhoto:
//	      oneOf:
//	        - $ref: "#/components/schemas/PhotoUrl"
//	        - type: "null"
//
// # Known false positives and limitations
//
// $ref targets are followed only when they are local to the document; a member
// that points to another file is not compared. Redocly's source tracking of
// external files has no equivalent here. Boolean schemas used as members are
// skipped. Duplicate detection compares the raw members, like the original, so
// two different $ref strings that resolve to equal schemas are reported by the
// overlap check instead.
func IllogicalCompositionKeywords() Rule {
	return New(Spec{
		Name:        "illogical-composition-keywords",
		Description: "allOf, anyOf and oneOf must combine schemas that a value can actually resolve against.",
		HowToFix:    "Give the composition at least two distinct, non-empty, mutually exclusive members, or use the schema directly.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check:       PerSchema(checkIllogicalCompositionKeywords),
	})
}

// illogicalCompositionKeywordsUnderstood lists the keywords a schema may carry
// without stopping the overlap comparison.
func illogicalCompositionKeywordsUnderstood() []string {
	return strings.Fields(
		"type format enum const nullable properties required additionalProperties discriminator " +
			"$ref $id $schema $anchor $comment title description default example examples " +
			"deprecated readOnly writeOnly externalDocs xml",
	)
}

// Literals used by this rule.
const (
	illogicalCompositionKeywordsAllOf    = "allOf"
	illogicalCompositionKeywordsAnyOf    = "anyOf"
	illogicalCompositionKeywordsNull     = "null"
	illogicalCompositionKeywordsInteger  = "integer"
	illogicalCompositionKeywordsNumber   = "number"
	illogicalCompositionKeywordsRequired = "required"
	illogicalCompositionKeywordsProps    = "properties"
	illogicalCompositionKeywordsNullable = "nullable"
	illogicalCompositionKeywordsXPrefix  = "x-"
	illogicalCompositionKeywords32       = "3.2"
	illogicalCompositionKeywordsNullTag  = "!!null"
	illogicalCompositionKeywordsBoolTag  = "!!bool"
)

func checkIllogicalCompositionKeywords(call *Call, schema oas.Schema) []Finding {
	keywords := []string{illogicalCompositionKeywordsAllOf, illogicalCompositionKeywordsAnyOf, literalOneof}

	return seq.FlatMapSlice(keywords, func(keyword string) []Finding {
		return illogicalCompositionKeywordsCheckList(call, schema, keyword, oas.Get(schema.Node, keyword))
	})
}

func illogicalCompositionKeywordsCheckList(call *Call, schema oas.Schema, keyword string, list *yaml.Node) []Finding {
	if !oas.IsSeq(list) {
		return nil
	}

	path := oas.Join(schema.Path, keyword)
	hasDiscriminator := oas.IsMap(oas.Get(schema.Node, "discriminator"))

	return slices.Concat(
		illogicalCompositionKeywordsSize(call.Root, schema, keyword, list, hasDiscriminator),
		illogicalCompositionKeywordsEmpty(list, path, keyword),
		illogicalCompositionKeywordsDuplicates(list, path, keyword),
		illogicalCompositionKeywordsInline(list, path, keyword, hasDiscriminator),
		illogicalCompositionKeywordsOneOf(call, schema, keyword, list, path),
	)
}

// illogicalCompositionKeywordsOneOf holds the checks that only apply to oneOf.
func illogicalCompositionKeywordsOneOf(
	call *Call,
	schema oas.Schema,
	keyword string,
	list *yaml.Node,
	path string,
) []Finding {
	if keyword != literalOneof {
		return nil
	}

	return slices.Concat(
		illogicalCompositionKeywordsNullableParent(call.Root, schema.Node, list, path),
		illogicalCompositionKeywordsAmbiguous(call, schema, list, path),
	)
}

func illogicalCompositionKeywordsSize(
	root *yaml.Node,
	schema oas.Schema,
	keyword string,
	list *yaml.Node,
	hasDiscriminator bool,
) []Finding {
	return When(
		illogicalCompositionKeywordsTooSmall(root, schema, keyword, list, hasDiscriminator),
		func() Finding {
			return Finding{
				Node:    list,
				Path:    oas.Join(schema.Path, keyword),
				Message: "`" + keyword + "` should have at least two schemas." + illogicalCompositionKeywordsHint(list),
			}
		},
	)
}

// illogicalCompositionKeywordsHint suggests using a lone member directly.
func illogicalCompositionKeywordsHint(list *yaml.Node) string {
	if len(list.Content) == 1 {
		return " Use the schema directly instead."
	}

	return ""
}

// illogicalCompositionKeywordsTooSmall reports a composition with too few members.
func illogicalCompositionKeywordsTooSmall(
	root *yaml.Node,
	schema oas.Schema,
	keyword string,
	list *yaml.Node,
	hasDiscriminator bool,
) bool {
	size := len(list.Content)

	if keyword != illogicalCompositionKeywordsAllOf {
		return size < illogicalCompositionKeywordsMinMembers && !hasDiscriminator
	}

	if size == 0 {
		return true
	}

	return size == 1 &&
		len(schema.Node.Content)/keyValuePairWidth == 1 &&
		!oas.IsMap(oas.Get(oas.Resolve(root, list.Content[0]), "discriminator"))
}

func illogicalCompositionKeywordsEmpty(list *yaml.Node, path, keyword string) []Finding {
	return seq.FlatMapIndexed(list.Content, func(i int, member *yaml.Node) []Finding {
		return When(oas.IsMap(member) && len(member.Content) == 0, func() Finding {
			return Finding{
				Node:    member,
				Path:    oas.Index(path, i),
				Message: "Schema in `" + keyword + "` is empty, so it matches any value.",
			}
		})
	})
}

func illogicalCompositionKeywordsInline(list *yaml.Node, path, keyword string, hasDiscriminator bool) []Finding {
	if !hasDiscriminator || keyword == illogicalCompositionKeywordsAllOf {
		return nil
	}

	return seq.FlatMapIndexed(list.Content, func(i int, member *yaml.Node) []Finding {
		return When(
			oas.Ref(member) == "" && (!oas.IsMap(member) || !oas.Has(member, "$id")),
			func() Finding {
				return Finding{
					Node: member,
					Path: oas.Index(path, i),
					Message: "Schema in `" + keyword +
						"` is inline, so the `discriminator` cannot select it. Use a `$ref` to a named schema.",
				}
			},
		)
	})
}

func illogicalCompositionKeywordsDuplicates(list *yaml.Node, path, keyword string) []Finding {
	values := seq.Map(slices.Values(list.Content), illogicalCompositionKeywordsValue)

	return seq.FlatMapIndexed(values, func(i int, value any) []Finding {
		first := slices.IndexFunc(values, func(other any) bool {
			return reflect.DeepEqual(other, value)
		})

		return When(i > 0 && first < i, func() Finding {
			return Finding{
				Node:    list.Content[i],
				Path:    oas.Index(path, i),
				Message: "Schema in `" + keyword + "` duplicates the schema at position " + strconv.Itoa(first+1) + ".",
			}
		})
	})
}

// illogicalCompositionKeywordsValue converts a YAML node into plain Go values
// so that two nodes can be compared deeply.
func illogicalCompositionKeywordsValue(input *yaml.Node) any {
	node := oas.Unwrap(input)

	switch {
	case node == nil:
		return nil
	case oas.IsMap(node):
		return maps.Collect(func(yield func(string, any) bool) {
			for key, value := range oas.Pairs(node) {
				if !yield(key.Value, illogicalCompositionKeywordsValue(value)) {
					return
				}
			}
		})
	case oas.IsSeq(node):
		return illogicalCompositionKeywordsNonNil(
			seq.Map(slices.Values(node.Content), illogicalCompositionKeywordsValue),
		)
	default:
		return illogicalCompositionKeywordsScalar(node)
	}
}

// illogicalCompositionKeywordsNonNil returns values, or an empty non-nil slice
// when values is nil, so that it encodes as [] and compares deeply as empty.
func illogicalCompositionKeywordsNonNil(values []any) []any {
	if values == nil {
		return []any{}
	}

	return values
}

func illogicalCompositionKeywordsScalar(node *yaml.Node) any {
	switch node.Tag {
	case illogicalCompositionKeywordsNullTag:
		return nil
	case illogicalCompositionKeywordsBoolTag:
		return node.Value == illogicalCompositionKeywordsTrue
	case "!!int", "!!float":
		if number, ok := oas.Number(node); ok {
			return number
		}
	}

	return node.Value
}

func illogicalCompositionKeywordsNullableParent(root, parent, list *yaml.Node, path string) []Finding {
	memberAcceptsNull := func(member *yaml.Node) bool {
		resolved := oas.Resolve(root, member)

		return oas.IsMap(resolved) && illogicalCompositionKeywordsAllowsNull(resolved)
	}

	return When(
		illogicalCompositionKeywordsAllowsNull(parent) && slices.ContainsFunc(list.Content, memberAcceptsNull),
		func() Finding {
			return Finding{
				Node: list,
				Path: path,
				Message: "The schema and a schema in `oneOf` both accept `null`, " +
					"so nothing decides which one applies to a null value.",
			}
		},
	)
}

// illogicalCompositionKeywordsMap returns node when it is a mapping, else nil.
func illogicalCompositionKeywordsMap(node *yaml.Node) *yaml.Node {
	if !oas.IsMap(node) {
		return nil
	}

	return node
}

func illogicalCompositionKeywordsAmbiguous(call *Call, schema oas.Schema, list *yaml.Node, path string) []Finding {
	discriminator := illogicalCompositionKeywordsMap(oas.Get(schema.Node, "discriminator"))
	gap := !strings.HasPrefix(oas.Version(call.Root), illogicalCompositionKeywords32)
	resolved := seq.Map(slices.Values(list.Content), func(member *yaml.Node) *yaml.Node {
		return illogicalCompositionKeywordsMap(oas.Resolve(call.Root, member))
	})
	lefts := resolved[:max(len(resolved)-1, 0)]

	return seq.FlatMapIndexed(lefts, func(left int, leftNode *yaml.Node) []Finding {
		return seq.FlatMapIndexed(resolved[left+1:], func(offset int, rightNode *yaml.Node) []Finding {
			right := left + 1 + offset
			reason := illogicalCompositionKeywordsPairReason(
				call.Root, list, left, right, leftNode, rightNode, discriminator, gap,
			)

			return When(reason != "", func() Finding {
				return Finding{
					Node: list,
					Path: path,
					Message: "Schemas in `oneOf` must be mutually exclusive. Found overlapping schemas: " +
						illogicalCompositionKeywordsDescribe(list.Content[left], left) + " and " +
						illogicalCompositionKeywordsDescribe(list.Content[right], right) + ". " + reason,
				}
			})
		})
	})
}

// illogicalCompositionKeywordsPairReason explains why two members overlap, or
// returns the empty string when they do not or cannot be compared.
func illogicalCompositionKeywordsPairReason(
	root, list *yaml.Node,
	left, right int,
	leftNode, rightNode, discriminator *yaml.Node,
	gap bool,
) string {
	if leftNode == nil || rightNode == nil {
		return ""
	}

	if reflect.DeepEqual(
		illogicalCompositionKeywordsValue(list.Content[left]),
		illogicalCompositionKeywordsValue(list.Content[right]),
	) {
		return ""
	}

	return illogicalCompositionKeywordsReason(root, leftNode, rightNode, discriminator, gap)
}

func illogicalCompositionKeywordsDescribe(member *yaml.Node, index int) string {
	if ref := oas.Ref(member); ref != "" {
		return "`" + ref + "`"
	}

	if title := oas.StringAt(member, "title"); title != "" {
		return "`" + title + "`"
	}

	return "schema at position " + strconv.Itoa(index+1)
}

func illogicalCompositionKeywordsReason(root, left, right, discriminator *yaml.Node, gap bool) string {
	if illogicalCompositionKeywordsUnsupported(left) || illogicalCompositionKeywordsUnsupported(right) {
		return ""
	}

	if illogicalCompositionKeywordsExclusive(left, right) {
		return ""
	}

	if illogicalCompositionKeywordsAllowsNull(left) && illogicalCompositionKeywordsAllowsNull(right) {
		return "Both schemas accept `null`."
	}

	leftValues, leftOK := illogicalCompositionKeywordsAllowed(left)
	rightValues, rightOK := illogicalCompositionKeywordsAllowed(right)
	if leftOK && rightOK {
		shared := illogicalCompositionKeywordsIntersect(leftValues, rightValues)

		return "Both schemas allow the values " + illogicalCompositionKeywordsJSON(shared) + "."
	}

	if discriminator != nil {
		if !gap {
			return ""
		}

		return illogicalCompositionKeywordsGap(left, right, oas.StringAt(discriminator, "propertyName"))
	}

	if reason := illogicalCompositionKeywordsPropertyOverlap(root, left, right); reason != "" {
		return reason
	}

	if !oas.Has(left, illogicalCompositionKeywordsProps) && !oas.Has(right, illogicalCompositionKeywordsProps) {
		if shared := illogicalCompositionKeywordsSharedTypes(left, right); len(shared) > 0 {
			return "Both schemas accept " + illogicalCompositionKeywordsQuote(shared) + "."
		}
	}

	return ""
}

func illogicalCompositionKeywordsJSON(values []any) string {
	encoded, err := json.Marshal(values)
	if err != nil {
		return "[]"
	}

	return string(encoded)
}

func illogicalCompositionKeywordsIntersect(left, right []any) []any {
	return illogicalCompositionKeywordsNonNil(seq.Filter(slices.Values(left), func(value any) bool {
		return slices.ContainsFunc(right, func(other any) bool { return reflect.DeepEqual(value, other) })
	}))
}

func illogicalCompositionKeywordsQuote(names []string) string {
	return strings.Join(seq.Map(slices.Values(names), func(name string) string { return "`" + name + "`" }), ", ")
}

func illogicalCompositionKeywordsGap(left, right *yaml.Node, property string) string {
	if property == "" {
		return ""
	}

	if illogicalCompositionKeywordsRequires(left, property) && illogicalCompositionKeywordsRequires(right, property) {
		return ""
	}

	declared := oas.Get(oas.Get(left, illogicalCompositionKeywordsProps), property) != nil &&
		oas.Get(oas.Get(right, illogicalCompositionKeywordsProps), property) != nil
	if !declared {
		return "Declare `" + property + "` in every schema and add it to `required`; " +
			"the `discriminator` cannot read a property a value may omit."
	}

	return "Add `" + property + "` to `required` in every schema; " +
		"the `discriminator` cannot read a property a value may omit."
}

func illogicalCompositionKeywordsRequires(schema *yaml.Node, property string) bool {
	return slices.Contains(oas.Strings(oas.Get(schema, illogicalCompositionKeywordsRequired)), property)
}

func illogicalCompositionKeywordsUnsupported(schema *yaml.Node) bool {
	understood := illogicalCompositionKeywordsUnderstood()
	for key := range oas.Pairs(schema) {
		if !slices.Contains(understood, key.Value) &&
			!strings.HasPrefix(key.Value, illogicalCompositionKeywordsXPrefix) {
			return true
		}
	}

	return false
}

func illogicalCompositionKeywordsNullFlag(schema *yaml.Node) bool {
	return oas.IsTrue(oas.Get(schema, illogicalCompositionKeywordsNullable))
}

func illogicalCompositionKeywordsAllowsNull(schema *yaml.Node) bool {
	if illogicalCompositionKeywordsNullFlag(schema) {
		return true
	}

	return slices.Contains(oas.Types(schema), illogicalCompositionKeywordsNull)
}

// illogicalCompositionKeywordsTypeSet returns the declared types of a schema,
// with null added for nullable schemas, and false when no type is declared.
func illogicalCompositionKeywordsTypeSet(schema *yaml.Node) ([]string, bool) {
	if !oas.Has(schema, "type") {
		return nil, false
	}

	types := oas.Types(schema)
	if illogicalCompositionKeywordsNullFlag(schema) && !slices.Contains(types, illogicalCompositionKeywordsNull) {
		return slices.Concat(types, []string{illogicalCompositionKeywordsNull}), true
	}

	return slices.Clone(types), true
}

// illogicalCompositionKeywordsSharedTypes returns the shared types, or nil when
// either side declares no type.
func illogicalCompositionKeywordsSharedTypes(left, right *yaml.Node) []string {
	shared, ok := illogicalCompositionKeywordsSharedKnown(left, right)
	if !ok {
		return nil
	}

	return shared
}

// illogicalCompositionKeywordsSharedKnown is like SharedTypes but also tells
// whether both sides declare a type, so an empty intersection can be told from
// an unknown one.
func illogicalCompositionKeywordsSharedKnown(left, right *yaml.Node) ([]string, bool) {
	leftTypes, leftOK := illogicalCompositionKeywordsTypeSet(left)
	rightTypes, rightOK := illogicalCompositionKeywordsTypeSet(right)
	if !leftOK || !rightOK {
		return nil, false
	}

	shared := seq.FlatMapIndexed(leftTypes, func(i int, name string) []string {
		if slices.Contains(rightTypes, name) && slices.Index(leftTypes, name) == i {
			return []string{name}
		}

		return nil
	})

	if len(shared) == 0 && illogicalCompositionKeywordsIntegerMeetsNumber(left, right, leftTypes, rightTypes) {
		return []string{illogicalCompositionKeywordsInteger}, true
	}

	return shared, true
}

func illogicalCompositionKeywordsIntegerMeetsNumber(left, right *yaml.Node, leftTypes, rightTypes []string) bool {
	switch {
	case slices.Contains(leftTypes, illogicalCompositionKeywordsInteger) && slices.Contains(rightTypes, illogicalCompositionKeywordsNumber):
		return illogicalCompositionKeywordsAllowsInteger(right)
	case slices.Contains(leftTypes, illogicalCompositionKeywordsNumber) && slices.Contains(rightTypes, illogicalCompositionKeywordsInteger):
		return illogicalCompositionKeywordsAllowsInteger(left)
	default:
		return false
	}
}

func illogicalCompositionKeywordsAllowsInteger(schema *yaml.Node) bool {
	values, ok := illogicalCompositionKeywordsAllowed(schema)
	if !ok {
		return true
	}

	return slices.ContainsFunc(values, func(value any) bool {
		number, isNumber := value.(float64)

		return isNumber && number == math.Trunc(number)
	})
}

// illogicalCompositionKeywordsAllowed returns the enum values, or the const as
// a single value, and false when neither is declared.
func illogicalCompositionKeywordsAllowed(schema *yaml.Node) ([]any, bool) {
	if enum := oas.Get(schema, "enum"); oas.IsSeq(enum) {
		values, _ := illogicalCompositionKeywordsValue(enum).([]any)

		return values, true
	}

	if constant := oas.Get(schema, "const"); constant != nil && !oas.IsNull(constant) {
		return []any{illogicalCompositionKeywordsValue(constant)}, true
	}

	return nil, false
}

func illogicalCompositionKeywordsExclusive(left, right *yaml.Node) bool {
	leftValues, leftOK := illogicalCompositionKeywordsAllowed(left)
	rightValues, rightOK := illogicalCompositionKeywordsAllowed(right)
	if leftOK && rightOK {
		return len(illogicalCompositionKeywordsIntersect(leftValues, rightValues)) == 0
	}

	shared, ok := illogicalCompositionKeywordsSharedKnown(left, right)

	return ok && len(shared) == 0
}

func illogicalCompositionKeywordsRequiredSet(schema *yaml.Node) []string {
	return oas.Strings(oas.Get(schema, illogicalCompositionKeywordsRequired))
}

func illogicalCompositionKeywordsHas(properties *yaml.Node, name string) bool {
	return oas.Get(properties, name) != nil
}

func illogicalCompositionKeywordsPropertyOverlap(root, left, right *yaml.Node) string {
	leftProps := oas.Get(left, illogicalCompositionKeywordsProps)
	rightProps := oas.Get(right, illogicalCompositionKeywordsProps)
	if !oas.IsMap(leftProps) || !oas.IsMap(rightProps) {
		return ""
	}

	leftRequired := illogicalCompositionKeywordsRequiredSet(left)
	rightRequired := illogicalCompositionKeywordsRequiredSet(right)

	if illogicalCompositionKeywordsForbids(left, leftProps, rightRequired) ||
		illogicalCompositionKeywordsForbids(right, rightProps, leftRequired) {
		return ""
	}

	if illogicalCompositionKeywordsRequireDifferent(leftProps, rightProps, leftRequired, rightRequired) {
		return ""
	}

	shared := seq.FlatMapPairs(oas.Pairs(leftProps), func(key, _ *yaml.Node) []string {
		if illogicalCompositionKeywordsHas(rightProps, key.Value) {
			return []string{key.Value}
		}

		return nil
	})

	if len(shared) == 0 {
		return ""
	}

	return illogicalCompositionKeywordsClassify(root, shared, leftProps, rightProps, leftRequired, rightRequired)
}

func illogicalCompositionKeywordsForbids(schema, declared *yaml.Node, required []string) bool {
	additional := oas.Get(schema, "additionalProperties")
	if !oas.IsScalar(additional) || additional.Value != illogicalCompositionKeywordsFalse {
		return false
	}

	return slices.ContainsFunc(required, func(name string) bool {
		return !illogicalCompositionKeywordsHas(declared, name)
	})
}

func illogicalCompositionKeywordsRequireDifferent(
	leftProps, rightProps *yaml.Node,
	leftRequired, rightRequired []string,
) bool {
	leftUnknown := slices.ContainsFunc(leftRequired, func(name string) bool {
		return !illogicalCompositionKeywordsHas(rightProps, name) && !slices.Contains(rightRequired, name)
	})
	rightUnknown := slices.ContainsFunc(rightRequired, func(name string) bool {
		return !illogicalCompositionKeywordsHas(leftProps, name) && !slices.Contains(leftRequired, name)
	})

	return leftUnknown && rightUnknown
}

// illogicalCompositionKeywordsClassified accumulates the shared properties
// found so far; stop is set when a property required in both schemas excludes
// the overlap.
type illogicalCompositionKeywordsClassified struct {
	optional, ambiguous []string
	stop                bool
}

func illogicalCompositionKeywordsClassify(
	root *yaml.Node,
	shared []string,
	leftProps, rightProps *yaml.Node,
	leftRequired, rightRequired []string,
) string {
	result := illogicalCompositionKeywordsClassifyAll(
		root, shared, leftProps, rightProps, leftRequired, rightRequired,
		illogicalCompositionKeywordsClassified{},
	)
	if result.stop {
		return ""
	}

	return illogicalCompositionKeywordsPropertyMessage(result.optional, result.ambiguous)
}

// illogicalCompositionKeywordsClassifyAll folds the shared names, in order.
func illogicalCompositionKeywordsClassifyAll(
	root *yaml.Node,
	names []string,
	leftProps, rightProps *yaml.Node,
	leftRequired, rightRequired []string,
	acc illogicalCompositionKeywordsClassified,
) illogicalCompositionKeywordsClassified {
	if len(names) == 0 || acc.stop {
		return acc
	}

	return illogicalCompositionKeywordsClassifyAll(
		root,
		names[1:],
		leftProps,
		rightProps,
		leftRequired,
		rightRequired,
		illogicalCompositionKeywordsClassifyOne(
			root,
			names[0],
			leftProps,
			rightProps,
			leftRequired,
			rightRequired,
			acc,
		),
	)
}

func illogicalCompositionKeywordsClassifyOne(
	root *yaml.Node,
	name string,
	leftProps, rightProps *yaml.Node,
	leftRequired, rightRequired []string,
	acc illogicalCompositionKeywordsClassified,
) illogicalCompositionKeywordsClassified {
	leftProperty := oas.Resolve(root, oas.Get(leftProps, name))
	rightProperty := oas.Resolve(root, oas.Get(rightProps, name))
	if !oas.IsMap(leftProperty) || !oas.IsMap(rightProperty) {
		return acc
	}

	if illogicalCompositionKeywordsUnsupported(leftProperty) ||
		illogicalCompositionKeywordsUnsupported(rightProperty) {
		return acc
	}

	if !illogicalCompositionKeywordsExclusive(leftProperty, rightProperty) {
		return illogicalCompositionKeywordsClassified{
			optional:  acc.optional,
			ambiguous: slices.Concat(acc.ambiguous, []string{name}),
		}
	}

	if slices.Contains(leftRequired, name) && slices.Contains(rightRequired, name) {
		return illogicalCompositionKeywordsClassified{stop: true}
	}

	return illogicalCompositionKeywordsClassified{
		optional:  slices.Concat(acc.optional, []string{name}),
		ambiguous: acc.ambiguous,
	}
}

// illogicalCompositionKeywordsAlso lists the other ambiguous shared properties.
func illogicalCompositionKeywordsAlso(ambiguous []string) string {
	if len(ambiguous) == 0 {
		return ""
	}

	return " Other shared properties: " + illogicalCompositionKeywordsQuote(ambiguous) + "."
}

func illogicalCompositionKeywordsPropertyMessage(optional, ambiguous []string) string {
	if len(optional) > 0 {
		return "Add " + illogicalCompositionKeywordsQuote(optional) +
			" to `required` in every schema; an optional property cannot distinguish the schemas." +
			illogicalCompositionKeywordsAlso(ambiguous)
	}

	if len(ambiguous) > 0 {
		return "Both schemas define " + illogicalCompositionKeywordsQuote(ambiguous) +
			" without constraints that exclude each other. " +
			"Add a discriminator, or constrain the shared properties to different values."
	}

	return ""
}

const (
	illogicalCompositionKeywordsTrue  = "true"
	illogicalCompositionKeywordsFalse = "false"
)

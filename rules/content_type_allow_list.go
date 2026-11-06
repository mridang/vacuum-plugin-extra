package rules

import (
	"encoding/json"
	"iter"
	"maps"
	"math"
	"net/mail"
	"net/netip"
	"net/url"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const contentTypeAllowListJSON = "application/json"

const (
	contentTypeAllowListOption   = "mimetypes"
	contentTypeAllowListExamples = "validateExamples"
)

// ContentTypeAllowList returns the extra-content-type-allow-list rule.
//
// # What it checks
//
// Every media type used in a request body or a response content map, whether
// inline or declared under components.requestBodies and components.responses,
// must be in the allowed list. Media type parameters (for example
// "; charset=utf-8") and letter case are ignored when comparing, and an
// allowed entry of the form "type/*" matches every subtype of that type.
//
// # Why
//
// Restricting an API to a small set of media types keeps client generators,
// gateways and documentation consistent and prevents a stray text/plain or
// application/xml body from slipping in unnoticed.
//
// # Where it comes from
//
// apilint (content-type), including its second half: for every media type that
// is allowed, the example bodies are parsed according to the media type and
// then validated against the schema of the media type.
//
// # Example bodies
//
// For each media type in the allowed list, the media type object's example and
// the value of each entry of examples (a $ref to components.examples is
// followed) is checked. For application/json (and +json types) a string example
// is parsed as JSON text, and for application/yaml, application/x-yaml and
// text/yaml (and +yaml types) as YAML text; a parse failure is reported as
// "JSON body does not parse correctly!" or "YAML body does not parse correctly!".
// A string example whose schema declares type string is taken as the string
// value itself and not parsed. An example written as structured YAML is already
// parsed. Other media types (XML, text) are not parsed, as in apilint. The parsed
// body is then validated against the media type schema (a $ref is followed) and
// the first failure is reported as "example body does not match schema!" with the
// path in the body. The validator covers type (including nullable and OpenAPI
// 3.1 type lists), enum, const, required, properties, additionalProperties,
// items, minItems, maxItems, uniqueItems, minProperties, maxProperties,
// minLength, maxLength, pattern, minimum, maximum, exclusiveMinimum,
// exclusiveMaximum (boolean and numeric forms), multipleOf, allOf, anyOf,
// oneOf, not and the formats date-time, date, time, email, uuid, uri, hostname,
// ipv4 and ipv6. Unknown formats and keywords are ignored.
//
// # Options
//
//   - mimetypes (list of strings, default ["application/json"]): the allowed
//     media types. A single string is accepted as a one element list.
//   - validateExamples (boolean, default true): parse and validate example
//     bodies of the allowed media types as described above.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            text/plain:
//	              schema: {type: string}
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema: {type: string}
//
// # Known false positives and limitations
//
// File upload and download endpoints legitimately use media types such as
// application/octet-stream or multipart/form-data; add them to the list.
// Wildcard media types in the document ("*/*") are compared literally. The
// example validator is a purpose built subset of JSON Schema; schemas that rely
// on unevaluated keywords, dependent schemas, conditionals or remote references
// are not fully checked.
func ContentTypeAllowList() Rule {
	return New(Spec{
		Name:        "content-type-allow-list",
		Description: "Request and response media types must be in the allowed list.",
		HowToFix:    "Use one of the allowed media types, or extend the mimetypes option.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        contentTypeAllowListOption,
				Description: "Allowed media types; type/* matches every subtype.",
				Default:     []any{contentTypeAllowListJSON},
			},
			{
				Name:        contentTypeAllowListExamples,
				Description: "Parse example bodies and validate them against the schema.",
				Default:     true,
			},
		},
		Check: OnDocument(checkContentTypeAllowList),
	})
}

// checkContentTypeAllowList reports media types outside the allowed list.
func checkContentTypeAllowList(call *Call) []Finding {
	allowed := contentTypeAllowListNormalise(call.Strings(contentTypeAllowListOption))
	if len(allowed) == 0 {
		return nil
	}

	return seq.FlatMapSlice(
		contentTypeAllowListContents(call.Root),
		func(content contentTypeAllowListContent) []Finding {
			return seq.FlatMapSlice(oas.MediaTypes(content.node, content.path), func(media oas.MediaType) []Finding {
				if !contentTypeAllowListAllows(allowed, media.Name) {
					return One(Finding{
						Node: media.Node,
						Path: media.Path,
						Message: "media type '" + media.Name + "' should be one of '" + strings.Join(
							allowed,
							"', '",
						) + "'",
					})
				}

				if !call.Bool(contentTypeAllowListExamples) {
					return nil
				}

				return contentTypeAllowListCheckExamples(call.Root, media)
			})
		},
	)
}

type contentTypeAllowListContent struct {
	node *yaml.Node
	path string
}

// contentTypeAllowListContents gathers every inline content map of the document.
func contentTypeAllowListContents(root *yaml.Node) []contentTypeAllowListContent {
	return slices.Concat(
		seq.FlatMap(oas.Operations(root), func(operation oas.Operation) []contentTypeAllowListContent {
			return slices.Concat(
				contentTypeAllowListOwned(
					oas.Get(operation.Node, "requestBody"),
					oas.Join(operation.Path, "requestBody"),
				),
				contentTypeAllowListEach(oas.Get(operation.Node, "responses"), oas.Join(operation.Path, "responses")),
			)
		}),
		contentTypeAllowListEach(oas.Dig(root, "components", "requestBodies"), "$.components.requestBodies"),
		contentTypeAllowListEach(oas.Dig(root, "components", "responses"), "$.components.responses"),
	)
}

// contentTypeAllowListEach returns the content maps of every entry of a mapping.
func contentTypeAllowListEach(owners *yaml.Node, path string) []contentTypeAllowListContent {
	return seq.FlatMapPairs(oas.Pairs(owners), func(key, owner *yaml.Node) []contentTypeAllowListContent {
		return contentTypeAllowListOwned(owner, oas.Join(path, key.Value))
	})
}

// contentTypeAllowListOwned returns the content map of the owner, when it has one.
func contentTypeAllowListOwned(owner *yaml.Node, path string) []contentTypeAllowListContent {
	content := oas.Get(owner, "content")
	if content == nil {
		return nil
	}

	return []contentTypeAllowListContent{{node: content, path: oas.Join(path, "content")}}
}

// contentTypeAllowListNormalise lower-cases the allowed entries and drops parameters.
func contentTypeAllowListNormalise(values []string) []string {
	return seq.Filter(
		slices.Values(seq.Map(slices.Values(values), contentTypeAllowListBase)),
		func(base string) bool {
			return base != ""
		},
	)
}

// contentTypeAllowListBase strips parameters from a media type and lower-cases it.
func contentTypeAllowListBase(mediaType string) string {
	base, _, _ := strings.Cut(mediaType, ";")

	return Fold(base)
}

// contentTypeAllowListAllows reports whether the media type is covered by the allowed list.
func contentTypeAllowListAllows(allowed []string, mediaType string) bool {
	base := contentTypeAllowListBase(mediaType)
	if slices.Contains(allowed, base) {
		return true
	}

	group, _, found := strings.Cut(base, "/")

	return found && slices.Contains(allowed, group+"/*")
}

const (
	contentTypeAllowListYAMLSuffix = "+yaml"
	contentTypeAllowListJSONSuffix = "+json"
	contentTypeAllowListString     = "string"
	contentTypeAllowListObject     = "object"
	contentTypeAllowListArray      = "array"
	contentTypeAllowListNumber     = "number"
	contentTypeAllowListInteger    = "integer"
	contentTypeAllowListBoolean    = "boolean"
	contentTypeAllowListNull       = "null"
	contentTypeAllowListMaxDepth   = 32
	contentTypeAllowListStrTag     = "!!str"
	contentTypeAllowListNullTag    = "!!null"
	contentTypeAllowListBoolTag    = "!!bool"
	contentTypeAllowListIntTag     = "!!int"
	contentTypeAllowListFloatTag   = "!!float"
	contentTypeAllowListDateTime   = "date-time"
	contentTypeAllowListDateFormat = "2006-01-02"
	contentTypeAllowListTimeFormat = "15:04:05Z07:00"
)

// contentTypeAllowListExample is one example value of a media type.
type contentTypeAllowListExample struct {
	node *yaml.Node
	path string
}

// contentTypeAllowListCheckExamples parses and validates the examples of an
// allowed media type.
func contentTypeAllowListCheckExamples(root *yaml.Node, media oas.MediaType) []Finding {
	kind := contentTypeAllowListKind(media.Name)
	if kind == "" {
		return nil
	}

	schema := oas.Get(media.Node, "schema")

	return seq.FlatMapSlice(
		contentTypeAllowListExampleNodes(root, media),
		func(example contentTypeAllowListExample) []Finding {
			failure := contentTypeAllowListFailure(root, kind, example, schema)

			return When(failure != "", func() Finding {
				return Finding{Node: example.node, Path: example.path, Message: failure}
			})
		},
	)
}

// contentTypeAllowListFailure returns why the example is invalid, or the empty string.
func contentTypeAllowListFailure(
	root *yaml.Node,
	kind string,
	example contentTypeAllowListExample,
	schema *yaml.Node,
) string {
	body, failure := contentTypeAllowListParse(root, kind, example.node, schema)
	if failure != "" || body == nil || schema == nil {
		return failure
	}

	validator := contentTypeAllowListValidator{root: root}
	if message := validator.validate(body, schema, "body", 0); message != "" {
		return "example body does not match schema! " + message
	}

	return ""
}

// contentTypeAllowListKind returns json or yaml for media types whose bodies
// are parsed, and the empty string for all others.
func contentTypeAllowListKind(mediaType string) string {
	base := contentTypeAllowListBase(mediaType)

	switch {
	case base == contentTypeAllowListJSON || strings.HasSuffix(base, contentTypeAllowListJSONSuffix):
		return "json"
	case base == "application/yaml" || base == "application/x-yaml" || base == "text/yaml" ||
		strings.HasSuffix(base, contentTypeAllowListYAMLSuffix):
		return "yaml"
	default:
		return ""
	}
}

// contentTypeAllowListExamplePath descends into the value of an inline example.
func contentTypeAllowListExamplePath(path string, inline bool) string {
	if inline {
		return oas.Join(path, "value")
	}

	return path
}

// contentTypeAllowListExampleNodes lists the example values of a media type.
func contentTypeAllowListExampleNodes(root *yaml.Node, media oas.MediaType) []contentTypeAllowListExample {
	named := seq.FlatMapPairs(
		oas.Pairs(oas.Get(media.Node, "examples")),
		func(key, entry *yaml.Node) []contentTypeAllowListExample {
			value := oas.Get(oas.Resolve(root, entry), "value")
			if value == nil {
				return nil
			}

			return []contentTypeAllowListExample{{
				node: value,
				path: contentTypeAllowListExamplePath(
					oas.JoinAll(media.Path, "examples", key.Value),
					oas.Ref(entry) == "",
				),
			}}
		},
	)

	return slices.Concat(contentTypeAllowListSingleExample(media), named)
}

// contentTypeAllowListSingleExample returns the example keyword of a media type, when it has one.
func contentTypeAllowListSingleExample(media oas.MediaType) []contentTypeAllowListExample {
	example := oas.Get(media.Node, "example")
	if example == nil {
		return nil
	}

	return []contentTypeAllowListExample{{node: example, path: oas.Join(media.Path, "example")}}
}

// contentTypeAllowListParse turns an example node into a Go value. The second
// result is a failure message when a text body does not parse.
func contentTypeAllowListParse(root *yaml.Node, kind string, node, schema *yaml.Node) (any, string) {
	text := oas.IsScalar(node) && node.Tag == contentTypeAllowListStrTag
	if !text || contentTypeAllowListIsStringSchema(root, schema) {
		return contentTypeAllowListValue(node), ""
	}

	if kind == "json" {
		body := new(any)
		if err := json.Unmarshal([]byte(node.Value), body); err != nil {
			return nil, "JSON body does not parse correctly! " + err.Error()
		}

		return *body, ""
	}

	document := new(yaml.Node)
	if err := yaml.Unmarshal([]byte(node.Value), document); err != nil {
		return nil, "YAML body does not parse correctly! " + err.Error()
	}

	return contentTypeAllowListValue(oas.Unwrap(document)), ""
}

// contentTypeAllowListIsStringSchema reports whether the schema is a string.
func contentTypeAllowListIsStringSchema(root, schema *yaml.Node) bool {
	return oas.HasType(oas.Resolve(root, schema), contentTypeAllowListString)
}

// contentTypeAllowListValue converts a YAML node to plain Go values: maps,
// slices, strings, float64, bool and nil.
func contentTypeAllowListValue(raw *yaml.Node) any {
	node := oas.Unwrap(raw)

	switch {
	case node == nil:
		return nil
	case oas.IsMap(node):
		return maps.Collect(contentTypeAllowListEntries(node))
	case oas.IsSeq(node):
		return append([]any{}, seq.Map(slices.Values(node.Content), contentTypeAllowListValue)...)
	default:
		return contentTypeAllowListScalar(node)
	}
}

// contentTypeAllowListEntries yields the converted entries of a mapping node.
func contentTypeAllowListEntries(node *yaml.Node) iter.Seq2[string, any] {
	return func(yield func(string, any) bool) {
		for key, value := range oas.Pairs(node) {
			if !yield(key.Value, contentTypeAllowListValue(value)) {
				return
			}
		}
	}
}

// contentTypeAllowListScalar converts a scalar by its YAML tag.
func contentTypeAllowListScalar(node *yaml.Node) any {
	switch node.Tag {
	case contentTypeAllowListNullTag:
		return nil
	case contentTypeAllowListBoolTag:
		return node.Value == "true"
	case contentTypeAllowListIntTag, contentTypeAllowListFloatTag:
		if number, err := strconv.ParseFloat(node.Value, 64); err == nil {
			return number
		}

		if number, err := strconv.ParseInt(node.Value, 0, 64); err == nil {
			return float64(number)
		}
	}

	return node.Value
}

// contentTypeAllowListValidator validates plain values against a schema.
type contentTypeAllowListValidator struct {
	root *yaml.Node
}

// validate returns the first violation of the schema, or the empty string.
func (v contentTypeAllowListValidator) validate(value any, declared *yaml.Node, path string, depth int) string {
	schema := oas.Resolve(v.root, declared)
	if !oas.IsMap(schema) || depth > contentTypeAllowListMaxDepth {
		return ""
	}

	if message := v.composition(value, schema, path, depth); message != "" {
		return message
	}

	if message := v.typeAndEnum(value, schema, path); message != "" {
		return message
	}

	switch typed := value.(type) {
	case string:
		return v.checkString(typed, schema, path)
	case float64:
		return v.checkNumber(typed, schema, path)
	case []any:
		return v.checkArray(typed, schema, path, depth)
	case map[string]any:
		return v.checkObject(typed, schema, path, depth)
	}

	return ""
}

// composition checks allOf, anyOf, oneOf and not.
func (v contentTypeAllowListValidator) composition(value any, schema *yaml.Node, path string, depth int) string {
	for _, sub := range oas.Items(oas.Get(schema, "allOf")) {
		if message := v.validate(value, sub, path, depth+1); message != "" {
			return message
		}
	}

	if branches := oas.Get(schema, "anyOf"); oas.IsSeq(branches) && v.matching(value, branches, path, depth) == 0 {
		return "'" + path + "' must match a schema in anyOf"
	}

	if branches := oas.Get(schema, "oneOf"); oas.IsSeq(branches) && v.matching(value, branches, path, depth) != 1 {
		return "'" + path + "' must match exactly one schema in oneOf"
	}

	if not := oas.Get(schema, "not"); not != nil && v.validate(value, not, path, depth+1) == "" {
		return "'" + path + "' must not be valid against the schema in not"
	}

	return ""
}

// matching counts the branches the value satisfies.
func (v contentTypeAllowListValidator) matching(value any, branches *yaml.Node, path string, depth int) int {
	return len(seq.Filter(slices.Values(branches.Content), func(branch *yaml.Node) bool {
		return v.validate(value, branch, path, depth+1) == ""
	}))
}

// typeAndEnum checks type, nullable, enum and const.
func (v contentTypeAllowListValidator) typeAndEnum(value any, schema *yaml.Node, path string) string {
	if types := oas.Types(schema); len(types) > 0 && !contentTypeAllowListTypeOK(value, types, schema) {
		return "'" + path + "' must be " + strings.Join(types, ",")
	}

	if enum := oas.Get(schema, "enum"); oas.IsSeq(enum) && !contentTypeAllowListIn(value, enum) {
		return "'" + path + "' must be equal to one of the allowed values"
	}

	if constant := oas.Get(
		schema,
		"const",
	); constant != nil &&
		!reflect.DeepEqual(value, contentTypeAllowListValue(constant)) {
		return "'" + path + "' must be equal to constant"
	}

	return ""
}

// contentTypeAllowListIn reports whether the value is one of the enum items.
func contentTypeAllowListIn(value any, enum *yaml.Node) bool {
	for _, item := range oas.Items(enum) {
		if reflect.DeepEqual(value, contentTypeAllowListValue(item)) {
			return true
		}
	}

	return false
}

// contentTypeAllowListTypeOK reports whether the value has one of the types.
func contentTypeAllowListTypeOK(value any, types []string, schema *yaml.Node) bool {
	if value == nil && oas.IsTrue(oas.Get(schema, "nullable")) {
		return true
	}

	for _, kind := range types {
		if contentTypeAllowListIsType(value, kind) {
			return true
		}
	}

	return false
}

// contentTypeAllowListIsType reports whether the value has the JSON Schema type.
func contentTypeAllowListIsType(value any, kind string) bool {
	switch kind {
	case contentTypeAllowListString:
		_, ok := value.(string)

		return ok
	case contentTypeAllowListNumber:
		_, ok := value.(float64)

		return ok
	case contentTypeAllowListInteger:
		number, ok := value.(float64)

		return ok && number == math.Trunc(number)
	case contentTypeAllowListBoolean:
		_, ok := value.(bool)

		return ok
	case contentTypeAllowListObject:
		_, ok := value.(map[string]any)

		return ok
	case contentTypeAllowListArray:
		_, ok := value.([]any)

		return ok
	case contentTypeAllowListNull:
		return value == nil
	default:
		return true
	}
}

// checkString checks length, pattern and format.
func (v contentTypeAllowListValidator) checkString(value string, schema *yaml.Node, path string) string {
	length := utf8.RuneCountInString(value)

	if limit, ok := oas.Int(oas.Get(schema, "minLength")); ok && length < limit {
		return "'" + path + "' must NOT have fewer than " + strconv.Itoa(limit) + " characters"
	}

	if limit, ok := oas.Int(oas.Get(schema, "maxLength")); ok && length > limit {
		return "'" + path + "' must NOT have more than " + strconv.Itoa(limit) + " characters"
	}

	if pattern := oas.Text(oas.Get(schema, "pattern")); pattern != "" {
		if compiled, err := regexp.Compile(pattern); err == nil && !compiled.MatchString(value) {
			return "'" + path + "' must match pattern \"" + pattern + "\""
		}
	}

	if format := oas.Text(oas.Get(schema, "format")); !contentTypeAllowListFormatOK(value, format) {
		return "'" + path + "' must match format \"" + format + "\""
	}

	return ""
}

// contentTypeAllowListMatches reports whether the value matches the format
// expression.
func contentTypeAllowListMatches(expression, value string) bool {
	return regexp.MustCompile(expression).MatchString(value)
}

const (
	contentTypeAllowListUUIDPattern = `^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`
	contentTypeAllowListHostPattern = `^[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?` +
		`(\.[A-Za-z0-9]([A-Za-z0-9-]{0,61}[A-Za-z0-9])?)*$`
	contentTypeAllowListTolerance = 1e-9
	contentTypeAllowListFloatFmt  = 'f'
)

// contentTypeAllowListFloat formats a number without an exponent.
func contentTypeAllowListFloat(value float64) string {
	return strconv.FormatFloat(value, contentTypeAllowListFloatFmt, -1, 64)
}

// contentTypeAllowListFormatOK checks the supported string formats.
func contentTypeAllowListFormatOK(value, format string) bool {
	switch format {
	case contentTypeAllowListDateTime:
		_, err := time.Parse(time.RFC3339, strings.ToUpper(value))

		return err == nil
	case "date":
		_, err := time.Parse(contentTypeAllowListDateFormat, value)

		return err == nil
	case "time":
		_, err := time.Parse(contentTypeAllowListTimeFormat, strings.ToUpper(value))

		return err == nil
	case "email":
		address, err := mail.ParseAddress(value)

		return err == nil && address.Address == value
	case "uuid":
		return contentTypeAllowListMatches(contentTypeAllowListUUIDPattern, value)
	case "uri":
		parsed, err := url.Parse(value)

		return err == nil && parsed.Scheme != ""
	case "hostname":
		return contentTypeAllowListMatches(contentTypeAllowListHostPattern, value)
	case "ipv4":
		address, err := netip.ParseAddr(value)

		return err == nil && address.Is4()
	case "ipv6":
		address, err := netip.ParseAddr(value)

		return err == nil && address.Is6()
	default:
		return true
	}
}

// checkNumber checks bounds and multipleOf.
func (v contentTypeAllowListValidator) checkNumber(value float64, schema *yaml.Node, path string) string {
	if message := contentTypeAllowListLower(value, schema, path); message != "" {
		return message
	}

	if message := contentTypeAllowListUpper(value, schema, path); message != "" {
		return message
	}

	if step, ok := oas.Number(oas.Get(schema, "multipleOf")); ok && step > 0 {
		if quotient := value / step; math.Abs(quotient-math.Round(quotient)) > contentTypeAllowListTolerance {
			return "'" + path + "' must be multiple of " + contentTypeAllowListFloat(step)
		}
	}

	return ""
}

// contentTypeAllowListLower checks minimum and exclusiveMinimum in both forms.
func contentTypeAllowListLower(value float64, schema *yaml.Node, path string) string {
	exclusive := oas.Get(schema, "exclusiveMinimum")

	if limit, ok := oas.Number(oas.Get(schema, "minimum")); ok {
		if value < limit || (oas.IsTrue(exclusive) && value == limit) {
			return "'" + path + "' must be greater than or equal to " + contentTypeAllowListFloat(limit)
		}
	}

	if limit, ok := oas.Number(exclusive); ok && value <= limit {
		return "'" + path + "' must be greater than " + contentTypeAllowListFloat(limit)
	}

	return ""
}

// contentTypeAllowListUpper checks maximum and exclusiveMaximum in both forms.
func contentTypeAllowListUpper(value float64, schema *yaml.Node, path string) string {
	exclusive := oas.Get(schema, "exclusiveMaximum")

	if limit, ok := oas.Number(oas.Get(schema, "maximum")); ok {
		if value > limit || (oas.IsTrue(exclusive) && value == limit) {
			return "'" + path + "' must be less than or equal to " + contentTypeAllowListFloat(limit)
		}
	}

	if limit, ok := oas.Number(exclusive); ok && value >= limit {
		return "'" + path + "' must be less than " + contentTypeAllowListFloat(limit)
	}

	return ""
}

// checkArray checks item count, uniqueness and items.
func (v contentTypeAllowListValidator) checkArray(value []any, schema *yaml.Node, path string, depth int) string {
	if limit, ok := oas.Int(oas.Get(schema, "minItems")); ok && len(value) < limit {
		return "'" + path + "' must NOT have fewer than " + strconv.Itoa(limit) + " items"
	}

	if limit, ok := oas.Int(oas.Get(schema, "maxItems")); ok && len(value) > limit {
		return "'" + path + "' must NOT have more than " + strconv.Itoa(limit) + " items"
	}

	if oas.IsTrue(oas.Get(schema, "uniqueItems")) && contentTypeAllowListDuplicated(value) {
		return "'" + path + "' must NOT have duplicate items"
	}

	items := oas.Get(schema, "items")

	for index, item := range value {
		if message := v.validate(item, items, path+"["+strconv.Itoa(index)+"]", depth+1); message != "" {
			return message
		}
	}

	return ""
}

// contentTypeAllowListDuplicated reports whether two items are equal.
func contentTypeAllowListDuplicated(values []any) bool {
	for index, value := range values {
		for _, other := range values[index+1:] {
			if reflect.DeepEqual(value, other) {
				return true
			}
		}
	}

	return false
}

// checkObject checks property counts, required, properties and
// additionalProperties.
func (v contentTypeAllowListValidator) checkObject(
	value map[string]any,
	schema *yaml.Node,
	path string,
	depth int,
) string {
	if limit, ok := oas.Int(oas.Get(schema, "minProperties")); ok && len(value) < limit {
		return "'" + path + "' must NOT have fewer than " + strconv.Itoa(limit) + " properties"
	}

	if limit, ok := oas.Int(oas.Get(schema, "maxProperties")); ok && len(value) > limit {
		return "'" + path + "' must NOT have more than " + strconv.Itoa(limit) + " properties"
	}

	for _, name := range oas.Strings(oas.Get(schema, "required")) {
		if _, present := value[name]; !present {
			return "'" + path + "' must have required property '" + name + "'"
		}
	}

	properties := oas.Get(schema, "properties")
	for key, sub := range oas.Pairs(properties) {
		if item, present := value[key.Value]; present {
			if message := v.validate(item, sub, path+"."+key.Value, depth+1); message != "" {
				return message
			}
		}
	}

	return v.additional(value, schema, properties, path, depth)
}

// additional checks properties that the schema does not declare.
func (v contentTypeAllowListValidator) additional(
	value map[string]any, schema, properties *yaml.Node, path string, depth int,
) string {
	extra := oas.Get(schema, "additionalProperties")
	if extra == nil {
		return ""
	}

	for _, name := range slices.Sorted(maps.Keys(value)) {
		if oas.Has(properties, name) {
			continue
		}

		if oas.IsScalar(extra) && extra.Value == "false" {
			return "'" + path + "' must NOT have additional properties '" + name + "'"
		}

		if oas.IsMap(extra) {
			if message := v.validate(value[name], extra, path+"."+name, depth+1); message != "" {
				return message
			}
		}
	}

	return ""
}

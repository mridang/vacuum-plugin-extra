package rules

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	operationIDVerbMatchesMethodMedia        = "media"
	operationIDVerbMatchesMethodFirst        = "${1}"
	operationIDVerbMatchesMethodGetMethod    = "get"
	operationIDVerbMatchesMethodDelete       = "delete"
	operationIDVerbMatchesMethodMultiParams  = 2
	operationIDVerbMatchesMethodStrict       = "strict"
	operationIDVerbMatchesMethodParamPattern = `\{[A-Za-z0-9\-_]+\}`
	operationIDVerbMatchesMethodVersion      = `^v\d+$`
	operationIDVerbMatchesMethodPrefix       = "operationIds should follow naming convention: "
)

// operationIDVerbMatchesMethodInflection is one singularisation rule.
type operationIDVerbMatchesMethodInflection struct {
	pattern     *regexp.Regexp
	replacement string
}

// OperationIDVerbMatchesMethod returns the extra-operation-id-verb-matches-method rule.
//
// # What it checks
//
// A port of the IBM function operationIdNamingConvention, run in its strict
// mode (as the ibm-operationid-naming-convention rule does). The operationId
// must be exactly one acceptable verb, an underscore and the resource part of
// the path, written as snake case words joined by underscores. The accepted
// verbs depend on the HTTP method and the path shape:
//
//   - GET on a resource-oriented collection path (/bands): list.
//   - GET on a path ending in a parameter (/bands/{id}): get, plus check when
//     the path has two or more parameter references.
//   - POST on a resource-oriented collection path: create.
//   - PATCH: update.
//   - PUT: replace, plus add on a path that ends in a parameter and has two or
//     more parameters, or set on a non resource-oriented path that does not
//     end in a parameter but has at least one parameter.
//   - DELETE: delete, plus remove (ends in a parameter, two or more
//     parameters) or unset (not resource-oriented, does not end in a
//     parameter, at least one parameter).
//
// GET and POST on paths that are not resource-oriented, and every other
// method, are not checked. A path counts as resource-oriented when it ends in a
// parameter (IBM's lookup for the matching collection path never fails, so every
// such path is treated as resource-oriented) or when another path starts with
// the path followed by "/{".
//
// The noun part is built as IBM builds it: the path is split on "/", segments
// that are path parameters are dropped, everything up to and including a
// version segment (v1, v2, ...) is dropped, and every remaining segment is
// singularised, except that the last segment stays plural for the verbs list,
// replace, set, delete, remove and unset when the path does not end in a
// parameter. Singularisation uses a port of the Rails inflector as shipped in
// the "inflected" package, including IBM's exceptions that keep metadata, data,
// media and criteria unchanged. For /bands/{id}/albums/{album_id}, GET must be
// get_band_album or check_band_album.
//
// In non-strict mode (strict: false) only the verb is checked: the operationId
// must start with one of the acceptable verbs. Operations without an
// operationId are skipped.
//
// # Why
//
// A predictable verb and noun vocabulary makes generated SDK method names
// consistent (client.bands.list(), client.bands.create()) and keeps the
// operationId honest about what the operation does and to which resource.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-operationid-naming-convention).
//
// # Options
//
//   - strict (bool, default true): require the full verb_resource form. When
//     false only the leading verb is checked.
//
// # Bad
//
//	paths:
//	  /bands:
//	    get:
//	      operationId: get_bands
//	  /bands/{id}:
//	    get:
//	      operationId: get_bands
//
// # Good
//
//	paths:
//	  /bands:
//	    get:
//	      operationId: list_bands
//	  /bands/{id}:
//	    get:
//	      operationId: get_band
//
// # Known false positives and limitations
//
// The rule is as strict as the original: camelCase identifiers such as
// listBands and nouns that deviate from the singularised path are reported.
// The inflector is a port of the English Rails rules, so unusual words follow
// its (sometimes surprising) results.
func OperationIDVerbMatchesMethod() Rule {
	return New(Spec{
		Name:        "operation-id-verb-matches-method",
		Description: "The verb and resource of an operationId must match the HTTP method and path of the operation.",
		HowToFix:    "Name the operation <verb>_<resource>, for example list_bands, get_band or create_band.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        operationIDVerbMatchesMethodStrict,
				Description: "Require the full verb_resource form; when false only the verb is checked.",
				Default:     true,
			},
		},
		Check: OnDocument(checkOperationIDVerbMatchesMethod),
	})
}

// checkOperationIDVerbMatchesMethod compares every operationId with the expected names.
func checkOperationIDVerbMatchesMethod(call *Call) []Finding {
	routes := seq.Map(oas.PathItems(call.Root), func(item oas.PathItem) string {
		return item.Route
	})
	inflector := operationIDVerbMatchesMethodInflector()
	strict := call.Bool(operationIDVerbMatchesMethodStrict)

	return seq.FlatMap(oas.Operations(call.Root), func(operation oas.Operation) []Finding {
		key, node := oas.Entry(operation.Node, "operationId")
		if !oas.IsScalar(node) || node.Value == "" {
			return nil
		}

		message := operationIDVerbMatchesMethodMessage(operation, node.Value, routes, inflector, strict)

		return When(message != "", func() Finding {
			return Finding{Node: node, Path: oas.Join(operation.Path, key.Value), Message: message}
		})
	})
}

// operationIDVerbMatchesMethodMessage returns the problem with an operationId, or the empty string.
func operationIDVerbMatchesMethodMessage(
	operation oas.Operation,
	id string,
	routes []string,
	inflector []operationIDVerbMatchesMethodInflection,
	strict bool,
) string {
	route := operation.Route
	endsWithParam := strings.HasSuffix(route, "}")
	params := len(regexp.MustCompile(operationIDVerbMatchesMethodParamPattern).FindAllString(route, -1))
	resource := endsWithParam || slices.ContainsFunc(routes, func(other string) bool {
		return strings.HasPrefix(other, route+"/{")
	})

	verbs := operationIDVerbMatchesMethodVerbs(operation.Method, resource, endsWithParam, params)
	if len(verbs) == 0 {
		return ""
	}

	if !strict {
		if slices.ContainsFunc(verbs, func(verb string) bool { return strings.HasPrefix(id, verb) }) {
			return ""
		}

		return operationIDVerbMatchesMethodPrefix + "operationId verb should be " + strings.Join(verbs, " or ")
	}

	noun := operationIDVerbMatchesMethodNoun(route, verbs, endsWithParam, inflector)
	expected := seq.Map(slices.Values(verbs), func(verb string) string {
		return verb + "_" + noun
	})

	if slices.Contains(expected, id) {
		return ""
	}

	return fmt.Sprintf(
		"%soperationId should be %s but it's %s instead",
		operationIDVerbMatchesMethodPrefix,
		strings.Join(expected, " or "),
		id,
	)
}

// operationIDVerbMatchesMethodVerbs returns the acceptable verbs for an operation, or nil when unconstrained.
func operationIDVerbMatchesMethodVerbs(method string, resource, endsWithParam bool, params int) []string {
	switch method {
	case operationIDVerbMatchesMethodGetMethod:
		return operationIDVerbMatchesMethodGet(resource, endsWithParam, params)
	case methodPost:
		if resource {
			return []string{"create"}
		}

		return nil
	case methodPatch:
		return []string{"update"}
	case methodPut:
		return operationIDVerbMatchesMethodWrite("replace", "add", "set", resource, endsWithParam, params)
	case operationIDVerbMatchesMethodDelete:
		return operationIDVerbMatchesMethodWrite(
			operationIDVerbMatchesMethodDelete,
			"remove",
			"unset",
			resource,
			endsWithParam,
			params,
		)
	default:
		return nil
	}
}

// operationIDVerbMatchesMethodGet returns the verbs for a GET.
func operationIDVerbMatchesMethodGet(resource, endsWithParam bool, params int) []string {
	if !resource {
		return nil
	}

	if endsWithParam {
		return operationIDVerbMatchesMethodWithCheck([]string{operationIDVerbMatchesMethodGetMethod}, params)
	}

	return operationIDVerbMatchesMethodWithCheck([]string{keywordList}, params)
}

// operationIDVerbMatchesMethodWithCheck adds the check verb when the path has several parameters.
func operationIDVerbMatchesMethodWithCheck(verbs []string, params int) []string {
	if params >= operationIDVerbMatchesMethodMultiParams {
		return slices.Concat(verbs, []string{"check"})
	}

	return verbs
}

// operationIDVerbMatchesMethodWrite returns the verbs for a PUT or DELETE.
func operationIDVerbMatchesMethodWrite(
	base, nested, detached string,
	resource, endsWithParam bool,
	params int,
) []string {
	switch {
	case resource && endsWithParam && params >= operationIDVerbMatchesMethodMultiParams:
		return []string{base, nested}
	case !resource && !endsWithParam && params >= 1:
		return []string{base, detached}
	default:
		return []string{base}
	}
}

// operationIDVerbMatchesMethodNoun builds the resource part of the operationId from the path.
func operationIDVerbMatchesMethodNoun(
	route string,
	verbs []string,
	endsWithParam bool,
	inflector []operationIDVerbMatchesMethodInflection,
) string {
	segments := seq.Filter(strings.SplitSeq(strings.TrimLeft(route, "/"), "/"), func(part string) bool {
		return !strings.HasPrefix(part, "{") && !strings.HasSuffix(part, "}")
	})

	version := regexp.MustCompile(operationIDVerbMatchesMethodVersion)
	parts := segments[slices.IndexFunc(segments, version.MatchString)+1:]

	plural := slices.ContainsFunc(verbs, func(verb string) bool {
		return slices.Contains([]string{keywordList, "replace", "set", "delete", "remove", "unset"}, verb)
	})

	singular := seq.FlatMapIndexed(parts, func(i int, part string) []string {
		if i != len(parts)-1 || !plural || endsWithParam {
			return []string{operationIDVerbMatchesMethodSingular(inflector, part)}
		}

		return []string{part}
	})

	return strings.Join(singular, "_")
}

// operationIDVerbMatchesMethodSingular singularises a word with the Rails inflection rules.
func operationIDVerbMatchesMethodSingular(inflector []operationIDVerbMatchesMethodInflection, word string) string {
	if word == "" {
		return word
	}

	if slices.ContainsFunc(operationIDVerbMatchesMethodUncountable(), func(uncountable string) bool {
		return regexp.MustCompile(`(?i)\b` + uncountable + `$`).MatchString(word)
	}) {
		return word
	}

	for _, rule := range inflector {
		location := rule.pattern.FindStringSubmatchIndex(word)
		if location == nil {
			continue
		}

		expanded := rule.pattern.ExpandString(nil, rule.replacement, word, location)

		return word[:location[0]] + string(expanded) + word[location[1]:]
	}

	return word
}

// operationIDVerbMatchesMethodUncountable returns the words that are never singularised.
func operationIDVerbMatchesMethodUncountable() []string {
	return []string{
		"equipment", "information", "rice", "money", "species", "series", "fish", "sheep", "jeans", "police",
	}
}

// operationIDVerbMatchesMethodInflector returns the singularisation rules, latest definition first.
func operationIDVerbMatchesMethodInflector() []operationIDVerbMatchesMethodInflection {
	definitions := [][2]string{
		{`s$`, ``},
		{`(ss)$`, operationIDVerbMatchesMethodFirst},
		{`(n)ews$`, `${1}ews`},
		{`([ti])a$`, `${1}um`},
		{`((a)naly|(b)a|(d)iagno|(p)arenthe|(p)rogno|(s)ynop|(t)he)(sis|ses)$`, `${1}sis`},
		{`(^analy)(sis|ses)$`, `${1}sis`},
		{`([^f])ves$`, `${1}fe`},
		{`(hive)s$`, operationIDVerbMatchesMethodFirst},
		{`(tive)s$`, operationIDVerbMatchesMethodFirst},
		{`([lr])ves$`, `${1}f`},
		{`([^aeiouy]|qu)ies$`, `${1}y`},
		{`(s)eries$`, `${1}eries`},
		{`(m)ovies$`, `${1}ovie`},
		{`(x|ch|ss|sh)es$`, operationIDVerbMatchesMethodFirst},
		{`^(m|l)ice$`, `${1}ouse`},
		{`(bus)(es)?$`, operationIDVerbMatchesMethodFirst},
		{`(o)es$`, operationIDVerbMatchesMethodFirst},
		{`(shoe)s$`, operationIDVerbMatchesMethodFirst},
		{`(cris|test)(is|es)$`, `${1}is`},
		{`^(a)x[ie]s$`, `${1}xis`},
		{`(octop|vir)(us|i)$`, `${1}us`},
		{`(alias|status)(es)?$`, operationIDVerbMatchesMethodFirst},
		{`^(ox)en`, operationIDVerbMatchesMethodFirst},
		{`(vert|ind)ices$`, `${1}ex`},
		{`(matr)ices$`, `${1}ix`},
		{`(quiz)zes$`, operationIDVerbMatchesMethodFirst},
		{`(database)s$`, operationIDVerbMatchesMethodFirst},
	}

	irregulars := [][2]string{
		{"person", "people"},
		{"man", "men"},
		{"child", "children"},
		{"sex", "sexes"},
		{"move", "moves"},
		{"zombie", "zombies"},
		{"metadata", "metadata"},
		{"data", "data"},
		{operationIDVerbMatchesMethodMedia, operationIDVerbMatchesMethodMedia},
		{"criteria", "criteria"},
	}

	all := slices.Concat(definitions, seq.FlatMapSlice(irregulars, func(pair [2]string) [][2]string {
		singular, plural := pair[0], pair[1]

		return [][2]string{
			{"(" + singular[:1] + ")" + singular[1:] + "$", "${1}" + singular[1:]},
			{"(" + plural[:1] + ")" + plural[1:] + "$", "${1}" + singular[1:]},
		}
	}))

	return seq.FlatMapPairs(
		slices.Backward(all),
		func(_ int, definition [2]string) []operationIDVerbMatchesMethodInflection {
			return []operationIDVerbMatchesMethodInflection{{
				pattern:     regexp.MustCompile(`(?i)` + definition[0]),
				replacement: definition[1],
			}}
		},
	)
}

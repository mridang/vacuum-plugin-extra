package rules

import (
	"maps"
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	pathSegmentPluralIgnoreLastOption = "ignoreLastPathSegment"
	pathSegmentPluralExceptionsOption = "exceptions"
	pathSegmentPluralCollectionOption = "onlyCollectionSegments"
)

// PathSegmentPlural returns the extra-path-segment-plural rule.
//
// # What it checks
//
// Every segment of every path template must be a plural noun, as in
// /pets/{petId}/toys. Collections are named in the plural so that the same
// segment reads naturally for the collection (/pets) and for one of its
// members (/pets/{petId}). A segment that is entirely a path parameter
// ({petId}) is skipped. Every other segment, including version segments such as
// v1, segments with a dot and hyphenated, snake_case or camelCase compounds, is
// judged as a whole word, exactly as Redocly CLI does, by a port of the
// pluralize library (blakeembrey/pluralize) that Redocly uses: a segment is
// singular when the library's singularisation rules leave it unchanged, which
// includes the library's uncountable words (sheep, analytics, news, ...) that
// have no distinct plural: like Redocly, the rule reports them unless they are
// listed in exceptions. The
// port contains the library's complete irregular word table (person/people,
// child/children, foot/feet, schema/schemata, ...), its uncountable word list
// and uncountable patterns (information, equipment, news, series, software,
// analytics, anything ending in fish, sheep, deer or -ese, ...), its
// singularisation rules, and its pluralisation rules, which are
// used to suggest the plural in the message. Segments that are empty (a
// trailing slash) are skipped.
//
// # Where it comes from
//
// Redocly CLI's path-segment-plural rule (ignoreLastPathSegment and exceptions
// options, behaviour of pluralize.isSingular) and apilint's resource-path-plural
// rule, merged into one. apilint uses a different inflection library and scope:
// it checks only resource names and exempts the last sub-resource; the
// onlyCollectionSegments option approximates that scope.
//
// # Options
//
//   - ignoreLastPathSegment: when true the last segment of each path is not
//     checked (useful for action endpoints such as /pets/lookup). Default: false.
//   - exceptions: list of segments that are allowed to be singular, compared
//     exactly and case-sensitively as in Redocly. Default: empty. Add v1, v2 and
//     similar version segments here, as Redocly requires.
//   - onlyCollectionSegments: when true only segments that are directly
//     followed by a path parameter segment are checked, which restricts the
//     rule to resource collections. Default: false.
//
// # Bad
//
//	paths:
//	  /pet/{petId}:
//	    get:
//	      responses: {}
//
// # Good
//
//	paths:
//	  /pets/{petId}:
//	    get:
//	      responses: {}
//
// # Known false positives
//
// Like the library it ports, the inflector has no dictionary: singular words
// that end in s (status, address, bus) are judged as plural or singular by
// pattern only, and words such as "lookup", "search" or "v1" are singular and are
// reported unless listed in exceptions. Words that are plural in the domain but
// singular in English must be listed too. The port was written from the library's
// published rule tables and may differ from it for rare words.
func PathSegmentPlural() Rule {
	return New(Spec{
		Name:        "path-segment-plural",
		Description: "Literal path segments must be plural nouns.",
		HowToFix:    "Use the plural form of the path segment, or add it to the exceptions option.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        pathSegmentPluralIgnoreLastOption,
				Description: "Skip the last segment of every path.",
				Default:     false,
			},
			{
				Name:        pathSegmentPluralExceptionsOption,
				Description: "Segments that are allowed to be singular.",
				Default:     []string{},
			},
			{
				Name:        pathSegmentPluralCollectionOption,
				Description: "Check only segments followed by a path parameter.",
				Default:     false,
			},
		},
		Check: OnDocument(checkPathSegmentPlural),
	})
}

// checkPathSegmentPlural reports every literal segment of a path that is
// singular. The inflector is built once because compiling its rules is costly.
func checkPathSegmentPlural(call *Call) []Finding {
	inflector := newPathSegmentPluralInflector()

	return seq.FlatMap(oas.PathItems(call.Root), func(item oas.PathItem) []Finding {
		return pathSegmentPluralItem(call, inflector, item)
	})
}

// pathSegmentPluralItem reports the singular segments of one path.
func pathSegmentPluralItem(call *Call, inflector *pathSegmentPluralInflector, item oas.PathItem) []Finding {
	ignoreLast := call.Bool(pathSegmentPluralIgnoreLastOption)
	collectionsOnly := call.Bool(pathSegmentPluralCollectionOption)
	exceptions := call.Strings(pathSegmentPluralExceptionsOption)
	segments := strings.Split(item.Route, "/")[1:]

	return seq.FlatMapIndexed(segments, func(index int, segment string) []Finding {
		skippedAsLast := ignoreLast && index == len(segments)-1
		skippedAsMember := collectionsOnly &&
			(index+1 >= len(segments) || !pathSegmentPluralIsParameter(segments[index+1]))

		return When(
			!skippedAsLast && !skippedAsMember && segment != "" && !pathSegmentPluralIsParameter(segment) &&
				!slices.Contains(exceptions, segment) && inflector.isSingular(segment),
			func() Finding {
				return Finding{
					Node: item.KeyNode,
					Path: item.Path,
					Message: "path segment `" + segment + "` should be plural, e.g. `" + inflector.plural(
						segment,
					) + "`",
				}
			},
		)
	})
}

// pathSegmentPluralIsParameter reports whether the whole segment is a path
// parameter such as {petId}.
func pathSegmentPluralIsParameter(segment string) bool {
	return strings.HasPrefix(segment, "{") && strings.HasSuffix(segment, "}")
}

// pathSegmentPluralRule is one inflection rule: a case-insensitive pattern and
// a replacement template that may refer to groups as $0 to $9.
type pathSegmentPluralRule struct {
	pattern  *regexp.Regexp
	template string
}

// pathSegmentPluralInflector is a port of the pluralize library.
type pathSegmentPluralInflector struct {
	singles     map[string]string
	plurals     map[string]string
	uncountable []string
	pluralRules []pathSegmentPluralRule
	singleRules []pathSegmentPluralRule
}

// newPathSegmentPluralInflector builds the inflector from the rule tables.
func newPathSegmentPluralInflector() *pathSegmentPluralInflector {
	irregulars := pathSegmentPluralIrregulars()
	patterns := pathSegmentPluralCompile(pathSegmentPluralUncountablePatterns())

	return &pathSegmentPluralInflector{
		singles:     pathSegmentPluralIndex(irregulars, 0, 1),
		plurals:     pathSegmentPluralIndex(irregulars, 1, 0),
		uncountable: pathSegmentPluralUncountableWords(),
		pluralRules: slices.Concat(pathSegmentPluralCompile(pathSegmentPluralPluralRules()), patterns),
		singleRules: slices.Concat(pathSegmentPluralCompile(pathSegmentPluralSingularRules()), patterns),
	}
}

// pathSegmentPluralIndex maps the lower-cased column from of every pair to its
// lower-cased column to; a later pair wins over an earlier one.
func pathSegmentPluralIndex(pairs [][2]string, from, to int) map[string]string {
	return maps.Collect(func(yield func(string, string) bool) {
		for _, pair := range pairs {
			if !yield(strings.ToLower(pair[from]), strings.ToLower(pair[to])) {
				return
			}
		}
	})
}

// pathSegmentPluralCompile compiles rule pairs into case-insensitive rules.
func pathSegmentPluralCompile(pairs [][2]string) []pathSegmentPluralRule {
	return seq.Map(slices.Values(pairs), func(pair [2]string) pathSegmentPluralRule {
		return pathSegmentPluralRule{
			pattern:  regexp.MustCompile("(?i)" + pair[0]),
			template: pair[1],
		}
	})
}

// isSingular mirrors pluralize.isSingular.
func (i *pathSegmentPluralInflector) isSingular(word string) bool {
	token := strings.ToLower(word)

	if _, ok := i.singles[token]; ok {
		return true
	}

	if _, ok := i.plurals[token]; ok {
		return false
	}

	return i.sanitize(token, i.singleRules) == token
}

// plural mirrors pluralize.plural for lower case output.
func (i *pathSegmentPluralInflector) plural(word string) string {
	token := strings.ToLower(word)

	if plural, ok := i.singles[token]; ok {
		return plural
	}

	if _, ok := i.plurals[token]; ok {
		return token
	}

	return i.sanitize(token, i.pluralRules)
}

// sanitize applies the last matching rule, as the library does.
func (i *pathSegmentPluralInflector) sanitize(token string, rules []pathSegmentPluralRule) string {
	if token == "" || slices.Contains(i.uncountable, token) {
		return token
	}

	for _, rule := range slices.Backward(rules) {
		match := rule.pattern.FindStringSubmatchIndex(token)
		if match != nil {
			return token[:match[0]] + pathSegmentPluralExpand(rule.template, token, match) + token[match[1]:]
		}
	}

	return token
}

// pathSegmentPluralExpand interpolates $0 to $9 in the template.
func pathSegmentPluralExpand(template, token string, match []int) string {
	return regexp.MustCompile(`\$[0-9]`).ReplaceAllStringFunc(template, func(reference string) string {
		group := int(reference[1] - '0')
		if 2*group+1 < len(match) && match[2*group] >= 0 {
			return token[match[2*group]:match[2*group+1]]
		}

		return ""
	})
}

const (
	pathSegmentPluralEs = `$1es`
	pathSegmentPluralA  = `$1a`
)

// pathSegmentPluralIrregulars lists singular and plural pairs.
func pathSegmentPluralIrregulars() [][2]string {
	return [][2]string{
		{"I", "we"},
		{"me", "us"},
		{"he", "they"},
		{"she", "they"},
		{"them", "them"},
		{"myself", "ourselves"},
		{"yourself", "yourselves"},
		{"itself", pathSegmentPluralThemselves},
		{
			"herself",
			pathSegmentPluralThemselves,
		},
		{"himself", pathSegmentPluralThemselves},
		{"themself", pathSegmentPluralThemselves},
		{"is", "are"},
		{"was", "were"},
		{"has", "have"},
		{"this", "these"},
		{"that", "those"},
		{
			"my",
			"our",
		},
		{"its", pathSegmentPluralTheir},
		{"his", pathSegmentPluralTheir},
		{"her", pathSegmentPluralTheir},
		{"echo", "echoes"},
		{"dingo", "dingoes"},
		{"volcano", "volcanoes"},
		{"tornado", "tornadoes"},
		{"torpedo", "torpedoes"},
		{"genus", "genera"},
		{"viscus", "viscera"},
		{"stigma", "stigmata"},
		{"stoma", "stomata"},
		{"dogma", "dogmata"},
		{"lemma", "lemmata"},
		{"schema", "schemata"},
		{"anathema", "anathemata"},
		{"ox", "oxen"},
		{"axe", "axes"},
		{"die", "dice"},
		{"yes", "yeses"},
		{"foot", "feet"},
		{"eave", "eaves"},
		{"goose", "geese"},
		{"tooth", "teeth"},
		{"quiz", "quizzes"},
		{"human", "humans"},
		{"proof", "proofs"},
		{"carve", "carves"},
		{"valve", "valves"},
		{"looey", "looies"},
		{"thief", "thieves"},
		{"groove", "grooves"},
		{"pickaxe", "pickaxes"},
		{"passerby", "passersby"},
	}
}

// pathSegmentPluralPluralRules lists the pluralisation rules in library order.
func pathSegmentPluralPluralRules() [][2]string {
	return [][2]string{
		{`s?$`, `s`},
		{`[^\x00-\x7F]$`, `$0`},
		{`([^aeiou]ese)$`, `$1`},
		{`(ax|test)is$`, pathSegmentPluralEs},
		{`(alias|[^aou]us|t[lm]as|gas|ris)$`, pathSegmentPluralEs},
		{`(e[mn]u)s?$`, `$1s`},
		{`([^l]ias|[aeiou]las|[ejzr]as|[iu]am)$`, `$1`},
		{`(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$`, `$1i`},
		{`(alumn|alg|vertebr)(?:a|ae)$`, `$1ae`},
		{`(seraph|cherub)(?:im)?$`, `$1im`},
		{`(her|at|gr)o$`, `$1oes`},
		{
			`(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|automat|quor)(?:a|um)$`,
			pathSegmentPluralA,
		},
		{
			`(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)(?:a|on)$`,
			pathSegmentPluralA,
		},
		{`sis$`, `ses`},
		{`(?:(kni|wi|li)fe|(ar|l|ea|eo|oa|hoo)f)$`, `$1$2ves`},
		{`([^aeiouy]|qu)y$`, `$1ies`},
		{`([^ch][ieo][ln])ey$`, `$1ies`},
		{`(x|ch|ss|sh|zz)$`, pathSegmentPluralEs},
		{`(matr|cod|mur|sil|vert|ind|append)(?:ix|ex)$`, `$1ices`},
		{`\b((?:tit)?m|l)(?:ice|ouse)$`, `$1ice`},
		{`(pe)(?:rson|ople)$`, `$1ople`},
		{`(child)(?:ren)?$`, `$1ren`},
		{`eaux$`, `$0`},
		{`m[ae]n$`, `men`},
	}
}

// pathSegmentPluralSingularRules lists the singularisation rules in library
// order.
func pathSegmentPluralSingularRules() [][2]string {
	return [][2]string{
		{`s$`, ``},
		{`(ss)$`, `$1`},
		{`(wi|kni|(?:after|half|high|low|mid|non|night|[^\w]|^)li)ves$`, `$1fe`},
		{`(ar|(?:wo|[ae])l|[eo][ao])ves$`, `$1f`},
		{`ies$`, `y`},
		{`(dg|ss|ois|lk|ok|wn|mb|th|ch|ec|oal|is|ck|ix|sser|ts|wb)ies$`, `$1ie`},
		{
			`\b(l|(?:neck|cross|hog|aun)?t|coll|faer|food|gen|goon|group|hipp|junk|vegg|(?:pork)?p|charl|calor|cut)ies$`,
			`$1ie`,
		},
		{`\b(mon|smil)ies$`, `$1ey`},
		{`\b((?:tit)?m|l)ice$`, `$1ouse`},
		{`(seraph|cherub)im$`, `$1`},
		{`(x|ch|ss|sh|zz|tto|go|cho|alias|[^aou]us|t[lm]as|gas|(?:her|at|gr)o|[aeiou]ris)(?:es)?$`, `$1`},
		{`(analy|diagno|parenthe|progno|synop|the|empha|cri|ne)(?:sis|ses)$`, `$1sis`},
		{`(movie|twelve|abuse|e[mn]u)s$`, `$1`},
		{`(test)(?:is|es)$`, `$1is`},
		{`(alumn|syllab|vir|radi|nucle|fung|cact|stimul|termin|bacill|foc|uter|loc|strat)(?:us|i)$`, `$1us`},
		{
			`(agend|addend|millenni|dat|extrem|bacteri|desiderat|strat|candelabr|errat|ov|symposi|curricul|quor)a$`,
			`$1um`,
		},
		{`(apheli|hyperbat|periheli|asyndet|noumen|phenomen|criteri|organ|prolegomen|hedr|automat)a$`, `$1on`},
		{`(alumn|alg|vertebr)ae$`, pathSegmentPluralA},
		{`(cod|mur|sil|vert|ind)ices$`, `$1ex`},
		{`(matr|append)ices$`, `$1ix`},
		{`(pe)(rson|ople)$`, `$1rson`},
		{`(child)ren$`, `$1`},
		{`(eau)x?$`, `$1`},
		{`men$`, `man`},
	}
}

// pathSegmentPluralUncountablePatterns lists the uncountable patterns, which
// are applied before every other rule.
func pathSegmentPluralUncountablePatterns() [][2]string {
	return [][2]string{
		{`pok[eé]mon$`, `$0`},
		{`[^aeiou]ese$`, `$0`},
		{`deer$`, `$0`},
		{`fish$`, `$0`},
		{`measles$`, `$0`},
		{`o[iu]s$`, `$0`},
		{`pox$`, `$0`},
		{`sheep$`, `$0`},
	}
}

// pathSegmentPluralUncountableWords lists the words that have no plural.
func pathSegmentPluralUncountableWords() []string {
	return []string{
		"adulthood", "advice", "agenda", "aid", "aircraft", "alcohol", "ammo", "analytics", "anime",
		"athletics", "audio", "bison", "blood", "bream", "buffalo", "butter", "carp", "cash", "chassis",
		"chess", "clothing", "cod", "commerce", "cooperation", "corps", "debris", "diabetes", "digestion",
		"elk", "energy", "equipment", "excretion", "expertise", "firmware", "flounder", "fun", "gallows",
		"garbage", "graffiti", "hardware", "headquarters", "health", "herpes", "highjinks", "homework",
		"housework", "information", "jeans", "justice", "kudos", "labour", "literature", "machinery",
		"mackerel", "mail", "media", "mews", "moose", "music", "mud", "manga", "news", "only", "personnel",
		"pike", "plankton", "pliers", "police", "pollution", "premises", "rain", "research", "rice",
		"salmon", "scissors", "series", "sewage", "shambles", "shrimp", "software", "staff", "swine",
		"tennis", "traffic", "transportation", "trout", "tuna", "wealth", "welfare", "whiting",
		"wildebeest", "wildlife", "you",
	}
}

const (
	pathSegmentPluralThemselves = "themselves"
	pathSegmentPluralTheir      = "their"
)

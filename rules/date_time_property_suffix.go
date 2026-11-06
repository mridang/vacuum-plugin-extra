package rules

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// dateTimePropertySuffixPatterns is the name of the option that lists the
// accepted name patterns.
const dateTimePropertySuffixPatterns = "patterns"

// dateTimePropertySuffixFormatTime is the time format of a string.
const dateTimePropertySuffixFormatTime = "time"

// dateTimePropertySuffixDefaults returns the default name patterns of the
// original rule.
func dateTimePropertySuffixDefaults() []string {
	return []string{
		`.+_at`,
		`(.+_)*time(stamp)?(_.+)?`,
		`(.+_)*date(_.+)*`,
		`(.+_)*day(_.+)*`,
	}
}

// DateTimePropertySuffix returns the extra-date-time-property-suffix rule.
//
// # What it checks
//
// Every property that is a string with format date, time or date-time must have
// a name that matches, in full, at least one of the configured patterns. The
// defaults are .+_at, (.+_)*time(stamp)?(_.+)?, (.+_)*date(_.+)*
// and (.+_)*day(_.+)*, so created_at, start_time, last_timestamp, birth_date
// and day_of_week are accepted, while created, createdAt and opens are
// reported. Matching is case-sensitive and the same list applies to the three
// formats. The property may be declared inline or through a $ref to a schema
// that declares type and format.
//
// Consistent names (created_at, birth_date, start_time) let readers and
// generated clients distinguish timestamps, calendar dates and times of day
// without opening the schema.
//
// # Where it comes from
//
// Zally rule 235, "Name date/time properties using the common conventions"
// (DateTimePropertiesSuffixRule), whose DateTimePropertiesSuffixRule.patterns
// setting holds the default patterns and whose severity is SHOULD.
//
// # Options
//
//   - patterns (list of strings, default the four patterns above): regular
//     expressions, each of which must match the whole property name. An empty
//     list disables the rule, and a pattern that is not a valid regular
//     expression is ignored.
//
// # Bad
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        created:
//	          type: string
//	          format: date-time
//
// # Good
//
//	components:
//	  schemas:
//	    Pet:
//	      type: object
//	      properties:
//	        created_at:
//	          type: string
//	          format: date-time
//
// # Known false positives / limitations
//
// Names that are clear without a suffix (for example "timestamp" alone is
// accepted, but "modified" is not) are reported unless the patterns are widened.
// Zally only looks at leaf properties; this rule looks at every string property
// found by the schema walker, which is the same set in practice. Patterns use
// the Go regexp syntax (RE2).
func DateTimePropertySuffix() Rule {
	return New(Spec{
		Name:        "date-time-property-suffix",
		Description: "String properties with a date, time or date-time format must have a matching name.",
		HowToFix:    "Rename the property so that it matches one of the configured name patterns.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        dateTimePropertySuffixPatterns,
				Description: "Regular expressions of which one must match the whole name of such a property.",
				Default:     dateTimePropertySuffixDefaults(),
			},
		},
		Check: PerSchema(checkDateTimePropertySuffix),
	})
}

// checkDateTimePropertySuffix reports temporal properties with an unexpected name.
func checkDateTimePropertySuffix(call *Call, schema oas.Schema) []Finding {
	patterns := dateTimePropertySuffixCompile(call.Strings(dateTimePropertySuffixPatterns))
	if len(patterns) == 0 {
		return nil
	}

	formats := []string{formatDate, dateTimePropertySuffixFormatTime, formatDateTime}

	return seq.FlatMapPairs(oas.Pairs(oas.Get(schema.Node, "properties")), func(name, property *yaml.Node) []Finding {
		resolved := oas.Resolve(call.Root, property)
		format := oas.StringAt(resolved, "format")

		return When(
			slices.Contains(formats, format) && oas.HasType(resolved, "string") &&
				!dateTimePropertySuffixMatches(patterns, name.Value),
			func() Finding {
				return Finding{
					Node: name,
					Path: oas.JoinAll(schema.Path, "properties", name.Value),
					Message: fmt.Sprintf(
						"property %q of type \"string\" and format %q should match one of the patterns %v",
						name.Value, format, dateTimePropertySuffixSources(patterns),
					),
				}
			},
		)
	})
}

func dateTimePropertySuffixCompile(sources []string) []*regexp.Regexp {
	return seq.FlatMapSlice(sources, func(source string) []*regexp.Regexp {
		expression, err := regexp.Compile(`\A(?:` + source + `)\z`)
		if err != nil {
			return nil
		}

		return []*regexp.Regexp{expression}
	})
}

func dateTimePropertySuffixMatches(patterns []*regexp.Regexp, name string) bool {
	return slices.ContainsFunc(patterns, func(pattern *regexp.Regexp) bool {
		return pattern.MatchString(name)
	})
}

func dateTimePropertySuffixSources(patterns []*regexp.Regexp) []string {
	return seq.Map(slices.Values(patterns), func(pattern *regexp.Regexp) string {
		source := pattern.String()

		return source[len(`\A(?:`) : len(source)-len(`)\z`)]
	})
}

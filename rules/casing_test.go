package rules_test

import (
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/casing"
	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

// casingSample is one name written in a convention.
type casingSample struct {
	name string
	// accepted lists the conventions the name follows.
	accepted []casing.Casing
}

// casingSamples holds a name per convention. The flat name follows several.
func casingSamples() []casingSample {
	return []casingSample{
		{"pet-owner", []casing.Casing{casing.Kebab}},
		{"pet_owner", []casing.Casing{casing.Snake}},
		{"petOwner", []casing.Casing{casing.Camel}},
		{"PetOwner", []casing.Casing{casing.Pascal}},
		{"PET_OWNER", []casing.Casing{casing.ScreamingSnake}},
		{"PET-OWNER", []casing.Casing{casing.ScreamingKebab}},
		{"petowner", []casing.Casing{casing.Kebab, casing.Snake, casing.Camel, casing.Lower}},
	}
}

// casingRule describes how one casing rule sees a name.
type casingRule struct {
	rule rules.Rule
	// def is the default convention, which a sample of another casing breaks.
	def []casing.Casing
	// document puts every name in a document the rule checks.
	document func(names []string) string
	// path is the JSONPath of the finding for a name.
	path func(index int, name string) string
}

func casingRules() []casingRule {
	return []casingRule{
		{
			rule: rules.ComponentSchemaNameCase(),
			def:  []casing.Casing{casing.Pascal},
			document: func(names []string) string {
				return ruletest.Doc30(
					"paths: {}\ncomponents:\n  schemas:\n" + lines(names, "    %s:\n      type: object\n"),
				)
			},
			path: func(_ int, name string) string { return "$.components.schemas" + member(name) },
		},
		{
			rule: rules.OperationIDCasing(),
			def:  []casing.Casing{casing.Snake},
			document: func(names []string) string {
				return ruletest.Doc30("paths:\n" + indexed(
					names,
					"  /p%[1]d:\n    get:\n      operationId: %[2]s\n      responses:\n        \"200\":\n          description: ok\n",
				))
			},
			path: func(index int, _ string) string { return fmt.Sprintf("$.paths['/p%d'].get.operationId", index) },
		},
		{
			rule: rules.PathParameterCase(),
			def:  []casing.Casing{casing.Snake},
			document: func(names []string) string {
				return ruletest.Doc30("paths:\n" + lines(names,
					"  /pets/{%s}:\n    get:\n      responses:\n        \"200\":\n          description: ok\n"))
			},
			path: func(_ int, name string) string { return "$.paths['/pets/{" + name + "}']" },
		},
		{
			rule: rules.PathSegmentCase(),
			def:  []casing.Casing{casing.Kebab},
			document: func(names []string) string {
				return ruletest.Doc30("paths:\n" + lines(names,
					"  /%s:\n    get:\n      responses:\n        \"200\":\n          description: ok\n"))
			},
			path: func(_ int, name string) string { return "$.paths['/" + name + "']" },
		},
		{
			rule: rules.EnumValueCase(),
			def:  []casing.Casing{casing.Snake, casing.ScreamingSnake},
			document: func(names []string) string {
				return ruletest.Doc30("components:\n  schemas:\n    Status:\n      type: string\n      enum:\n" +
					lines(names, "        - %s\n"))
			},
			path: func(index int, _ string) string { return fmt.Sprintf("$.components.schemas.Status.enum[%d]", index) },
		},
		{
			rule: rules.ExampleBodyKeyCase(),
			def:  []casing.Casing{casing.Snake},
			document: func(names []string) string {
				return ruletest.Doc30("paths:\n  /pets:\n    get:\n      responses:\n        \"200\":\n" +
					"          description: ok\n          content:\n            application/json:\n" +
					"              example:\n" + lines(names, "                %s: 1\n"))
			},
			path: func(_ int, name string) string {
				return "$.paths['/pets'].get.responses['200'].content['application/json'].example" + member(name)
			},
		},
	}
}

// member renders a mapping key as a JSONPath segment, bracketed when it has a hyphen.
func member(name string) string {
	if strings.Contains(name, "-") {
		return "['" + name + "']"
	}

	return "." + name
}

func lines(names []string, format string) string {
	return strings.Join(seq.Map(slices.Values(names), func(name string) string {
		return fmt.Sprintf(format, name)
	}), "")
}

func indexed(names []string, format string) string {
	return strings.Join(seq.FlatMapIndexed(names, func(i int, name string) []string {
		return []string{fmt.Sprintf(format, i, name)}
	}), "")
}

func sampleNames() []string {
	return seq.Map(slices.Values(casingSamples()), func(sample casingSample) string {
		return sample.name
	})
}

// violations returns the paths of the samples that break every convention in allowed.
func (c casingRule) violations(allowed []casing.Casing) []string {
	return seq.FlatMapIndexed(casingSamples(), func(i int, sample casingSample) []string {
		followed := slices.ContainsFunc(allowed, func(convention casing.Casing) bool {
			return slices.Contains(sample.accepted, convention)
		})

		if followed {
			return nil
		}

		return []string{c.path(i, sample.name)}
	})
}

func TestSharedCasingOption(t *testing.T) {
	t.Parallel()

	for _, subject := range casingRules() {
		t.Run(subject.rule.ID(), func(t *testing.T) {
			t.Parallel()

			document := subject.document(sampleNames())

			t.Run("declares the shared option with its default", func(t *testing.T) {
				t.Parallel()

				options := subject.rule.Spec().Options

				require.Equal(t, rules.CasingOption, options[0].Name)
				require.Contains(t, options[0].Description, "kebab-case")
				require.Len(t, options[0].Default, len(subject.def))
				ruletest.RequirePaths(t, subject.rule, document, nil, subject.violations(subject.def)...)
			})

			for _, convention := range casing.All() {
				t.Run("enforces "+string(convention), func(t *testing.T) {
					t.Parallel()

					subject.requireEnforced(t, document, convention)
				})
			}

			t.Run("accepts a list of conventions", func(t *testing.T) {
				t.Parallel()

				subject.requireList(t, document)
			})

			t.Run("an empty list disables the check", func(t *testing.T) {
				t.Parallel()

				ruletest.RequireClean(t, subject.rule, document, ruletest.Options{rules.CasingOption: []string{}})
			})

			t.Run("an unknown value is reported with the allowed values", func(t *testing.T) {
				t.Parallel()

				subject.requireUnknownReported(t, document)
			})
		})
	}
}

func (c casingRule) requireEnforced(t *testing.T, document string, convention casing.Casing) {
	t.Helper()

	results := ruletest.Run(t, c.rule, document, ruletest.Options{rules.CasingOption: string(convention)})

	require.NotEmpty(t, results)
	require.ElementsMatch(t, c.violations([]casing.Casing{convention}), ruletest.Paths(results))

	for _, message := range ruletest.Messages(results) {
		require.Contains(t, message, string(convention))
	}
}

func (c casingRule) requireList(t *testing.T, document string) {
	t.Helper()

	options := ruletest.Options{rules.CasingOption: []any{"PascalCase", "kebab-case"}}
	results := ruletest.Run(t, c.rule, document, options)

	require.ElementsMatch(t, c.violations([]casing.Casing{casing.Pascal, casing.Kebab}), ruletest.Paths(results))

	for _, message := range ruletest.Messages(results) {
		require.Contains(t, message, "PascalCase or kebab-case")
	}
}

func (c casingRule) requireUnknownReported(t *testing.T, document string) {
	t.Helper()

	for _, value := range []any{"snake", []any{"snake_case", "shouty"}} {
		results := ruletest.Run(t, c.rule, document, ruletest.Options{rules.CasingOption: value})

		require.Equal(t, []string{"$"}, ruletest.Paths(results))
		require.Contains(t, results[0].Message, "option casing: unknown casing")

		for _, name := range casing.Names() {
			require.Contains(t, results[0].Message, name)
		}
	}
}

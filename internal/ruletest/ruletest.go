package ruletest

import (
	"log/slog"
	"slices"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/motor"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/rules"
)

// Options are the functionOptions handed to a rule under test.
type Options map[string]any

// Run lints document with only the given rule enabled and returns the results.
// Options override the defaults of the rule; a nil Options keeps the defaults.
func Run(t *testing.T, rule rules.Rule, document string, options Options) []model.RuleFunctionResult {
	t.Helper()

	entry := rule.RuleEntryWith(options)

	result := motor.ApplyRulesToRuleSet(&motor.RuleSetExecution{
		RuleSet:           &rulesets.RuleSet{Rules: map[string]*model.Rule{rule.ID(): entry}},
		Spec:              []byte(strings.TrimSpace(document) + "\n"),
		CustomFunctions:   map[string]model.RuleFunction{rule.ID(): rule},
		SilenceLogs:       true,
		SkipDocumentCheck: true,
		Logger:            slog.New(slog.DiscardHandler),
	})
	defer result.ReleaseOwnedResources()
	require.Empty(t, result.Errors, "engine errors")

	return result.Results
}

// Messages returns the messages of the results.
func Messages(results []model.RuleFunctionResult) []string {
	return slices.AppendSeq(make([]string, 0, len(results)), func(yield func(string) bool) {
		for _, result := range results {
			if !yield(result.Message) {
				return
			}
		}
	})
}

// Paths returns the JSONPaths of the results.
func Paths(results []model.RuleFunctionResult) []string {
	return slices.AppendSeq(make([]string, 0, len(results)), func(yield func(string) bool) {
		for _, result := range results {
			if !yield(result.Path) {
				return
			}
		}
	})
}

// RequireClean fails the test when the document violates the rule.
func RequireClean(t *testing.T, rule rules.Rule, document string, options Options) {
	t.Helper()
	require.Empty(t, Messages(Run(t, rule, document, options)), "expected no violations")
}

// RequirePaths fails the test unless the rule reports exactly the given paths.
func RequirePaths(t *testing.T, rule rules.Rule, document string, options Options, want ...string) {
	t.Helper()
	require.ElementsMatch(t, want, Paths(Run(t, rule, document, options)))
}

// Doc30 prefixes body with a minimal OpenAPI 3.0.3 header. Body holds the
// remaining top-level keys and must not repeat openapi or info.
func Doc30(body string) string {
	return "openapi: 3.0.3\ninfo:\n  title: Test\n  version: \"1.0\"\n" + strings.TrimSpace(body) + "\n"
}

// Doc31 prefixes body with a minimal OpenAPI 3.1.0 header. Body holds the
// remaining top-level keys and must not repeat openapi or info.
func Doc31(body string) string {
	return "openapi: 3.1.0\ninfo:\n  title: Test\n  version: \"1.0\"\n" + strings.TrimSpace(body) + "\n"
}

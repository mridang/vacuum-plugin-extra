package extra_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/daveshanley/vacuum/rulesets"
	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"

	extra "github.com/mridang/vacuum-plugin-extra"
	"github.com/mridang/vacuum-plugin-extra/internal/ruleset"
	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestRulesetIsUpToDate(t *testing.T) {
	t.Parallel()

	want, err := ruleset.Render()
	require.NoError(t, err)

	got, err := os.ReadFile("ruleset.yaml")
	require.NoError(t, err)

	// dprint reformats the file, so compare the parsed content, not the bytes.
	require.Equal(t, parseYAML(t, want), parseYAML(t, got), "ruleset.yaml is stale: run go generate ./...")
	require.Equal(t, parseYAML(t, got), parseYAML(t, extra.Ruleset()), "embedded ruleset differs from ruleset.yaml")
}

func parseYAML(t *testing.T, data []byte) any {
	t.Helper()

	var out any
	require.NoError(t, yaml.Unmarshal(data, &out))

	return out
}

func TestCatalogueMetadata(t *testing.T) {
	t.Parallel()

	kebab := regexp.MustCompile(`^extra-[a-z0-9]+(-[a-z0-9]+)*$`)
	seen := map[string]bool{}
	severities := []string{model.SeverityError, model.SeverityWarn, model.SeverityInfo, model.SeverityHint}

	for _, rule := range extra.Rules() {
		spec := rule.Spec()
		require.Regexp(t, kebab, rule.ID())
		require.False(t, seen[rule.ID()], "duplicate rule %s", rule.ID())
		seen[rule.ID()] = true

		require.NotEmpty(t, spec.Description, rule.ID())
		require.NotEmpty(t, spec.HowToFix, rule.ID())
		require.NotEmpty(t, spec.Given, rule.ID())
		require.Contains(t, severities, spec.Severity, rule.ID())
		require.Contains(t, []string{"style", "validation"}, spec.Type, rule.ID())
		require.NotNil(t, spec.Check, rule.ID())
		require.Equal(t, rule.ID(), rule.GetSchema().Name)
	}

	require.GreaterOrEqual(t, len(seen), 160)
}

func TestRegisterAddsEveryRule(t *testing.T) {
	t.Parallel()

	manager := plugin.CreatePluginManager()
	extra.Register(manager)

	require.Equal(t, len(rules.All()), manager.LoadedFunctionCount())
	require.Contains(t, manager.GetCustomFunctions(), "extra-summary-no-trailing-period")
}

// fixtureDir holds, for every rule, a document that violates it and, for the
// rules that are inert without configuration, the functionOptions to run it with.
const fixtureDir = "testdata/fire"

func TestEveryRuleFires(t *testing.T) {
	t.Parallel()

	for _, rule := range extra.Rules() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()

			name := filepath.Join(fixtureDir, rules.ShortName(rule.ID()))

			document, err := os.ReadFile(name + ".yaml")
			require.NoError(t, err, "add a violating document for %s", rule.ID())

			results := ruletest.Run(t, rule, string(document), fixtureOptions(t, name+".options.json"))

			require.NotEmpty(t, results, "%s did not report its fixture", rule.ID())
		})
	}
}

func TestEveryFixtureBelongsToARule(t *testing.T) {
	t.Parallel()

	known := map[string]bool{}
	for _, rule := range extra.Rules() {
		known[rules.ShortName(rule.ID())] = true
	}

	entries, err := os.ReadDir(fixtureDir)
	require.NoError(t, err)

	for _, entry := range entries {
		name := strings.TrimSuffix(strings.TrimSuffix(entry.Name(), ".json"), ".yaml")
		name = strings.TrimSuffix(name, ".options")
		require.True(t, known[name], "fixture %s has no rule", entry.Name())
	}
}

func fixtureOptions(t *testing.T, path string) ruletest.Options {
	t.Helper()

	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}

	require.NoError(t, err)

	var options ruletest.Options
	require.NoError(t, json.Unmarshal(data, &options))

	return options
}

func TestEmbeddedRulesetKeepsRefsUnresolved(t *testing.T) {
	t.Parallel()

	set, err := rulesets.CreateRuleSetFromData(extra.Ruleset())
	require.NoError(t, err)
	require.Len(t, set.Rules, len(rules.All()))

	for id, rule := range set.Rules {
		require.False(t, rule.Resolved, "%s must set resolved: false", id)
	}
}

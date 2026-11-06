package extra_test

import (
	"os"
	"regexp"
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/daveshanley/vacuum/plugin"
	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"

	extra "github.com/mridang/vacuum-plugin-extra"
	"github.com/mridang/vacuum-plugin-extra/internal/ruleset"
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

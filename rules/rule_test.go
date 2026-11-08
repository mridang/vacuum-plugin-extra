package rules_test

import (
	"testing"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"
	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestEveryRuleDoesNothingWhenSwitchedOff(t *testing.T) {
	t.Parallel()

	var root yaml.Node
	require.NoError(t, yaml.Unmarshal([]byte("openapi: 3.0.3\ninfo: {title: t, version: '1'}\npaths: {}\n"), &root))

	for _, rule := range rules.All() {
		t.Run(rule.ID(), func(t *testing.T) {
			t.Parallel()

			ctx := model.RuleFunctionContext{Rule: &model.Rule{Severity: rules.SeverityOff}}

			require.Empty(t, rule.RunRule([]*yaml.Node{root.Content[0]}, ctx))
		})
	}
}

func TestRuleRunsUnlessSwitchedOff(t *testing.T) {
	t.Parallel()

	rule := rules.SummaryNoTrailingPeriod()

	var doc yaml.Node
	spec := "openapi: 3.0.3\npaths:\n  /pets:\n    get:\n      summary: List pets.\n"
	require.NoError(t, yaml.Unmarshal([]byte(spec), &doc))

	node := doc.Content[0]

	for severity, want := range map[string]int{"error": 1, "warn": 1, "info": 1, "hint": 1, "off": 0} {
		t.Run(severity, func(t *testing.T) {
			t.Parallel()

			ctx := model.RuleFunctionContext{Rule: &model.Rule{Severity: severity}}

			require.Len(t, rule.RunRule([]*yaml.Node{node}, ctx), want)
		})
	}
}

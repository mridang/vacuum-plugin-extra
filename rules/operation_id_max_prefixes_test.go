package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const operationIDMaxPrefixesDoc = `
paths:
  /a:
    get: {operationId: getA, responses: {"200": {description: ok}}}
    post: {operationId: createA, responses: {"200": {description: ok}}}
  /b:
    get: {operationId: fetchB, responses: {"200": {description: ok}}}
    put: {operationId: updateB, responses: {"200": {description: ok}}}
    delete: {operationId: remove_b, responses: {"200": {description: ok}}}
  /c:
    get: {operationId: listC, responses: {"200": {description: ok}}}
    post: {operationId: getC2, responses: {"200": {description: ok}}}
`

func TestOperationIDMaxPrefixes(t *testing.T) {
	t.Parallel()

	rule := rules.OperationIDMaxPrefixes()

	t.Run("flags more prefixes than allowed", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc), ruletest.Options{"max": 3})

		require.Len(t, results, 1)
		require.Equal(t, "$", results[0].Path)
		require.Contains(t, results[0].Message, "create, fetch, get, list, update")
	})

	t.Run("accepts the default limit", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc), nil)
	})

	t.Run("flags documents above the default of five", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc), ruletest.Options{"max": 4}, "$")
	})

	t.Run("reads the prefix in the configured casing", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc),
			ruletest.Options{"casing": "snake_case", "max": 0})

		require.Len(t, results, 1)
		require.Contains(t, results[0].Message, "found 1 distinct operationId prefixes (remove)")
	})

	t.Run("reads each identifier in the first casing that captures a prefix", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc),
			ruletest.Options{"casing": []string{"camelCase", "snake_case"}, "max": 5})

		require.Len(t, results, 1)
		require.Contains(t, results[0].Message, "create, fetch, get, list, remove, update")
	})

	t.Run("collects nothing for lowercase or an empty list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc),
			ruletest.Options{"casing": "lowercase", "max": 0})
		ruletest.RequireClean(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc),
			ruletest.Options{"casing": []string{}, "max": 0})
	})

	t.Run("reports an unknown casing", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(operationIDMaxPrefixesDoc),
			ruletest.Options{"casing": "bogus"}, "$")
	})
}

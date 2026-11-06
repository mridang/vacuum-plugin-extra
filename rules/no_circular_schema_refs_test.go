package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

// noCircularSchemaRefsRun runs the rule and keeps only its own results, because
// the engine reports its own circular-references result for the same documents.
func noCircularSchemaRefsRun(t *testing.T, document string, options ruletest.Options) ([]string, []string) {
	t.Helper()

	rule := rules.NoCircularSchemaRefs()

	var paths, messages []string
	for _, result := range ruletest.Run(t, rule, document, options) {
		if result.RuleId != rule.ID() {
			continue
		}

		paths = append(paths, result.Path)
		messages = append(messages, result.Message)
	}

	return paths, messages
}

func TestNoCircularSchemaRefs(t *testing.T) {
	t.Parallel()

	t.Run("flags a direct self reference", func(t *testing.T) {
		t.Parallel()

		paths, messages := noCircularSchemaRefsRun(t, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Node:
      type: object
      properties:
        parent:
          $ref: "#/components/schemas/Node"
`), nil)

		require.Equal(t, []string{"$.components.schemas.Node"}, paths)
		require.Contains(t, messages[0], "Node -> Node")
	})

	t.Run("flags every schema on an indirect cycle", func(t *testing.T) {
		t.Parallel()

		paths, _ := noCircularSchemaRefsRun(t, ruletest.Doc30(`
paths: {}
components:
  schemas:
    A:
      allOf:
        - $ref: "#/components/schemas/B"
    B:
      type: array
      items:
        $ref: "#/components/schemas/A"
    C:
      type: object
      properties:
        a:
          $ref: "#/components/schemas/A"
`), nil)

		require.ElementsMatch(t, []string{"$.components.schemas.A", "$.components.schemas.B"}, paths)
	})

	t.Run("accepts acyclic schemas", func(t *testing.T) {
		t.Parallel()

		paths, _ := noCircularSchemaRefsRun(t, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Owner:
      type: object
    Pet:
      type: object
      properties:
        owner:
          $ref: "#/components/schemas/Owner"
`), nil)

		require.Empty(t, paths)
	})

	t.Run("ignoreProperties skips property edges only", func(t *testing.T) {
		t.Parallel()

		paths, _ := noCircularSchemaRefsRun(t, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Tree:
      type: object
      properties:
        child:
          $ref: "#/components/schemas/Tree"
    List:
      type: array
      items:
        $ref: "#/components/schemas/List"
`), ruletest.Options{"ignoreProperties": true})

		require.Equal(t, []string{"$.components.schemas.List"}, paths)
	})
}

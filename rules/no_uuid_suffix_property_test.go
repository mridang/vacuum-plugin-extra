package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestNoUUIDSuffixProperty(t *testing.T) {
	t.Parallel()

	rule := rules.NoUUIDSuffixProperty()

	t.Run("flags a plain uuid property named with _uuid", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Instance:
      type: object
      properties:
        project_uuid:
          type: string
          format: uuid
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$.components.schemas.Instance.properties.project_uuid", results[0].Path)
		require.Contains(t, results[0].Message, "project_id")
	})

	t.Run("accepts _id names", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Instance:
      type: object
      properties:
        project_id:
          type: string
          format: uuid
`), nil)
	})

	t.Run("ignores constrained strings, references and other formats", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths: {}
components:
  schemas:
    Uuid:
      type: string
      format: uuid
    Instance:
      type: object
      properties:
        a_uuid:
          type: string
          format: uuid
          pattern: "^x"
        b_uuid:
          $ref: "#/components/schemas/Uuid"
        c_uuid:
          type: string
        d_uuid:
          type: string
          format: uuid
          enum: ["00000000-0000-0000-0000-000000000000"]
`), nil)
	})
}

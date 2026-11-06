package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func operationDescriptionOpensWithSentenceDoc(description string) string {
	return ruletest.Doc30(`
paths:
  /pets:
    get:
      description: ` + description + `
      responses:
        "200":
          description: ok
`)
}

func TestOperationDescriptionOpensWithSentence(t *testing.T) {
	t.Parallel()

	rule := rules.OperationDescriptionOpensWithSentence()

	t.Run("flags descriptions opening with markup", func(t *testing.T) {
		t.Parallel()

		openings := []string{
			`"## Overview"`, `"> quoted"`, `"- item"`, `"* item"`, `"1. item"`,
			`"| a | b |"`, "\"```json\"", `"<div>x</div>"`, `"~~~"`, `"\n    code"`,
		}

		for _, opening := range openings {
			results := ruletest.Run(t, rule, operationDescriptionOpensWithSentenceDoc(opening), nil)

			require.Len(t, results, 1, opening)
			require.Equal(t, "$.paths['/pets'].get.description", results[0].Path)
		}
	})

	t.Run("accepts a plain sentence followed by markup", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      description: |
        Lists the pets.

        ## Details

        - one
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("looks at the first non blank line of a block scalar", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      description: |

        # Title
        text
      responses:
        "200":
          description: ok
`), nil, "$.paths['/pets'].get.description")
	})

	t.Run("ignores operations without a description and hash tags", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc31(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
    post:
      description: "#hashtag is not a heading"
      responses:
        "200":
          description: ok
`), nil)
	})
}

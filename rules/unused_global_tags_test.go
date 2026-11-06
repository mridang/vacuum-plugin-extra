package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestUnusedGlobalTags(t *testing.T) {
	t.Parallel()

	rule := rules.UnusedGlobalTags()

	t.Run("flags tags that no operation uses", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
tags:
  - name: pets
  - name: orders
  - name: owners
paths:
  /pets:
    get:
      tags: [pets]
      responses:
        "200":
          description: ok
`), nil)

		require.ElementsMatch(t, []string{"$.tags[1].name", "$.tags[2].name"}, ruletest.Paths(results))
		require.Contains(t, ruletest.Messages(results)[0], "not used")
	})

	t.Run("accepts tags used by any method", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
tags:
  - name: pets
  - name: orders
paths:
  /pets:
    get:
      tags: [pets]
      responses:
        "200":
          description: ok
    post:
      tags: [orders, other]
      responses:
        "201":
          description: ok
`), nil)
	})

	t.Run("counts tags used by webhook operations", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc31(`
tags:
  - name: hooks
  - name: unused
webhooks:
  newPet:
    post:
      tags: [hooks]
      responses:
        "200":
          description: ok
paths: {}
`), nil, "$.tags[1].name")
	})

	t.Run("does nothing without a global tag list", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      tags: [pets]
      responses:
        "200":
          description: ok
`), nil)
	})
}

package rules_test

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestAPIVersionInURL(t *testing.T) {
	t.Parallel()

	rule := rules.APIVersionInURL()

	t.Run("flags a document without any version", func(t *testing.T) {
		t.Parallel()

		results := ruletest.Run(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)

		require.Len(t, results, 1)
		require.Equal(t, "$", results[0].Path)
	})

	t.Run("accepts a version in a server url", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com/v1
paths: {}
`), nil)
	})

	t.Run("accepts a version in a path", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /v2/pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("accepts a version server variable", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com/{version}
    variables:
      version:
        default: v1
paths: {}
`), nil)
	})

	t.Run("accepts a version request header", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      parameters:
        - name: Api-Version
          in: header
          schema:
            type: string
      responses:
        "200":
          description: ok
`), nil)
	})

	t.Run("accepts a version response header", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          headers:
            X-API-Version:
              schema:
                type: string
`), nil)
	})

	t.Run("does not treat words starting with v as versions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
servers:
  - url: https://api.example.com/vets
paths:
  /vaccines:
    get:
      responses:
        "200":
          description: ok
`), nil, "$")
	})
}

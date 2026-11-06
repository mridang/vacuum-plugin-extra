package rules_test

import (
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

const serverVariablesConsistentDoc = `
servers:
  - url: https://{region}.example.com/{stage}
    variables:
      region:
        default: ""
      stage:
        default: prod
        enum: [dev, test]
      extra:
        default: x
        enum: []
paths:
  /pets:
    servers:
      - url: https://{host}
    get:
      servers:
        - url: https://ok.example.com
          variables:
            v:
              default: a
              enum: [a, b]
      responses:
        "200":
          description: ok
`

func TestServerVariablesConsistent(t *testing.T) {
	t.Parallel()

	rule := rules.ServerVariablesConsistent()

	t.Run("flags defaults, enums and placeholders", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(serverVariablesConsistentDoc), nil,
			"$.servers[0].variables.region",
			"$.servers[0].variables.stage",
			"$.servers[0].variables.extra",
			"$.paths['/pets'].servers[0]")
	})

	t.Run("requireEnum flags variables without an enum", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
servers:
  - url: https://{region}.example.com
    variables:
      region:
        default: eu
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), ruletest.Options{"requireEnum": true}, "$.servers[0].variables.region")
	})

	t.Run("accepts consistent servers", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
servers:
  - url: https://{region}.example.com
    variables:
      region:
        default: eu
        enum: [eu, us]
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
`), nil)
	})
}

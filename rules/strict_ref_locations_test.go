package rules_test

import (
	"fmt"
	"testing"

	"github.com/mridang/vacuum-plugin-extra/internal/ruletest"
	"github.com/mridang/vacuum-plugin-extra/rules"
)

func TestStrictRefLocations(t *testing.T) {
	t.Parallel()

	rule := rules.StrictRefLocations()

	t.Run("flags references where the specification forbids them", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, `
openapi: 3.0.3
info:
  title: Test
  version: "1.0"
  contact:
    $ref: "#/components/schemas/Pet"
paths:
  /pets:
    get:
      $ref: "#/components/schemas/Pet"
      responses:
        "200":
          description: ok
          content:
            application/json:
              $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet:
      type: object
`, nil,
			"$.info.contact['$ref']",
			"$.paths['/pets'].get['$ref']",
			"$.paths['/pets'].get.responses['200'].content['application/json']['$ref']")
	})

	t.Run("accepts references at legal locations", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
paths:
  /owners:
    get:
      parameters:
        - $ref: "#/components/parameters/Id"
      responses:
        "200":
          $ref: "#/components/responses/Ok"
components:
  parameters:
    Id:
      name: id
      in: query
      schema:
        $ref: "#/components/schemas/Name"
  responses:
    Ok:
      description: ok
  schemas:
    Pet:
      type: object
      properties:
        content:
          $ref: "#/components/schemas/Name"
    Name:
      type: string
`), nil)
	})

	t.Run("flags references on operations, responses maps and component maps", func(t *testing.T) {
		t.Parallel()

		ruletest.RequirePaths(t, rule, ruletest.Doc30(`
paths:
  /pets:
    get:
      responses:
        $ref: "#/components/schemas/Pet"
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              discriminator:
                $ref: "#/components/schemas/Pet"
              xml:
                $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet: {type: object}
    $ref: "#/components/schemas/Pet"
  securitySchemes:
    Oauth:
      type: oauth2
      flows:
        implicit:
          $ref: "#/components/schemas/Pet"
servers:
  - url: /
    variables:
      v:
        $ref: "#/components/schemas/Pet"
`), nil,
			"$.paths['/pets'].get.responses['$ref']",
			"$.paths['/pets'].post.requestBody.content['application/json'].schema.discriminator['$ref']",
			"$.paths['/pets'].post.requestBody.content['application/json'].schema.xml['$ref']",
			"$.components.schemas['$ref']",
			"$.components.securitySchemes.Oauth.flows.implicit['$ref']",
			"$.servers[0].variables.v['$ref']",
		)
	})

	t.Run("allows media type references only in OpenAPI 3.2", func(t *testing.T) {
		t.Parallel()

		doc := `
openapi: %s
info: {title: t, version: "1"}
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet: {type: object}
`

		ruletest.RequirePaths(t, rule, fmt.Sprintf(doc, "3.1.0"), nil,
			"$.paths['/pets'].get.responses['200'].content['application/json']['$ref']")
		ruletest.RequireClean(t, rule, fmt.Sprintf(doc, "3.2.0"), nil)
	})

	t.Run("ignores free-form values and extensions", func(t *testing.T) {
		t.Parallel()

		ruletest.RequireClean(t, rule, ruletest.Doc30(`
info:
  x-thing:
    $ref: "#/components/schemas/Pet"
components:
  schemas:
    Pet: {type: object}
paths:
  /pets:
    get:
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                example:
                  $ref: "#/components/schemas/Pet"
                default:
                  $ref: "#/components/schemas/Pet"
                properties:
                  $ref:
                    type: string
`), nil)
	})
}

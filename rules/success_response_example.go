package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// successResponseExampleJSON is the default media type.
const successResponseExampleJSON = "application/json"

// SuccessResponseExample returns the extra-success-response-example rule.
//
// # What it checks
//
// Every media type of a success (2xx) response, in the list given by the
// mediaTypes option, must carry an example. An example is an example or
// examples keyword on the media type object, or an example or examples keyword
// on its schema ($ref schemas are followed).
//
// # Why
//
// Examples are what readers copy from documentation and what mock servers and
// test tools replay. Success responses are the ones every consumer reads, so
// requiring examples there gives most of the benefit at a fraction of the effort
// of requiring them everywhere.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-success-response-example). vacuum's
// oas3-missing-example requires examples on every object instead.
//
// # Options
//
//   - mediaTypes: the media types to check. Default: application/json.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                type: object
//	              example:
//	                name: Rex
//
// # Known false positives / limitations
//
// Examples on individual properties of the schema do not count; the example must
// be on the media type or on the schema itself. Responses that are $ref objects
// are resolved and reported at their use.
func SuccessResponseExample() Rule {
	return New(Spec{
		Name:        "success-response-example",
		Description: "Success response media types should include an example.",
		HowToFix:    "Add an example or examples keyword to the media type, or an example to its schema.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        "mediaTypes",
			Description: "The response media types that must have an example.",
			Default:     []string{successResponseExampleJSON},
		}},
		Check: PerResponse(checkSuccessResponseExample),
	})
}

func checkSuccessResponseExample(call *Call, _ oas.Operation, response oas.Response) []Finding {
	if !oas.IsSuccess(response.Code) {
		return nil
	}

	wanted := seq.Map(slices.Values(call.Strings("mediaTypes")), func(mediaType string) string {
		return strings.ToLower(strings.TrimSpace(mediaType))
	})

	return seq.FlatMapPairs(oas.Pairs(oas.Get(response.Node, "content")), func(name, media *yaml.Node) []Finding {
		return When(
			slices.Contains(wanted, strings.ToLower(name.Value)) && !successResponseExampleHas(call.Root, media),
			func() Finding {
				return Finding{
					Node:    media,
					Path:    oas.JoinAll(response.Path, "content", name.Value),
					Message: "response " + response.Code + " media type " + name.Value + " should include an example",
				}
			},
		)
	})
}

// successResponseExampleHas reports whether the media type or its schema has an
// example.
func successResponseExampleHas(root, media *yaml.Node) bool {
	if oas.Has(media, "example") || oas.Has(media, "examples") {
		return true
	}

	schema := oas.Resolve(root, oas.Get(media, "schema"))

	return oas.Has(schema, "example") || oas.Has(schema, "examples")
}

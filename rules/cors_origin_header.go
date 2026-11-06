package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// CORSOriginHeader returns the extra-cors-origin-header rule.
//
// # What it checks
//
// Every response of every operation must define the Access-Control-Allow-Origin
// header in its headers map. A response without headers, or with headers that
// do not include it, is reported. Header names are compared case-insensitively.
//
// # Why
//
// Browsers only let a page read a cross-origin response when the server returns
// Access-Control-Allow-Origin. Documenting it on every response makes the
// intended CORS policy explicit and reviewable, and lets clients and gateways
// know about it.
//
// # Where it comes from
//
// The OWASP API security ruleset for Spectral (owasp:api8:2023-define-cors-origin).
//
// # Options
//
//   - header: the header every response must define. Default:
//     Access-Control-Allow-Origin.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          headers:
//	            Access-Control-Allow-Origin:
//	              schema:
//	                type: string
//
// # Known false positives / limitations
//
// APIs that are never called from a browser do not need the header, and for
// them the rule is noise; disable it. Responses defined only under
// components.responses are checked where they are referenced from an operation.
func CORSOriginHeader() Rule {
	return New(Spec{
		Name:        "cors-origin-header",
		Description: "Every response must define the Access-Control-Allow-Origin header.",
		HowToFix:    "Add the Access-Control-Allow-Origin header to the headers of the response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        keywordHeader,
			Description: "The header name every response must define.",
			Default:     "Access-Control-Allow-Origin",
		}},
		Check: OnDocument(checkCORSOriginHeader),
	})
}

func checkCORSOriginHeader(call *Call) []Finding {
	header := call.String("header")
	if header == "" {
		return nil
	}

	return PerResponse(func(_ *Call, _ oas.Operation, response oas.Response) []Finding {
		return When(!corsOriginHeaderDefined(response.Node, header), func() Finding {
			return Finding{
				Node:    response.Node,
				Path:    response.Path,
				Message: "response " + response.Code + " does not define the " + header + " header",
			}
		})
	})(call)
}

// corsOriginHeaderDefined reports whether the headers of the response contain
// the wanted header, ignoring case.
func corsOriginHeaderDefined(response *yaml.Node, header string) bool {
	return slices.ContainsFunc(oas.Keys(oas.Get(response, "headers")), func(name string) bool {
		return strings.EqualFold(name, header)
	})
}

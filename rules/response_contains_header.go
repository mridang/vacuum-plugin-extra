package rules

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// ResponseContainsHeader returns the extra-response-contains-header rule.
//
// # What it checks
//
// The responses of every operation must declare the configured headers. The
// headers option maps a response code to the list of header names that the
// response must define in its headers object. A configured key matches a
// response whose code is the same (compared case-insensitively, so default and
// 4XX work), and a range key such as 4XX also matches every response whose code
// starts with that digit, for example 404. Header names are compared
// case-insensitively. Responses defined through a local $ref are resolved.
//
// The rule does nothing when the headers option is empty, which is its default.
//
// Declaring headers such as rate-limit counters or a request id in the
// contract lets clients depend on them.
//
// # Where it comes from
//
// Redocly CLI, rule response-contains-header, which is configured with a map of
// response codes to header names.
//
// # Options
//
//   - headers (map of response code to list of header names, no default): the
//     headers that responses must declare, for example
//     {"200": ["X-Request-Id"], "4XX": ["Retry-After"]}.
//
// # Bad
//
// With headers set to {"200": ["X-Request-Id"]}:
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
//	            X-Request-Id:
//	              schema:
//	                type: string
//
// # Known false positives
//
// A response that is required by one key and satisfied by another (for
// example 404 matched by both 404 and 4XX) must declare the union of the
// headers of every matching key. The Content-Type header is not a response
// header in OpenAPI and cannot be required.
func ResponseContainsHeader() Rule {
	return New(Spec{
		Name:        "response-contains-header",
		Description: "Responses must declare the headers configured for their status code.",
		HowToFix:    "Add the missing header to the headers of the response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{
			{
				Name:        keywordHeaders,
				Description: "A map from response code (or range such as 4XX) to the header names that must be declared.",
			},
		},
		Check: PerResponse(checkResponseContainsHeader),
	})
}

func checkResponseContainsHeader(call *Call, _ oas.Operation, response oas.Response) []Finding {
	return responseContainsHeaderMissing(response, responseContainsHeaderConfig(call.raw("headers")))
}

func responseContainsHeaderMissing(response oas.Response, required map[string][]string) []Finding {
	declared := oas.Get(response.Node, "headers")

	return seq.FlatMapPairs(maps.All(required), func(code string, names []string) []Finding {
		if !responseContainsHeaderMatches(code, response.Code) {
			return nil
		}

		return seq.FlatMapSlice(names, func(name string) []Finding {
			return When(!responseContainsHeaderDeclares(declared, name), func() Finding {
				return Finding{
					Node:    response.Node,
					Path:    response.Path,
					Message: fmt.Sprintf("response %s must declare the %s header", response.Code, name),
				}
			})
		})
	})
}

func responseContainsHeaderMatches(configured, code string) bool {
	return strings.EqualFold(configured, code) || (len(configured) == len("4XX") &&
		strings.HasSuffix(strings.ToUpper(configured), "XX") &&
		strings.HasPrefix(code, configured[:1]))
}

func responseContainsHeaderDeclares(headers *yaml.Node, name string) bool {
	return slices.ContainsFunc(oas.Keys(headers), func(key string) bool {
		return strings.EqualFold(key, name)
	})
}

func responseContainsHeaderConfig(raw any) map[string][]string {
	switch value := raw.(type) {
	case map[string][]string:
		return value
	case map[string]any:
		return responseContainsHeaderNamesByCode(value, func(code string) string { return code })
	case map[any]any:
		return responseContainsHeaderNamesByCode(value, func(code any) string { return fmt.Sprint(code) })
	default:
		return map[string][]string{}
	}
}

// responseContainsHeaderNamesByCode converts the configured entries into the
// header names of every response code.
func responseContainsHeaderNamesByCode[K comparable](entries map[K]any, code func(K) string) map[string][]string {
	return maps.Collect(func(yield func(string, []string) bool) {
		for key, names := range entries {
			if !yield(code(key), responseContainsHeaderNames(names)) {
				return
			}
		}
	})
}

func responseContainsHeaderNames(raw any) []string {
	switch value := raw.(type) {
	case string:
		return []string{value}
	case []string:
		return value
	case []any:
		return stringsFromAny(value)
	default:
		return nil
	}
}

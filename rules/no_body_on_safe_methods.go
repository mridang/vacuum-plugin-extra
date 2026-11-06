package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// Methods that appear more than once in this rule.
const (
	noBodyOnSafeMethodsDelete = "delete"
	noBodyOnSafeMethodsGet    = "get"
)

// NoBodyOnSafeMethods returns the extra-no-body-on-safe-methods rule.
//
// # What it checks
//
// Operations whose HTTP method is in a configurable list must not define a
// requestBody. The default list is delete, get, head and options.
//
// # Why
//
// RFC 9110 gives a request body on GET, HEAD, DELETE and OPTIONS no defined
// semantics. Proxies, caches and client libraries may drop it or reject the
// request, and many HTTP clients cannot send one at all.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-no-operation-requestbody), which has the same
// configurable list of methods.
//
// # Options
//
//   - httpMethods: the methods that must not have a request body, compared
//     case-insensitively. Default: delete, get, head, options.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema:
//	              type: object
//
// # Good
//
//	paths:
//	  /pets:
//	    get:
//	      parameters:
//	        - name: filter
//	          in: query
//	          schema:
//	            type: string
//
// # Known false positives / limitations
//
// OpenAPI 3.1 allows request bodies on every method, and some APIs rely on a
// body for DELETE or GET; remove the method from httpMethods for them.
func NoBodyOnSafeMethods() Rule {
	return New(Spec{
		Name:        "no-body-on-safe-methods",
		Description: "Operations with a method such as GET, HEAD, DELETE or OPTIONS must not define a request body.",
		HowToFix:    "Remove the requestBody, or move its data into parameters.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Options: []Option{{
			Name:        "httpMethods",
			Description: "The HTTP methods that must not define a request body.",
			Default:     []string{noBodyOnSafeMethodsDelete, noBodyOnSafeMethodsGet, "head", "options"},
		}},
		Check: OnDocument(checkNoBodyOnSafeMethods),
	})
}

func checkNoBodyOnSafeMethods(call *Call) []Finding {
	methods := seq.Map(slices.Values(call.Strings("httpMethods")), func(method string) string {
		return strings.ToLower(strings.TrimSpace(method))
	})

	return PerOperation(func(_ *Call, operation oas.Operation) []Finding {
		key, _ := oas.Entry(operation.Node, "requestBody")

		return When(key != nil && slices.Contains(methods, operation.Method), func() Finding {
			return Finding{
				Node:    key,
				Path:    oas.Join(operation.Path, "requestBody"),
				Message: strings.ToUpper(operation.Method) + " operations must not define a request body",
			}
		})
	})(call)
}

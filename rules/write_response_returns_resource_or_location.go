package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// writeResponseReturnsResourceOrLocationIgnoreOption is the name of the option
// listing the status codes that may be empty.
const writeResponseReturnsResourceOrLocationIgnoreOption = "ignoreCodes"

// WriteResponseReturnsResourceOrLocation returns the extra-write-response-returns-resource-or-location rule.
//
// # What it checks
//
// Every 2xx response of a POST, PUT or PATCH operation must tell the caller
// what happened to the resource. It passes when it has a content map (the
// resource, or a description of the result is returned) or, for the codes 201
// and 204, when it declares a Location response header (matched
// case-insensitively) that points at the resource. Any other 2xx response
// without content is reported, as is a 201 or 204 that has neither content nor
// Location. Response objects reached through $ref are resolved.
//
// # Where it comes from
//
// apilint, which asks that writes return the resource or a Location header.
//
// # Options
//
//   - ignoreCodes: status codes that may be empty without a Location header.
//     Default: 202, because an accepted asynchronous request has no resource
//     yet.
//
// # Bad
//
//	paths:
//	  /pets:
//	    post:
//	      responses:
//	        "201":
//	          description: created
//
// # Good
//
//	paths:
//	  /pets:
//	    post:
//	      responses:
//	        "201":
//	          description: created
//	          headers:
//	            Location:
//	              schema:
//	                type: string
//
// # Known false positives
//
// Command style endpoints (POST /pets/{id}/archive) that legitimately return
// 200 or 204 without a body are flagged; list their codes in ignoreCodes or
// return a small result object.
func WriteResponseReturnsResourceOrLocation() Rule {
	return New(Spec{
		Name:        "write-response-returns-resource-or-location",
		Description: "POST, PUT and PATCH 2xx responses must return the resource or a Location header.",
		HowToFix:    "Return the resource in the response content, or declare a Location header for 201 and 204.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{{
			Name:        writeResponseReturnsResourceOrLocationIgnoreOption,
			Description: "Status codes that may be empty without a Location header.",
			Default:     []string{"202"},
		}},
		Check: PerOperation(checkWriteResponseReturnsResourceOrLocation),
	})
}

func checkWriteResponseReturnsResourceOrLocation(call *Call, operation oas.Operation) []Finding {
	if !slices.Contains([]string{literalPost, literalPut, literalPatch}, operation.Method) {
		return nil
	}

	ignored := call.Strings(writeResponseReturnsResourceOrLocationIgnoreOption)

	return seq.FlatMapSlice(oas.Responses(call.Root, operation), func(response oas.Response) []Finding {
		return When(
			oas.IsSuccess(response.Code) && !slices.Contains(ignored, response.Code) &&
				!oas.Has(response.Node, "content") && !writeResponseReturnsResourceOrLocationHasLocation(response),
			func() Finding {
				return Finding{
					Node:    response.Node,
					Path:    response.Path,
					Message: operation.Method + " response " + response.Code + " must return the resource or a Location header",
				}
			},
		)
	})
}

func writeResponseReturnsResourceOrLocationHasLocation(response oas.Response) bool {
	if response.Code != "201" && response.Code != "204" {
		return false
	}

	for name := range oas.Pairs(oas.Get(response.Node, "headers")) {
		if strings.EqualFold(name.Value, "location") {
			return true
		}
	}

	return false
}

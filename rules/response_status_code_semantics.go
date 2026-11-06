package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	responseStatusCodeSemanticsStatusPattern  = `^[1-5][0-9][0-9]$`
	responseStatusCodeSemanticsSuccessPattern = `^2[0-9][0-9]$`
	responseStatusCodeSemanticsCreate         = "create"
	responseStatusCodeSemanticsAccepted       = "202"
	responseStatusCodeSemanticsNoContent      = "204"
	responseStatusCodeSemanticsOK             = "200"
	responseStatusCodeSemanticsCreated        = "201"
	responseStatusCodeSemanticsSwitching      = "101"
	responseStatusCodeSemanticsRedirectMsg    = "A 301, 302, 305 or 307 response should include a response body, " +
		"use a different status code for responses without content"
)

// ResponseStatusCodeSemantics returns the extra-response-status-code-semantics rule.
//
// # What it checks
//
// A port of the IBM function responseStatusCodes. Operations with a responses
// object that has at least one three-digit status code (1xx to 5xx; keys such as
// default or 2XX are not status codes for this rule) are checked as follows:
//
//   - 422 is discouraged, use 400 for invalid request payloads (reported on the
//     422 response).
//   - 302 is discouraged, use 303 or 307 (reported on the 302 response).
//   - At least one success (2xx) code must be present, unless 101 is (reported
//     on responses).
//   - 101 must not be combined with success codes (reported on the 101
//     response).
//   - A 204 response must not have content (reported on its content).
//   - A create operation must return 201 or 202. A create is an operation whose
//     operationId starts with "create" (trimmed, case-insensitive), or a POST on
//     a collection path that has a sibling {path}/{param} path. A 204 is
//     accepted instead only when the create returns 204 and the resource has no
//     body representation: there is no sibling path, or the GET on the sibling
//     path has no 204 response. IBM reports this even when the operation has no
//     success code at all.
//   - PUT must return 200, 201 or 202, or 204 when the GET on the same path
//     itself declares a 204 response (a minimally represented resource).
//   - PATCH must return 200 or 202.
//   - 202 must not be combined with other 2xx codes.
//   - 301, 302, 305 and 307 responses must have content (reported on responses,
//     once per offending code).
//
// Messages are the IBM messages. $ref values on responses are resolved.
//
// # Why
//
// Clients branch on status codes. A 204 with a body, a create answered with 200
// or a redirect without a target description each break an assumption that
// generic clients, caches and gateways make.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-response-status-codes).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /things:
//	    post:
//	      operationId: createThing
//	      responses:
//	        "200":
//	          description: ok
//	        "422":
//	          description: invalid
//
// # Good
//
//	paths:
//	  /things:
//	    post:
//	      operationId: createThing
//	      responses:
//	        "201":
//	          description: created
//	        "400":
//	          description: invalid
//
// # Known false positives / limitations
//
// The rule ports IBM literally, including the exceptions above: the minimal
// representation test only looks for a 204 key on the relevant GET operation.
// Range keys such as 3XX are not examined.
func ResponseStatusCodeSemantics() Rule {
	return New(Spec{
		Name:        "response-status-code-semantics",
		Description: "Operations must use status codes consistently with their meaning.",
		HowToFix:    "Use the status codes recommended by the message: 400 for 422, 303 or 307 for 302, 201 for creates, and so on.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkResponseStatusCodeSemantics),
	})
}

// responseStatusCodeSemanticsOperation bundles what the individual checks need.
type responseStatusCodeSemanticsOperation struct {
	root      *yaml.Node
	operation oas.Operation
	codes     []string
	success   []string
	base      string
}

// checkResponseStatusCodeSemantics applies the IBM status code checks to one operation.
func checkResponseStatusCodeSemantics(call *Call, operation oas.Operation) []Finding {
	responses := oas.Get(operation.Node, "responses")
	if !oas.IsMap(responses) {
		return nil
	}

	statusPattern := regexp.MustCompile(responseStatusCodeSemanticsStatusPattern)
	successPattern := regexp.MustCompile(responseStatusCodeSemanticsSuccessPattern)
	codes := oas.Keys(responses)

	if !slices.ContainsFunc(codes, statusPattern.MatchString) {
		return nil
	}

	subject := &responseStatusCodeSemanticsOperation{
		root:      call.Root,
		operation: operation,
		codes:     codes,
		success:   seq.Filter(slices.Values(codes), successPattern.MatchString),
		base:      oas.Join(operation.Path, "responses"),
	}

	return slices.Concat(
		responseStatusCodeSemanticsCodeChecks(subject),
		responseStatusCodeSemanticsMethodChecks(subject),
	)
}

// responseStatusCodeSemanticsAt builds a finding at a response code.
func responseStatusCodeSemanticsAt(s *responseStatusCodeSemanticsOperation, code, message string) Finding {
	key, _ := oas.Entry(oas.Get(s.operation.Node, "responses"), code)

	return Finding{Node: key, Path: oas.Join(s.base, code), Message: message}
}

// responseStatusCodeSemanticsCodeChecks runs the checks that depend only on the set of codes.
func responseStatusCodeSemanticsCodeChecks(s *responseStatusCodeSemanticsOperation) []Finding {
	has := func(code string) bool { return slices.Contains(s.codes, code) }

	return slices.Concat(
		When(has("422"), func() Finding {
			return responseStatusCodeSemanticsAt(s, "422",
				"Operation responses should use status code 400 instead of 422 for invalid request payloads")
		}),
		When(has("302"), func() Finding {
			return responseStatusCodeSemanticsAt(s, "302",
				"Operation responses should use status code 303 or 307 instead of 302")
		}),
		When(len(s.success) == 0 && !has(responseStatusCodeSemanticsSwitching), func() Finding {
			return Finding{
				Path:    s.base,
				Message: "Operation responses should include at least one success status code (2xx)",
			}
		}),
		When(len(s.success) > 0 && has(responseStatusCodeSemanticsSwitching), func() Finding {
			return responseStatusCodeSemanticsAt(s, responseStatusCodeSemanticsSwitching,
				"Operation responses should not include status code 101 when success status codes (2xx) are present")
		}),
		When(slices.Contains(s.success, responseStatusCodeSemanticsAccepted) && len(s.success) > 1, func() Finding {
			return Finding{
				Path:    s.base,
				Message: "An operation that returns a 202 status code should not return any other 2xx status codes",
			}
		}),
		responseStatusCodeSemanticsContentChecks(s),
	)
}

// responseStatusCodeSemanticsContentChecks checks which responses may or must carry content.
func responseStatusCodeSemanticsContentChecks(s *responseStatusCodeSemanticsOperation) []Finding {
	responses := oas.Get(s.operation.Node, "responses")
	noContent := oas.Resolve(s.root, oas.Get(responses, responseStatusCodeSemanticsNoContent))
	body := oas.Get(noContent, "content")

	return slices.Concat(
		When(body != nil, func() Finding {
			return Finding{
				Node: body,
				Path: oas.JoinAll(s.base, responseStatusCodeSemanticsNoContent, "content"),
				Message: "A 204 response must not include a response body; " +
					"use a different status code for responses with content",
			}
		}),
		seq.FlatMapSlice([]string{"301", "302", "305", "307"}, func(code string) []Finding {
			response := oas.Resolve(s.root, oas.Get(responses, code))

			return When(response != nil && oas.Get(response, "content") == nil, func() Finding {
				return Finding{Path: s.base, Message: responseStatusCodeSemanticsRedirectMsg}
			})
		}),
	)
}

// responseStatusCodeSemanticsMethodChecks checks the codes a create, PUT or PATCH operation must return.
func responseStatusCodeSemanticsMethodChecks(s *responseStatusCodeSemanticsOperation) []Finding {
	anyOf := func(wanted ...string) bool {
		return slices.ContainsFunc(s.success, func(code string) bool { return slices.Contains(wanted, code) })
	}
	noContent := slices.Contains(s.success, responseStatusCodeSemanticsNoContent)
	sibling := resourceResponseConsistencySibling(s.root, s.operation.Route)
	isCreate := strings.HasPrefix(
		strings.ToLower(strings.TrimSpace(oas.StringAt(s.operation.Node, "operationId"))),
		responseStatusCodeSemanticsCreate,
	) || (s.operation.Method == methodPost && sibling != "")

	// A create without a sibling path has no body representation to compare with.
	bodyRepresentation := sibling != "" && !responseStatusCodeSemanticsMinimal(s.root, sibling)

	return slices.Concat(
		When(
			isCreate && !anyOf(responseStatusCodeSemanticsCreated, responseStatusCodeSemanticsAccepted) &&
				(!noContent || bodyRepresentation),
			func() Finding {
				return Finding{
					Path:    s.base,
					Message: "A 201 or 202 status code should be returned by a 'create' operation",
				}
			},
		),
		When(
			s.operation.Method == methodPut && len(s.success) > 0 &&
				!anyOf(
					responseStatusCodeSemanticsOK,
					responseStatusCodeSemanticsCreated,
					responseStatusCodeSemanticsAccepted,
				) &&
				(!noContent || !responseStatusCodeSemanticsMinimal(s.root, s.operation.Route)),
			func() Finding {
				return Finding{
					Path:    s.base,
					Message: "PUT operations should return a 200, 201, or 202 status code",
				}
			},
		),
		When(
			s.operation.Method == methodPatch && len(s.success) > 0 &&
				!anyOf(responseStatusCodeSemanticsOK, responseStatusCodeSemanticsAccepted),
			func() Finding {
				return Finding{Path: s.base, Message: "PATCH operations should return a 200 or 202 status code"}
			},
		),
	)
}

// responseStatusCodeSemanticsMinimal reports whether the GET on the route has a 204 response, which IBM calls a
// minimally represented resource.
func responseStatusCodeSemanticsMinimal(root *yaml.Node, route string) bool {
	item := oas.Resolve(root, oas.Dig(root, "paths", route))

	return oas.Has(oas.Dig(item, "get", "responses"), responseStatusCodeSemanticsNoContent)
}

package rules

import (
	"regexp"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
)

// APIVersionInURL returns the extra-api-version-in-url rule.
//
// # What it checks
//
// The API must carry a version somewhere a client can see it. The document
// passes when at least one server URL, one path template, or one header
// parameter (request) or response header names a version. A version segment is
// a path segment such as v1, v2.1 or V3; a server variable whose name contains
// "version" also counts, as does a header whose name contains "version", such
// as Api-Version or X-API-Version. When none is found a single finding is
// reported on the document root.
//
// # Where it comes from
//
// apilint, which requires a version in a server URL, a URI template matching
// v[0-9]+, or a version request or response header.
//
// # Options
//
// None.
//
// # Bad
//
//	servers:
//	  - url: https://api.example.com
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//
// # Good
//
//	servers:
//	  - url: https://api.example.com/v1
//
// # Known false positives
//
// APIs versioned through media types (application/vnd.example.v2+json) or
// through query parameters other than a header are not recognised, and neither
// are versions that live only in a gateway outside the document.
func APIVersionInURL() Rule {
	return New(Spec{
		Name:        "api-version-in-url",
		Description: "The API must expose a version in a server URL, a path or a version header.",
		HowToFix:    "Add a version segment such as /v1 to a server URL or path, or declare a version header.",
		Severity:    model.SeverityInfo,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkAPIVersionInURL),
	})
}

func apiVersionInURLSegment() *regexp.Regexp {
	return regexp.MustCompile(`(?i)(^|[/.])v[0-9]+(\.[0-9]+)*([/.?#]|$)`)
}

func apiVersionInURLVariable() *regexp.Regexp {
	return regexp.MustCompile(`(?i)\{[^{}]*version[^{}]*\}`)
}

func checkAPIVersionInURL(call *Call) []Finding {
	exposed := apiVersionInURLInServers(call.Root) ||
		apiVersionInURLInPaths(call.Root) ||
		apiVersionInURLInHeaders(call.Root)

	return When(!exposed, func() Finding {
		return Finding{
			Node:    call.Root,
			Path:    "$",
			Message: "the API does not expose a version in a server URL, a path or a version header",
		}
	})
}

func apiVersionInURLMatches(text string) bool {
	return apiVersionInURLSegment().MatchString(text) || apiVersionInURLVariable().MatchString(text)
}

func apiVersionInURLInServers(root *yaml.Node) bool {
	for _, server := range oas.Items(oas.Get(root, "servers")) {
		if apiVersionInURLMatches(oas.StringAt(server, "url")) {
			return true
		}
	}

	return false
}

func apiVersionInURLInPaths(root *yaml.Node) bool {
	for item := range oas.PathItems(root) {
		if apiVersionInURLMatches(item.Route) {
			return true
		}
	}

	return false
}

func apiVersionInURLIsHeaderName(name string) bool {
	return strings.Contains(strings.ToLower(name), "version")
}

func apiVersionInURLInHeaders(root *yaml.Node) bool {
	for operation := range oas.Operations(root) {
		inParameters := slices.ContainsFunc(oas.Parameters(root, operation), func(parameter oas.Parameter) bool {
			return parameter.In() == keywordHeader && apiVersionInURLIsHeaderName(parameter.Name())
		})
		inResponses := slices.ContainsFunc(oas.Responses(root, operation), func(response oas.Response) bool {
			return slices.ContainsFunc(oas.Keys(oas.Get(response.Node, "headers")), apiVersionInURLIsHeaderName)
		})

		if inParameters || inResponses {
			return true
		}
	}

	return false
}

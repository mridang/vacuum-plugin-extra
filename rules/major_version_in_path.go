package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

// MajorVersionInPath returns the extra-major-version-in-path rule.
//
// # What it checks
//
// The API must carry its major version in the URL. Either a server URL has a
// path segment of the form v<n> (for example https://api.example.com/v1), or
// every path of the paths object starts with such a segment (for example
// /v1/pets). When the paths carry the version, all of them must use the same
// version, unless the consistent option is turned off.
//
// # Why
//
// A version in the URL lets a breaking change ship next to the old version, and
// makes the version visible in logs, gateways and browser address bars.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-major-version-in-path).
//
// # Options
//
//   - allowServerVersion: accept a version segment in a server URL in place of
//     versioned paths. Default: true.
//   - consistent: require every versioned path to use the same version.
//     Default: true.
//
// # Bad
//
//	servers:
//	  - url: https://api.example.com
//	paths:
//	  /pets: {}
//
// # Good
//
//	servers:
//	  - url: https://api.example.com/v1
//	paths:
//	  /pets: {}
//
// # Known false positives / limitations
//
// Only the lower-case form v<digits> is recognized; minor versions such as v1.2
// and date based versions are not. APIs that version through a header or a media
// type are reported; disable the rule for them. Server URLs that consist only of
// a variable are not inspected.
func MajorVersionInPath() Rule {
	return New(Spec{
		Name:        "major-version-in-path",
		Description: "The major version must be a v<n> segment in a server URL or at the start of every path.",
		HowToFix:    "Add a version segment such as /v1 to the server URL or to the start of every path.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Formats:     []string{formatOAS3},
		Options: []Option{
			{
				Name:        "allowServerVersion",
				Description: "Accept a version segment in a server URL in place of versioned paths.",
				Default:     true,
			},
			{
				Name:        "consistent",
				Description: "Require every versioned path to use the same version.",
				Default:     true,
			},
		},
		Check: OnDocument(checkMajorVersionInPath),
	})
}

func checkMajorVersionInPath(call *Call) []Finding {
	if call.Bool("allowServerVersion") && majorVersionInPathServerHasVersion(call) {
		return nil
	}

	versions := seq.FlatMap(oas.PathItems(call.Root), func(item oas.PathItem) []string {
		version, ok := majorVersionInPathSegment(item.Route)
		if !ok {
			return nil
		}

		return []string{version}
	})

	return seq.FlatMap(oas.PathItems(call.Root), func(item oas.PathItem) []Finding {
		version, ok := majorVersionInPathSegment(item.Route)
		if !ok {
			return One(Finding{
				Node:    item.KeyNode,
				Path:    item.Path,
				Message: "path '" + item.Route + "' must start with a major version segment such as /v1",
			})
		}

		return When(call.Bool("consistent") && version != versions[0], func() Finding {
			return Finding{
				Node:    item.KeyNode,
				Path:    item.Path,
				Message: "path '" + item.Route + "' uses version " + version + " but other paths use " + versions[0],
			}
		})
	})
}

// majorVersionInPathServerHasVersion reports whether any server URL has a
// version segment in its path.
func majorVersionInPathServerHasVersion(call *Call) bool {
	urls := seq.FlatMapPairs(oas.Items(oas.Get(call.Root, "servers")), func(_ int, server *yaml.Node) []string {
		return []string{oas.StringAt(server, "url")}
	})

	return slices.ContainsFunc(urls, func(url string) bool {
		return slices.ContainsFunc(strings.Split(majorVersionInPathURLPath(url), "/"), majorVersionInPathIsVersion)
	})
}

// majorVersionInPathURLPath returns the path of a server URL, or the URL itself
// when it has no scheme.
func majorVersionInPathURLPath(url string) string {
	_, rest, found := strings.Cut(url, "://")
	if !found {
		return url
	}

	_, path, _ := strings.Cut(rest, "/")

	return "/" + path
}

// majorVersionInPathSegment returns the version at the start of the route.
func majorVersionInPathSegment(route string) (string, bool) {
	first, _, _ := strings.Cut(strings.TrimPrefix(route, "/"), "/")

	return first, majorVersionInPathIsVersion(first)
}

// majorVersionInPathIsVersion reports whether the segment is v followed by
// digits only.
func majorVersionInPathIsVersion(segment string) bool {
	digits, found := strings.CutPrefix(segment, "v")

	return found && digits != "" && !strings.ContainsFunc(digits, func(r rune) bool {
		return r < '0' || r > '9'
	})
}

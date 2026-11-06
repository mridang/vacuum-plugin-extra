package rules

import (
	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	noWildcardMediaTypeWildcard = "*/*"
	noWildcardMediaTypeContent  = "content"
)

// NoWildcardMediaType returns the extra-no-wildcard-media-type rule.
//
// # What it checks
//
// No content map may use the media type */*. The check covers the content of
// parameters, request bodies and responses, whether they are inline or defined
// under components (parameters, requestBodies, responses, headers). One finding
// is reported per wildcard key.
//
// # Why
//
// A wildcard says nothing about the representation of the payload. Clients cannot
// choose a parser, generators cannot pick a type and content negotiation becomes
// guesswork. Declare the concrete media types that the API supports.
//
// # Where it comes from
//
// IBM's API Handbook ruleset (ibm-content-contains-schema family; the wildcard
// restriction of ibm-valid-media-type).
//
// # Options
//
// None.
//
// # Bad
//
//	requestBody:
//	  content:
//	    "*/*":
//	      schema:
//	        type: object
//
// # Good
//
//	requestBody:
//	  content:
//	    application/json:
//	      schema:
//	        type: object
//
// # Known false positives
//
// Only the full wildcard */* is reported. Partial wildcards such as image/* are
// accepted. File download endpoints that return arbitrary content legitimately
// use */* and need to be suppressed.
func NoWildcardMediaType() Rule {
	return New(Spec{
		Name:        "no-wildcard-media-type",
		Description: "Content maps must not use the */* media type.",
		HowToFix:    "Replace */* with the concrete media types that the API supports.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkNoWildcardMediaType),
	})
}

// checkNoWildcardMediaType reports wildcard keys of all content maps.
func checkNoWildcardMediaType(call *Call) []Finding {
	return seq.FlatMap(oas.Walk(call.Root, "$"), func(node oas.Node) []Finding {
		if node.Key != noWildcardMediaTypeContent || !oas.IsMap(node.Node) {
			return nil
		}

		return noWildcardMediaTypeIn(node.Node, node.Path)
	})
}

// noWildcardMediaTypeIn reports the wildcard keys of one content map.
func noWildcardMediaTypeIn(content *yaml.Node, path string) []Finding {
	return seq.FlatMapPairs(oas.Pairs(content), func(key, media *yaml.Node) []Finding {
		return When(key.Value == noWildcardMediaTypeWildcard, func() Finding {
			return Finding{
				Node:    media,
				Path:    oas.Join(path, key.Value),
				Message: "content must not use the wildcard media type */*",
			}
		})
	})
}

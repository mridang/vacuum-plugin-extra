package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	noBinaryInJSONAPIJSONType   = "application/json"
	noBinaryInJSONAPIWhitespace = " \t\r\n\f\v"
	noBinaryInJSONAPIBinaryFmt  = "binary"
	noBinaryInJSONAPISuffix     = " should not contain binary values (type: string, format: binary)"
	noBinaryInJSONAPIMaxDepth   = 32
	noBinaryInJSONAPIAllOf      = "allOf"
)

// NoBinaryInJSONAPI returns the extra-no-binary-in-json-api rule.
//
// # What it checks
//
// A binary string schema (type string with format binary), or a schema whose
// direct items are such a schema, must not be used as:
//
//   - the schema of an operation or path item parameter (parameters[n].schema),
//     or of a parameter's application/json content;
//   - the schema of an application/json request body;
//   - the schema of an application/json response body.
//
// This follows the IBM implementation exactly. A schema is binary when it, or
// any allOf member, is a binary string, or when every oneOf or anyOf member is
// one (IBM schemaHasConstraint). Only the top-level schema and its direct items
// are examined, as in the original. A media type counts as JSON only when it is
// application/json, optionally followed by parameters (for example a charset);
// application/vnd.api+json and other +json types are not JSON for this rule.
// $ref values are resolved for schemas, parameters, request bodies and
// responses, and the finding is reported where the schema is used. Reusable
// objects under components are only checked through the operations using them,
// as in IBM, which only visits usage locations.
//
// Findings use the IBM messages: "Parameters should not contain binary values
// (type: string, format: binary)", "Request bodies with JSON content ..." and
// "Responses with JSON content ...".
//
// # Why
//
// JSON cannot carry raw bytes. A binary schema in a JSON message or in a
// parameter cannot be serialised faithfully, and generators produce broken
// clients for it. Binary content belongs in an octet-stream or multipart body,
// or should be encoded as a base64 string with format byte.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-binary-schemas).
//
// # Options
//
// None.
//
// # Bad
//
//	requestBody:
//	  content:
//	    application/json:
//	      schema:
//	        type: string
//	        format: binary
//
// # Good
//
//	requestBody:
//	  content:
//	    application/octet-stream:
//	      schema:
//	        type: string
//	        format: binary
//
// # Known false positives and limitations
//
// A binary property nested in an object schema is not reported, matching the
// original. Only local $ref values are followed.
func NoBinaryInJSONAPI() Rule {
	return New(Spec{
		Name:        "no-binary-in-json-api",
		Description: "Binary string schemas must not be used for parameters or JSON request and response bodies.",
		HowToFix:    "Use a non-JSON media type for binary data, or encode the value as a base64 string with format byte.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeValidation,
		Check: Concat(
			PerPathItem(checkNoBinaryInJSONAPIPathItem),
			PerOperation(checkNoBinaryInJSONAPIOperation),
		),
	})
}

// checkNoBinaryInJSONAPIPathItem inspects the parameters shared by the operations of a path item.
func checkNoBinaryInJSONAPIPathItem(call *Call, item oas.PathItem) []Finding {
	return noBinaryInJSONAPIParameters(
		call.Root,
		oas.Get(item.Node, "parameters"),
		oas.Join(item.Path, "parameters"),
	)
}

// checkNoBinaryInJSONAPIOperation inspects the parameters, request body and responses of an operation.
func checkNoBinaryInJSONAPIOperation(call *Call, operation oas.Operation) []Finding {
	return slices.Concat(
		noBinaryInJSONAPIParameters(
			call.Root,
			oas.Get(operation.Node, "parameters"),
			oas.Join(operation.Path, "parameters"),
		),
		noBinaryInJSONAPIBody(
			call.Root,
			oas.Resolve(call.Root, oas.Get(operation.Node, "requestBody")),
			oas.Join(operation.Path, "requestBody"),
			"Request bodies with JSON content",
		),
		seq.FlatMapPairs(oas.Pairs(oas.Get(operation.Node, "responses")), func(code, response *yaml.Node) []Finding {
			path := oas.JoinAll(operation.Path, "responses", code.Value)

			return noBinaryInJSONAPIBody(
				call.Root,
				oas.Resolve(call.Root, response),
				path,
				"Responses with JSON content",
			)
		}),
	)
}

// noBinaryInJSONAPIReport returns a finding when the schema is binary.
func noBinaryInJSONAPIReport(root, schema *yaml.Node, path string, what string) []Finding {
	return When(noBinaryInJSONAPIBinary(root, schema), func() Finding {
		return Finding{Node: schema, Path: path, Message: what + noBinaryInJSONAPISuffix}
	})
}

// noBinaryInJSONAPIParameters checks every parameter of a list.
func noBinaryInJSONAPIParameters(root, list *yaml.Node, path string) []Finding {
	return seq.FlatMapPairs(oas.Items(list), func(i int, item *yaml.Node) []Finding {
		parameter := oas.Resolve(root, item)
		parameterPath := oas.Index(path, i)

		return slices.Concat(
			noBinaryInJSONAPIReport(
				root,
				oas.Get(parameter, "schema"),
				oas.Join(parameterPath, "schema"),
				"Parameters",
			),
			seq.FlatMapSlice(
				oas.MediaTypes(oas.Get(parameter, "content"), oas.Join(parameterPath, "content")),
				func(media oas.MediaType) []Finding {
					if !noBinaryInJSONAPIJSON(media.Name) {
						return nil
					}

					return noBinaryInJSONAPIReport(
						root,
						oas.Get(media.Node, "schema"),
						oas.Join(media.Path, "schema"),
						"Parameters",
					)
				},
			),
		)
	})
}

// noBinaryInJSONAPIBody checks the JSON media types of a request body or response.
func noBinaryInJSONAPIBody(root, owner *yaml.Node, path string, what string) []Finding {
	return seq.FlatMapSlice(
		oas.MediaTypes(oas.Get(owner, "content"), oas.Join(path, "content")),
		func(media oas.MediaType) []Finding {
			if !noBinaryInJSONAPIJSON(media.Name) {
				return nil
			}

			return noBinaryInJSONAPIReport(root, oas.Get(media.Node, "schema"), oas.Join(media.Path, "schema"), what)
		},
	)
}

// noBinaryInJSONAPIJSON reports whether the media type is application/json with optional parameters.
func noBinaryInJSONAPIJSON(mediaType string) bool {
	base, _, _ := strings.Cut(Fold(mediaType), ";")

	return strings.TrimRight(base, noBinaryInJSONAPIWhitespace) == noBinaryInJSONAPIJSONType
}

// noBinaryInJSONAPIBinary reports whether the schema, or its direct items, is a binary string.
func noBinaryInJSONAPIBinary(root, schema *yaml.Node) bool {
	resolved := oas.Resolve(root, schema)
	if noBinaryInJSONAPIHasConstraint(root, resolved, noBinaryInJSONAPIString, 0) {
		return true
	}

	items := oas.Resolve(root, oas.Get(resolved, "items"))

	return noBinaryInJSONAPIHasConstraint(root, items, noBinaryInJSONAPIString, 0)
}

// noBinaryInJSONAPIString reports whether the schema is a string with format binary.
func noBinaryInJSONAPIString(schema *yaml.Node) bool {
	return oas.HasType(schema, "string") && oas.StringAt(schema, "format") == noBinaryInJSONAPIBinaryFmt
}

// noBinaryInJSONAPIHasConstraint mirrors IBM schemaHasConstraint: the predicate
// holds for the schema, for any allOf member, or for every oneOf or anyOf member.
func noBinaryInJSONAPIHasConstraint(root, node *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	schema := oas.Resolve(root, node)
	if !oas.IsMap(schema) || depth > noBinaryInJSONAPIMaxDepth {
		return false
	}

	if predicate(schema) {
		return true
	}

	for _, member := range oas.Items(oas.Get(schema, noBinaryInJSONAPIAllOf)) {
		if noBinaryInJSONAPIHasConstraint(root, member, predicate, depth+1) {
			return true
		}
	}

	for _, key := range []string{literalOneof, keywordAnyOf} {
		if noBinaryInJSONAPIAll(root, oas.Get(schema, key), predicate, depth) {
			return true
		}
	}

	return false
}

// noBinaryInJSONAPIAll reports whether a non-empty list has the constraint in every member.
func noBinaryInJSONAPIAll(root, list *yaml.Node, predicate func(*yaml.Node) bool, depth int) bool {
	return oas.IsSeq(list) && len(list.Content) > 0 && !slices.ContainsFunc(list.Content, func(member *yaml.Node) bool {
		return !noBinaryInJSONAPIHasConstraint(root, member, predicate, depth+1)
	})
}

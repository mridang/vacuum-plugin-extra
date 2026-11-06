package rules

import (
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const putBodyMatchesGetResponseDescription = "description"

const (
	putBodyMatchesGetResponseCode  = "responseCode"
	putBodyMatchesGetResponseDepth = 16
)

// PutBodyMatchesGetResponse returns the extra-put-body-matches-get-response rule.
//
// # What it checks
//
// When a path item has both a get and a put operation, the schema of each put
// request body media type must be equal to the schema of the same media type in
// the get success response. The default response is the 200 response. Schemas
// are compared structurally after local $refs are followed, so a $ref and the
// inlined schema it points to are equal; description, title and example
// keywords are ignored. A put media type that the get response does not offer
// is not compared, and neither is a path without both operations.
//
// # Why
//
// PUT replaces the representation that GET returns. When the two schemas drift
// apart, clients cannot read a resource, change a field and write it back, and
// the API stops being round-trippable.
//
// # Where it comes from
//
// openapilint (operation-payload-put), which enforces that the PUT payload
// matches the GET 200 response.
//
// # Options
//
//   - responseCode (string, default "200"): the get response whose schema the
//     put body must match.
//
// # Bad
//
//	paths:
//	  /pets/{id}:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema: {$ref: "#/components/schemas/Pet"}
//	    put:
//	      requestBody:
//	        content:
//	          application/json:
//	            schema: {$ref: "#/components/schemas/NewPet"}
//
// # Good
//
//	schema: {$ref: "#/components/schemas/Pet"}
//
// # Known false positives and limitations
//
// Resources whose read model contains server-generated fields (ids, timestamps)
// legitimately differ from the write model; configure a different
// response code or suppress the rule for them. Equality is exact: reordering of
// allOf members or required entries counts as a difference.
func PutBodyMatchesGetResponse() Rule {
	return New(Spec{
		Name:        "put-body-matches-get-response",
		Description: "The PUT request body schema must equal the GET success response schema of the same path.",
		HowToFix:    "Use the same schema for the PUT request body and the GET response.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Options: []Option{
			{
				Name:        putBodyMatchesGetResponseCode,
				Description: "The get response code whose schema the put body must match.",
				Default:     "200",
			},
		},
		Check: PerPathItem(checkPutBodyMatchesGetResponse),
	})
}

// checkPutBodyMatchesGetResponse compares put bodies with get responses.
func checkPutBodyMatchesGetResponse(call *Call, item oas.PathItem) []Finding {
	if !oas.IsMap(oas.Get(item.Node, "put")) || !oas.IsMap(oas.Get(item.Node, "get")) {
		return nil
	}

	return putBodyMatchesGetResponsePair(call.Root, item, call.String(putBodyMatchesGetResponseCode))
}

// putBodyMatchesGetResponsePair compares the shared media types of one put and get pair.
func putBodyMatchesGetResponsePair(root *yaml.Node, item oas.PathItem, code string) []Finding {
	body := oas.Resolve(root, oas.Dig(item.Node, "put", "requestBody"))
	response := oas.Resolve(root, oas.Dig(item.Node, "get", "responses", code))
	responseContent := oas.Get(response, "content")
	path := oas.JoinAll(item.Path, "put", "requestBody", "content")

	return seq.FlatMapSlice(oas.MediaTypes(oas.Get(body, "content"), path), func(media oas.MediaType) []Finding {
		putSchema := oas.Get(media.Node, "schema")
		getSchema := oas.Dig(responseContent, media.Name, "schema")

		return When(
			putSchema != nil && getSchema != nil &&
				putBodyMatchesGetResponseCanon(root, putSchema, nil, 0) !=
					putBodyMatchesGetResponseCanon(root, getSchema, nil, 0),
			func() Finding {
				return Finding{
					Node:    putSchema,
					Path:    oas.Join(media.Path, "schema"),
					Message: "the put request body schema for '" + media.Name + "' must equal the get " + code + " response schema",
				}
			},
		)
	})
}

// putBodyMatchesGetResponseIgnored lists keywords that do not change what a schema accepts.
func putBodyMatchesGetResponseIgnored() []string {
	return []string{putBodyMatchesGetResponseDescription, literalTitle, "example", "examples", "externalDocs"}
}

// putBodyMatchesGetResponseCanon renders a node in a canonical form with local references followed.
func putBodyMatchesGetResponseCanon(root, node *yaml.Node, seen []string, depth int) string {
	switch {
	case node == nil:
		return ""
	case depth > putBodyMatchesGetResponseDepth:
		return "..."
	case oas.IsMap(node):
		return putBodyMatchesGetResponseCanonMap(root, node, seen, depth)
	case oas.IsSeq(node):
		parts := seq.Map(slices.Values(node.Content), func(item *yaml.Node) string {
			return putBodyMatchesGetResponseCanon(root, item, seen, depth+1)
		})

		return "[" + strings.Join(parts, ",") + "]"
	default:
		return "'" + node.Value + "'"
	}
}

// putBodyMatchesGetResponseCanonMap renders a mapping with sorted keys, following a $ref.
func putBodyMatchesGetResponseCanonMap(root, node *yaml.Node, seen []string, depth int) string {
	if ref := oas.Ref(node); ref != "" {
		target := oas.Pointer(root, ref)
		if target == nil || slices.Contains(seen, ref) {
			return "$ref(" + ref + ")"
		}

		return putBodyMatchesGetResponseCanon(root, target, slices.Concat(seen, []string{ref}), depth+1)
	}

	parts := seq.FlatMapPairs(oas.Pairs(node), func(key, value *yaml.Node) []string {
		if slices.Contains(putBodyMatchesGetResponseIgnored(), key.Value) {
			return nil
		}

		return []string{key.Value + ":" + putBodyMatchesGetResponseCanon(root, value, seen, depth+1)}
	})

	return "{" + strings.Join(slices.Sorted(slices.Values(parts)), ",") + "}"
}

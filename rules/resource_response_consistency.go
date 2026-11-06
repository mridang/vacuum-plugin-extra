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
	resourceResponseConsistencyMaxDepth   = 24
	resourceResponseConsistencyCollection = "Collection"
	resourceResponseConsistencyOK         = "200"
	resourceResponseConsistencyCreated    = "201"
	resourceResponseConsistencyCodePat    = `^2[0-9][0-9]$`
	resourceResponseConsistencyWhitespace = " \t\r\n\f\v"
	resourceResponseConsistencyInstance   = "Operations on a single resource instance should return the " +
		"resource canonical schema"
	resourceResponseConsistencyBulk = "Bulk resource operations should return the resource collection schema"
)

// ResourceResponseConsistency returns the extra-resource-response-consistency rule.
//
// # What it checks
//
// A port of the IBM function resourceResponseConsistency. It examines POST,
// PUT and PATCH operations. A POST is only examined when it is a create
// operation: its operationId (trimmed, case-insensitive) starts with "create",
// or a sibling path of the form {path}/{param} exists (POST /things next to
// /things/{id}).
//
// The expected ("canonical") schema is the JSON schema of the GET operation on
// the resource: for a POST the GET of the sibling instance path, for PUT and
// PATCH the GET of the same path. That schema is the one of the 200 response
// when the GET has one, otherwise of the first 2xx response with a JSON schema,
// taken from the first application/json media type (optionally with
// parameters). Only the media types that are exactly application/json count.
//
// Every JSON media type of the 200 and 201 responses of the examined operation
// that defines a schema must have a schema deeply equal to the canonical one
// after $ref resolution (key order is irrelevant, array order is not). The
// finding is reported on the media type object, for example
// paths['/things'].post.responses['201'].content['application/json'].
//
// A PUT on a path whose GET returns a $ref schema named *Collection is a bulk
// operation and gets the message "Bulk resource operations should return the
// resource collection schema: Name"; everything else gets "Operations on a
// single resource instance should return the resource canonical schema" with
// ": Name" appended when the GET schema is a $ref to a named schema. When the
// GET has no JSON schema, the operation has no 2xx or 200/201 JSON response,
// or there is no sibling path for a POST, nothing is reported.
//
// # Why
//
// A client that creates or updates a resource expects to receive the
// representation it would get when reading it. Diverging schemas force clients
// to model the same resource twice and are a common source of generator bugs.
//
// # Where it comes from
//
// IBM openapi-validator (ibm-resource-response-consistency).
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
//	      responses:
//	        "201":
//	          content:
//	            application/json:
//	              schema: {$ref: "#/components/schemas/ThingSummary"}
//	  /things/{id}:
//	    get:
//	      responses:
//	        "200":
//	          content:
//	            application/json:
//	              schema: {$ref: "#/components/schemas/Thing"}
//
// # Good
//
//	paths:
//	  /things:
//	    post:
//	      responses:
//	        "201":
//	          content:
//	            application/json:
//	              schema: {$ref: "#/components/schemas/Thing"}
//	  /things/{id}:
//	    get:
//	      responses:
//	        "200":
//	          content:
//	            application/json:
//	              schema: {$ref: "#/components/schemas/Thing"}
//
// # Known false positives / limitations
//
// The comparison is structural and exact, as in IBM: two schemas that describe
// the same data in different ways are reported. Only local $ref values are
// resolved, and the schema name in the message is the last segment of the $ref
// held directly by the GET media type.
func ResourceResponseConsistency() Rule {
	return New(Spec{
		Name:        "resource-response-consistency",
		Description: "Create and update operations must return the same schema as the GET of the resource.",
		HowToFix:    "Return the schema that the GET operation of the resource returns.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       PerOperation(checkResourceResponseConsistency),
	})
}

// checkResourceResponseConsistency compares the POST, PUT and PATCH responses of one operation with the GET schema.
func checkResourceResponseConsistency(call *Call, operation oas.Operation) []Finding {
	canonical := resourceResponseConsistencyCanonical(call.Root, operation)
	if canonical == nil {
		return nil
	}

	return resourceResponseConsistencyCompare(call.Root, operation, canonical)
}

// resourceResponseConsistencyTarget is the GET schema an operation must return.
type resourceResponseConsistencyTarget struct {
	schema *yaml.Node
	name   string
	bulk   bool
}

// resourceResponseConsistencyCanonical finds the GET schema the operation is compared with, or nil.
func resourceResponseConsistencyCanonical(root *yaml.Node, operation oas.Operation) *resourceResponseConsistencyTarget {
	route, ok := resourceResponseConsistencyGetRoute(root, operation)
	if !ok {
		return nil
	}

	schema, name := resourceResponseConsistencyGetSchema(root, route)
	if schema == nil {
		return nil
	}

	bulk := operation.Method == methodPut && !strings.HasSuffix(operation.Route, "}") &&
		strings.HasSuffix(name, resourceResponseConsistencyCollection)

	return &resourceResponseConsistencyTarget{schema: schema, name: name, bulk: bulk}
}

// resourceResponseConsistencyGetRoute returns the route whose GET the operation is compared with.
func resourceResponseConsistencyGetRoute(root *yaml.Node, operation oas.Operation) (string, bool) {
	switch operation.Method {
	case methodPost:
		sibling := resourceResponseConsistencySibling(root, operation.Route)

		return sibling, sibling != ""
	case methodPut, methodPatch:
		return operation.Route, true
	default:
		return "", false
	}
}

// resourceResponseConsistencySibling returns the path of the form route/{name} when route is a collection path.
func resourceResponseConsistencySibling(root *yaml.Node, route string) string {
	if strings.HasSuffix(strings.TrimSpace(route), "}") {
		return ""
	}

	for item := range oas.PathItems(root) {
		rest, found := strings.CutPrefix(item.Route, route+"/{")
		if !found || !strings.HasSuffix(rest, "}") {
			continue
		}

		name := rest[:len(rest)-1]
		if name != "" && !strings.ContainsAny(name, "{}/") {
			return item.Route
		}
	}

	return ""
}

// resourceResponseConsistencyGetSchema returns the JSON schema of the success response of the GET on route.
func resourceResponseConsistencyGetSchema(root *yaml.Node, route string) (*yaml.Node, string) {
	item := oas.Resolve(root, oas.Dig(root, "paths", route))
	responses := oas.Dig(item, "get", "responses")

	codes := resourceResponseConsistencySuccessCodes(responses)
	if len(codes) == 0 {
		return nil, ""
	}

	code, found := resourceResponseConsistencyPickCode(root, responses, codes)
	if !found {
		return nil, ""
	}

	schema := resourceResponseConsistencyJSONSchema(root, oas.Get(responses, code))
	if schema == nil {
		return nil, ""
	}

	return oas.Resolve(root, schema), oas.RefName(oas.Ref(schema))
}

// resourceResponseConsistencyPickCode returns the 200 code when present, and otherwise the first
// success code that has a JSON schema.
func resourceResponseConsistencyPickCode(root, responses *yaml.Node, codes []string) (string, bool) {
	if slices.Contains(codes, resourceResponseConsistencyOK) {
		return resourceResponseConsistencyOK, true
	}

	index := slices.IndexFunc(codes, func(candidate string) bool {
		return resourceResponseConsistencyJSONSchema(root, oas.Get(responses, candidate)) != nil
	})
	if index < 0 {
		return "", false
	}

	return codes[index], true
}

// resourceResponseConsistencySuccessCodes returns the 2xx status code keys of a responses object.
func resourceResponseConsistencySuccessCodes(responses *yaml.Node) []string {
	pattern := regexp.MustCompile(resourceResponseConsistencyCodePat)

	return seq.FlatMapPairs(oas.Pairs(responses), func(key, _ *yaml.Node) []string {
		if !pattern.MatchString(key.Value) {
			return nil
		}

		return []string{key.Value}
	})
}

// resourceResponseConsistencyJSONSchema returns the schema of the first application/json media type.
func resourceResponseConsistencyJSONSchema(root, response *yaml.Node) *yaml.Node {
	content := oas.Get(oas.Resolve(root, response), "content")

	for name, media := range oas.Pairs(content) {
		if !resourceResponseConsistencyJSON(name.Value) {
			continue
		}

		schema := oas.Get(oas.Resolve(root, media), "schema")
		if schema == nil {
			return nil
		}

		return schema
	}

	return nil
}

// resourceResponseConsistencyJSON reports whether the media type is application/json with optional parameters.
func resourceResponseConsistencyJSON(mediaType string) bool {
	base, _, _ := strings.Cut(strings.ToLower(mediaType), ";")

	return strings.TrimRight(base, resourceResponseConsistencyWhitespace) == "application/json"
}

// resourceResponseConsistencyCompare checks the 200 and 201 responses against the canonical schema.
func resourceResponseConsistencyCompare(
	root *yaml.Node,
	operation oas.Operation,
	target *resourceResponseConsistencyTarget,
) []Finding {
	want := resourceResponseConsistencyCanon(root, target.schema, 0, nil)
	message := resourceResponseConsistencyMessage(target)
	responses := oas.Get(operation.Node, "responses")
	codes := resourceResponseConsistencySuccessCodes(responses)

	return seq.FlatMapSlice(
		[]string{resourceResponseConsistencyOK, resourceResponseConsistencyCreated},
		func(code string) []Finding {
			if !slices.Contains(codes, code) {
				return nil
			}

			response := oas.Resolve(root, oas.Get(responses, code))
			base := oas.JoinAll(operation.Path, "responses", code, "content")

			return seq.FlatMapPairs(oas.Pairs(oas.Get(response, "content")), func(name, media *yaml.Node) []Finding {
				schema := oas.Get(oas.Resolve(root, media), "schema")

				return When(
					schema != nil && resourceResponseConsistencyJSON(name.Value) &&
						resourceResponseConsistencyCanon(root, schema, 0, nil) != want,
					func() Finding {
						return Finding{Node: media, Path: oas.Join(base, name.Value), Message: message}
					},
				)
			})
		},
	)
}

// resourceResponseConsistencyMessage builds the finding message for the target.
func resourceResponseConsistencyMessage(target *resourceResponseConsistencyTarget) string {
	if target.bulk {
		return resourceResponseConsistencyNamed(resourceResponseConsistencyBulk, target.name)
	}

	return resourceResponseConsistencyNamed(resourceResponseConsistencyInstance, target.name)
}

// resourceResponseConsistencyNamed appends the schema name to the message when there is one.
func resourceResponseConsistencyNamed(message, name string) string {
	if name == "" {
		return message
	}

	return message + ": " + name
}

// resourceResponseConsistencyCanon returns a canonical string of the schema in
// which keys are sorted and local references are resolved. The ancestors guard
// against reference cycles.
func resourceResponseConsistencyCanon(root, node *yaml.Node, depth int, ancestors []*yaml.Node) string {
	ref := oas.Ref(node)
	resolved := oas.Resolve(root, node)

	if resolved == nil || depth > resourceResponseConsistencyMaxDepth || slices.Contains(ancestors, resolved) {
		return "~" + ref
	}

	path := slices.Concat(ancestors, []*yaml.Node{resolved})

	switch {
	case oas.IsMap(resolved):
		parts := seq.Map(slices.Values(slices.Sorted(slices.Values(oas.Keys(resolved)))), func(key string) string {
			return key + ":" + resourceResponseConsistencyCanon(root, oas.Get(resolved, key), depth+1, path)
		})

		return "{" + strings.Join(parts, ",") + "}"
	case oas.IsSeq(resolved):
		parts := seq.Map(slices.Values(resolved.Content), func(item *yaml.Node) string {
			return resourceResponseConsistencyCanon(root, item, depth+1, path)
		})

		return "[" + strings.Join(parts, ",") + "]"
	default:
		return resolved.Tag + resolved.Value
	}
}

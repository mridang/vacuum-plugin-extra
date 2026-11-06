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
	resourceSchemaNamingConventionAllOf    = "allOf"
	resourceSchemaNamingConventionAnyOf    = "anyOf"
	resourceSchemaNamingConventionOneOf    = "oneOf"
	resourceSchemaNamingConventionMaxDepth = 16
	resourceSchemaNamingConventionGet      = "get"
	resourceSchemaNamingConventionPost     = "post"
	resourceSchemaNamingConventionPut      = "put"
	resourceSchemaNamingConventionPatch    = "patch"
	resourceSchemaNamingConventionOK       = "200"
	resourceSchemaNamingConventionContent  = "content"
)

// ResourceSchemaNamingConvention returns the extra-resource-schema-naming-convention rule.
//
// # What it checks
//
// A port of IBM's ibm-schema-naming-convention. For every pair of
// resource-oriented paths, a collection path (one that does not end in a path
// variable, such as /pets) and its first sibling instance path
// (/pets/{pet_id}: the collection path followed by one {variable} segment), the
// schema names must derive from the canonical schema. The canonical schema is
// the JSON schema of the success response of GET on the instance path (code 200
// when present, else the first 2xx code with a JSON schema; the first
// application/json media type, parameters allowed). Its name is the component
// name the schema references; an inline schema has no name and the checks that
// need it are skipped. For a canonical name Pet:
//
//   - the success response schema of GET on the collection path must be named
//     PetCollection ("Collection schema for path '/pets' should use the name:
//     PetCollection");
//   - when that schema has a property named after the last segment of the
//     collection path (found directly, in allOf, or in every oneOf and anyOf
//     member), the schema of its items must be named Pet or PetSummary;
//   - the application/json request body schema of POST on the collection path
//     and of PUT on the instance path must be named PetPrototype or Pet;
//   - the application/json-patch+json or application/merge-patch+json request
//     body schema of PATCH on the instance path must be named PetPatch.
//
// Findings sit on the schema of the media type, or on the items of the list
// property, and use IBM's messages.
//
// # Why
//
// Predictable names let consumers and generated SDKs find the schema that
// belongs to an operation: a client can derive PetCollection, PetPrototype and
// PetPatch from Pet without reading the document.
//
// # Where it comes from
//
// IBM's openapi-validator (ibm-schema-naming-convention, severity warn).
//
// # Options
//
// None.
//
// # Bad
//
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                $ref: "#/components/schemas/PetList"
//	  /pets/{pet_id}:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              schema:
//	                $ref: "#/components/schemas/Pet"
//
// # Good
//
// The same document with the collection response referencing
// "#/components/schemas/PetCollection".
//
// # Known false positives and limitations
//
// As in the original, the relation between the canonical schema name and the
// path segments (pluralisation) and the names of identity or reference schemas
// are not checked. A schema name is read from the $ref found at the schema
// position; chains of references are not followed.
func ResourceSchemaNamingConvention() Rule {
	return New(Spec{
		Name:        "resource-schema-naming-convention",
		Description: "Schema names of resource paths must derive from the canonical schema name.",
		HowToFix:    "Rename the schemas to <Canonical>Collection, <Canonical>Prototype and <Canonical>Patch.",
		Severity:    model.SeverityWarn,
		Given:       "$",
		Type:        TypeStyle,
		Check:       OnDocument(checkResourceSchemaNamingConvention),
	})
}

// resourceSchemaNamingConventionChecker holds the immutable inputs of one run.
type resourceSchemaNamingConventionChecker struct {
	root  *yaml.Node
	items []oas.PathItem
}

// resourceSchemaNamingConventionSchema is a located schema reference.
type resourceSchemaNamingConventionSchema struct {
	node *yaml.Node
	path string
	name string
}

// resourceSchemaNamingConventionProperty is a located property schema.
type resourceSchemaNamingConventionProperty struct {
	node *yaml.Node
	path string
}

// checkResourceSchemaNamingConvention checks every collection and instance path pair.
func checkResourceSchemaNamingConvention(call *Call) []Finding {
	checker := &resourceSchemaNamingConventionChecker{
		root:  call.Root,
		items: slices.Collect(oas.PathItems(call.Root)),
	}

	return seq.FlatMapSlice(checker.items, func(item oas.PathItem) []Finding {
		if strings.HasSuffix(strings.TrimSpace(item.Route), "}") {
			return nil
		}

		sibling, ok := checker.sibling(item.Route)
		if !ok {
			return nil
		}

		return checker.pair(item, sibling)
	})
}

// sibling finds the first instance path /route/{variable} of a collection route.
func (c *resourceSchemaNamingConventionChecker) sibling(route string) (oas.PathItem, bool) {
	pattern, err := regexp.Compile("^" + regexp.QuoteMeta(route) + `/\{[^{}/]+\}$`)
	if err != nil {
		return oas.PathItem{}, false
	}

	index := slices.IndexFunc(c.items, func(item oas.PathItem) bool {
		return pattern.MatchString(item.Route)
	})
	if index < 0 {
		return oas.PathItem{}, false
	}

	return c.items[index], true
}

// pair applies the conventions to one collection and instance pair.
func (c *resourceSchemaNamingConventionChecker) pair(collection, instance oas.PathItem) []Finding {
	canonical := c.responseSchema(instance, resourceSchemaNamingConventionGet)
	if canonical.name == "" {
		return nil
	}

	return slices.Concat(
		c.collection(collection, canonical.name),
		c.prototype(collection, resourceSchemaNamingConventionPost, canonical.name),
		c.prototype(instance, resourceSchemaNamingConventionPut, canonical.name),
		c.patch(instance, canonical.name),
	)
}

// collection checks the collection response and its list property.
func (c *resourceSchemaNamingConventionChecker) collection(item oas.PathItem, canonical string) []Finding {
	schema := c.responseSchema(item, resourceSchemaNamingConventionGet)
	if schema.node == nil {
		return nil
	}

	return slices.Concat(
		When(schema.name != "" && schema.name != canonical+"Collection", func() Finding {
			return resourceSchemaNamingConventionFinding(
				schema,
				"Collection schema for path '"+item.Route+"' should use the name: "+canonical+"Collection",
			)
		}),
		c.listItems(schema, resourceSchemaNamingConventionLastSegment(item.Route), canonical),
	)
}

// listItems checks the items schema of the list property of a collection response.
func (c *resourceSchemaNamingConventionChecker) listItems(
	schema resourceSchemaNamingConventionSchema,
	listName, canonical string,
) []Finding {
	resolved := oas.Resolve(c.root, schema.node)
	if !c.hasProperty(resolved, listName, 0) {
		return nil
	}

	property := c.property(resolved, c.basePath(schema), listName, 0)
	if property.node == nil {
		return nil
	}

	items := c.named(oas.Get(property.node, "items"), oas.Join(property.path, "items"))

	return When(
		items.name != "" && items.name != canonical && items.name != canonical+"Summary",
		func() Finding {
			return resourceSchemaNamingConventionFinding(
				items,
				"Items schema for collection resource list property '"+listName+
					"' should use the name: "+canonical+" or "+canonical+"Summary",
			)
		},
	)
}

// prototype checks the request body schema name of a POST or PUT operation.
func (c *resourceSchemaNamingConventionChecker) prototype(item oas.PathItem, method, canonical string) []Finding {
	schema := c.requestSchema(item, method)

	return When(
		schema.name != "" && schema.name != canonical+"Prototype" && schema.name != canonical,
		func() Finding {
			return resourceSchemaNamingConventionFinding(
				schema,
				"Prototype schema ("+strings.ToUpper(method)+" request body) for path '"+item.Route+
					"' should use the name: "+canonical+"Prototype",
			)
		},
	)
}

// patch checks the request body schema name of a PATCH operation.
func (c *resourceSchemaNamingConventionChecker) patch(item oas.PathItem, canonical string) []Finding {
	schema := c.requestSchema(item, resourceSchemaNamingConventionPatch)

	return When(schema.name != "" && schema.name != canonical+"Patch", func() Finding {
		return resourceSchemaNamingConventionFinding(
			schema,
			"Patch schema for path '"+item.Route+"' should use the name: "+canonical+"Patch",
		)
	})
}

// resourceSchemaNamingConventionMime matches a media type, ignoring parameters and case.
func resourceSchemaNamingConventionMime(mime, subtype string) bool {
	return regexp.MustCompile(`(?i)^application/` + regexp.QuoteMeta(subtype) + `(\s*;.*)?$`).MatchString(mime)
}

// resourceSchemaNamingConventionRequestMedia reports whether a media type carries the request body of a method.
func resourceSchemaNamingConventionRequestMedia(method, name string) bool {
	if method == resourceSchemaNamingConventionPatch {
		return resourceSchemaNamingConventionMime(name, "json-patch+json") ||
			resourceSchemaNamingConventionMime(name, "merge-patch+json")
	}

	return resourceSchemaNamingConventionMime(name, "json")
}

// requestSchema returns the schema of the JSON (or JSON patch) request body.
func (c *resourceSchemaNamingConventionChecker) requestSchema(
	item oas.PathItem,
	method string,
) resourceSchemaNamingConventionSchema {
	operation := oas.Get(item.Node, method)
	if operation == nil {
		return resourceSchemaNamingConventionSchema{}
	}

	body := oas.Resolve(c.root, oas.Get(operation, "requestBody"))
	path := oas.JoinAll(item.Path, method, "requestBody", resourceSchemaNamingConventionContent)
	medias := oas.MediaTypes(oas.Get(body, resourceSchemaNamingConventionContent), path)

	index := slices.IndexFunc(medias, func(media oas.MediaType) bool {
		return resourceSchemaNamingConventionRequestMedia(method, media.Name)
	})
	if index < 0 {
		return resourceSchemaNamingConventionSchema{}
	}

	schema := oas.Get(medias[index].Node, "schema")
	if !oas.IsMap(schema) {
		return resourceSchemaNamingConventionSchema{}
	}

	return c.named(schema, oas.Join(medias[index].Path, "schema"))
}

// responseSchema returns the JSON schema of the success response of an operation:
// code 200 when present, else the first 2xx code that has a JSON schema.
func (c *resourceSchemaNamingConventionChecker) responseSchema(
	item oas.PathItem,
	method string,
) resourceSchemaNamingConventionSchema {
	responses := oas.Get(oas.Get(item.Node, method), "responses")
	code := c.successCode(responses)
	if code == "" {
		return resourceSchemaNamingConventionSchema{}
	}

	name, media := c.jsonMediaNamed(oas.Resolve(c.root, oas.Get(responses, code)))
	if media == nil {
		return resourceSchemaNamingConventionSchema{}
	}

	path := oas.JoinAll(item.Path, method, "responses", code, resourceSchemaNamingConventionContent, name, "schema")

	return c.named(oas.Get(media, "schema"), path)
}

// successCode returns 200 when present, else the first 2xx code with a JSON schema, else the empty string.
func (c *resourceSchemaNamingConventionChecker) successCode(responses *yaml.Node) string {
	success := regexp.MustCompile(`^2[0-9][0-9]$`)
	codes := seq.Filter(slices.Values(oas.Keys(responses)), success.MatchString)

	if slices.Contains(codes, resourceSchemaNamingConventionOK) {
		return resourceSchemaNamingConventionOK
	}

	index := slices.IndexFunc(codes, func(candidate string) bool {
		_, media := c.jsonMediaNamed(oas.Resolve(c.root, oas.Get(responses, candidate)))

		return media != nil
	})
	if index < 0 {
		return ""
	}

	return codes[index]
}

// jsonMediaNamed returns the name and object of the first application/json
// media type of a response; the schema must be present.
func (c *resourceSchemaNamingConventionChecker) jsonMediaNamed(response *yaml.Node) (string, *yaml.Node) {
	content := oas.Get(response, resourceSchemaNamingConventionContent)
	names := seq.Filter(slices.Values(oas.Keys(content)), func(name string) bool {
		return resourceSchemaNamingConventionMime(name, "json")
	})
	if len(names) == 0 {
		return "", nil
	}

	media := oas.Get(content, names[0])
	if !oas.IsMap(media) || oas.Get(media, "schema") == nil {
		return "", nil
	}

	return names[0], media
}

// named wraps a schema node with the component name it references.
func (c *resourceSchemaNamingConventionChecker) named(
	node *yaml.Node,
	path string,
) resourceSchemaNamingConventionSchema {
	return resourceSchemaNamingConventionSchema{node: node, path: path, name: oas.RefName(oas.Ref(node))}
}

// basePath returns the path used for locations inside the resolved schema.
func (c *resourceSchemaNamingConventionChecker) basePath(schema resourceSchemaNamingConventionSchema) string {
	ref := oas.Ref(schema.node)
	if ref == "" || !strings.HasPrefix(ref, "#/") {
		return schema.path
	}

	return oas.JoinAll("$", seq.FlatMapSlice(
		strings.Split(strings.TrimPrefix(ref, "#/"), "/"),
		func(token string) []string {
			return []string{strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")}
		},
	)...)
}

// hasProperty is IBM's schemaHasProperty: the property is declared by the
// schema, by any allOf member, or by every oneOf and anyOf member.
func (c *resourceSchemaNamingConventionChecker) hasProperty(schema *yaml.Node, name string, depth int) bool {
	resolved := oas.Resolve(c.root, schema)
	if !oas.IsMap(resolved) || depth > resourceSchemaNamingConventionMaxDepth {
		return false
	}

	if oas.Dig(resolved, "properties", name) != nil {
		return true
	}

	allOf := oas.Get(resolved, resourceSchemaNamingConventionAllOf)
	if oas.IsSeq(allOf) && slices.ContainsFunc(allOf.Content, func(member *yaml.Node) bool {
		return c.hasProperty(member, name, depth+1)
	}) {
		return true
	}

	return slices.ContainsFunc(
		[]string{resourceSchemaNamingConventionOneOf, resourceSchemaNamingConventionAnyOf},
		func(keyword string) bool {
			members := oas.Get(resolved, keyword)
			if !oas.IsSeq(members) || len(members.Content) == 0 {
				return false
			}

			return !slices.ContainsFunc(members.Content, func(member *yaml.Node) bool {
				return !c.hasProperty(member, name, depth+1)
			})
		},
	)
}

// property finds a named property through properties and composition keywords,
// like IBM's computePathToProperty, and returns it with its path. The zero value
// means that it was not found.
func (c *resourceSchemaNamingConventionChecker) property(
	schema *yaml.Node,
	path, name string,
	depth int,
) resourceSchemaNamingConventionProperty {
	if !oas.IsMap(schema) || depth > resourceSchemaNamingConventionMaxDepth {
		return resourceSchemaNamingConventionProperty{}
	}

	if value := oas.Dig(schema, "properties", name); value != nil {
		return resourceSchemaNamingConventionProperty{
			node: oas.Resolve(c.root, value),
			path: oas.JoinAll(path, "properties", name),
		}
	}

	keywords := []string{
		resourceSchemaNamingConventionAllOf,
		resourceSchemaNamingConventionOneOf,
		resourceSchemaNamingConventionAnyOf,
	}
	found := seq.FlatMapSlice(keywords, func(keyword string) []resourceSchemaNamingConventionProperty {
		return seq.FlatMapPairs(
			oas.Items(oas.Get(schema, keyword)),
			func(i int, member *yaml.Node) []resourceSchemaNamingConventionProperty {
				property := c.property(
					oas.Resolve(c.root, member),
					oas.Index(oas.Join(path, keyword), i),
					name,
					depth+1,
				)

				if property.node == nil {
					return nil
				}

				return []resourceSchemaNamingConventionProperty{property}
			},
		)
	})
	if len(found) == 0 {
		return resourceSchemaNamingConventionProperty{}
	}

	return found[0]
}

// resourceSchemaNamingConventionFinding builds a finding for a schema.
func resourceSchemaNamingConventionFinding(schema resourceSchemaNamingConventionSchema, message string) Finding {
	return Finding{Node: schema.node, Path: schema.path, Message: message}
}

// resourceSchemaNamingConventionLastSegment returns the last segment of a path template.
func resourceSchemaNamingConventionLastSegment(route string) string {
	return route[strings.LastIndex(route, "/")+1:]
}

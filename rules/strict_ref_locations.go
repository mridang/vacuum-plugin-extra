package rules

import (
	"iter"
	"maps"
	"slices"
	"strings"

	"github.com/daveshanley/vacuum/model"
	"github.com/pb33f/go-yaml"

	"github.com/mridang/vacuum-plugin-extra/internal/oas"
	"github.com/mridang/vacuum-plugin-extra/internal/seq"
)

const (
	strictRefLocationsMapPrefix  = "map:"
	strictRefLocationsListPrefix = "list:"
	strictRefLocationsAny        = "*"
	strictRefLocationsRef        = "$ref"
	strictRefLocationsMediaType  = "MediaType"
	strictRefLocationsSchema     = "Schema"
	strictRefLocationsPathItem   = "PathItem"
	strictRefLocationsHeader     = "Header"
	strictRefLocationsExample    = "Example"
	strictRefLocationsServer     = "Server"
	strictRefLocationsExtPrefix  = "x-"
	strictRefLocationsVersion32  = "3.2"
	strictRefLocationsMaxDepth   = 64
)

// StrictRefLocations returns the extra-strict-ref-locations rule.
//
// # What it checks
//
// A port of Redocly CLI's spec-strict-refs. The OpenAPI specification only
// makes some objects referenceable. The document is walked following the
// OpenAPI object model, and a $ref found in an object whose type is not in the
// list below is reported on the $ref keyword with the message "Field $ref is
// not expected here.".
//
// Referenceable (a $ref is accepted): Schema, Response, Parameter, RequestBody,
// Example, Header, SecurityScheme, Link, Callback, PathItem, and MediaType in
// a content entry of OpenAPI 3.2.
//
// Everything else rejects a $ref, for example: Info, Contact, License, Server,
// ServerVariable, Tag, ExternalDocs, Paths, Operation, Responses, Encoding,
// Discriminator, XML, OAuthFlows and OAuthFlow, security requirements,
// Components and each of its maps (components.schemas itself), and, before
// OpenAPI 3.2, MediaType.
//
// Free-form regions are not examined: example, examples values, default, enum,
// const, and everything below x- extension keys.
//
// # Where it comes from
//
// Redocly CLI, rule spec-strict-refs
// (packages/core/src/rules/common/spec-strict-refs.ts).
//
// # Options
//
// None.
//
// # Bad
//
//	info:
//	  contact:
//	    $ref: "#/components/x-contact"
//	paths:
//	  /pets:
//	    get:
//	      responses:
//	        "200":
//	          description: ok
//	          content:
//	            application/json:
//	              $ref: "#/components/x-media"
//
// # Good
//
//	paths:
//	  /pets:
//	    $ref: "#/components/pathItems/Pets"
//
// # Known false positives and limitations
//
// The document is classified by position in the standard object model, so an
// object in a position the model does not know (a misspelled keyword) is not
// examined. Only local structure is classified; the target of a $ref is checked
// where it is defined. Swagger 2.0 documents are not modelled.
func StrictRefLocations() Rule {
	return New(Spec{
		Name:        "strict-ref-locations",
		Description: "A $ref may only appear in objects that the specification allows to be references.",
		HowToFix:    "Inline the referenced object, or move the $ref to an object that the specification allows to be a reference.",
		Severity:    model.SeverityError,
		Given:       "$",
		Type:        TypeValidation,
		Check:       OnDocument(checkStrictRefLocations),
	})
}

// strictRefLocationsReferenceable lists the object types that accept a $ref.
func strictRefLocationsReferenceable() []string {
	return strings.Fields("Schema Response Parameter RequestBody Example Header SecurityScheme Link Callback PathItem")
}

// strictRefLocationsModel describes the OpenAPI object model: one line per
// object type, followed by key=type entries. A type is a plain object type,
// map:T for a map of T or list:T for a list of T, and the key * stands for every
// key that is not listed.
const strictRefLocationsModel = `
Root: info=Info servers=list:Server tags=list:Tag paths=Paths components=Components externalDocs=ExternalDocs webhooks=map:PathItem security=list:SecurityRequirement
Info: contact=Contact license=License
Tag: externalDocs=ExternalDocs
Server: variables=map:ServerVariable
Paths: *=PathItem
PathItem: parameters=list:Parameter servers=list:Server additionalOperations=map:Operation get=Operation put=Operation post=Operation delete=Operation options=Operation head=Operation patch=Operation trace=Operation query=Operation
Operation: parameters=list:Parameter requestBody=RequestBody responses=Responses callbacks=map:Callback servers=list:Server externalDocs=ExternalDocs security=list:SecurityRequirement
Responses: *=Response
Response: headers=map:Header content=map:MediaType links=map:Link
MediaType: schema=Schema itemSchema=Schema examples=map:Example encoding=map:Encoding
Encoding: headers=map:Header
Parameter: schema=Schema examples=map:Example content=map:MediaType
Header: schema=Schema examples=map:Example content=map:MediaType
RequestBody: content=map:MediaType
Callback: *=PathItem
Link: server=Server
Components: schemas=map:Schema responses=map:Response parameters=map:Parameter examples=map:Example requestBodies=map:RequestBody headers=map:Header securitySchemes=map:SecurityScheme links=map:Link callbacks=map:Callback pathItems=map:PathItem mediaTypes=map:MediaType
SecurityScheme: flows=OAuthFlows
OAuthFlows: implicit=OAuthFlow password=OAuthFlow clientCredentials=OAuthFlow authorizationCode=OAuthFlow deviceAuthorization=OAuthFlow
Schema: discriminator=Discriminator xml=XML externalDocs=ExternalDocs properties=map:Schema patternProperties=map:Schema dependentSchemas=map:Schema $defs=map:Schema allOf=list:Schema anyOf=list:Schema oneOf=list:Schema prefixItems=list:Schema items=Schema additionalProperties=Schema not=Schema contains=Schema propertyNames=Schema if=Schema then=Schema else=Schema unevaluatedProperties=Schema unevaluatedItems=Schema
`

// strictRefLocationsParseModel turns the model table into a lookup from object
// type to its children.
func strictRefLocationsParseModel() map[string]map[string]string {
	return maps.Collect(func(yield func(string, map[string]string) bool) {
		for line := range strings.SplitSeq(strictRefLocationsModel, "\n") {
			kind, entries, found := strings.Cut(line, ": ")
			if !found {
				continue
			}

			if !yield(kind, maps.Collect(strictRefLocationsChildren(entries))) {
				return
			}
		}
	})
}

// strictRefLocationsChildren yields the key and child type of each entry.
func strictRefLocationsChildren(entries string) iter.Seq2[string, string] {
	return func(yield func(string, string) bool) {
		for entry := range strings.FieldsSeq(entries) {
			key, child, _ := strings.Cut(entry, "=")
			if !yield(key, child) {
				return
			}
		}
	}
}

// strictRefLocationsChild looks up the child type of a key, falling back to the
// wildcard entry.
func strictRefLocationsChild(children map[string]string, key string) (string, bool) {
	if child, ok := children[key]; ok {
		return child, true
	}

	child, ok := children[strictRefLocationsAny]

	return child, ok
}

// strictRefLocationsWalker carries the state of one walk.
type strictRefLocationsWalker struct {
	model          map[string]map[string]string
	allowMediaType bool
}

func checkStrictRefLocations(call *Call) []Finding {
	walker := strictRefLocationsWalker{
		model:          strictRefLocationsParseModel(),
		allowMediaType: strings.HasPrefix(oas.Version(call.Root), strictRefLocationsVersion32),
	}

	return walker.visit("Root", oas.Unwrap(call.Root), "$", 0)
}

func (w strictRefLocationsWalker) visit(kind string, node *yaml.Node, path string, depth int) []Finding {
	if depth > strictRefLocationsMaxDepth || node == nil {
		return nil
	}

	switch {
	case strings.HasPrefix(kind, strictRefLocationsMapPrefix):
		return w.visitMap(strings.TrimPrefix(kind, strictRefLocationsMapPrefix), node, path, depth)
	case strings.HasPrefix(kind, strictRefLocationsListPrefix):
		return seq.FlatMapPairs(oas.Items(node), func(i int, item *yaml.Node) []Finding {
			return w.visit(strings.TrimPrefix(kind, strictRefLocationsListPrefix), item, oas.Index(path, i), depth+1)
		})
	default:
		return w.visitObject(kind, node, path, depth)
	}
}

func (w strictRefLocationsWalker) visitMap(element string, node *yaml.Node, path string, depth int) []Finding {
	if !oas.IsMap(node) {
		return nil
	}

	return slices.Concat(
		w.checkRef("NamedMap", node, path),
		seq.FlatMapPairs(oas.Pairs(node), func(key, value *yaml.Node) []Finding {
			if key.Value == strictRefLocationsRef || strings.HasPrefix(key.Value, strictRefLocationsExtPrefix) {
				return nil
			}

			return w.visit(element, value, oas.Join(path, key.Value), depth+1)
		}),
	)
}

func (w strictRefLocationsWalker) visitObject(kind string, node *yaml.Node, path string, depth int) []Finding {
	if !oas.IsMap(node) {
		return nil
	}

	children := w.model[kind]

	return slices.Concat(
		w.checkRef(kind, node, path),
		seq.FlatMapPairs(oas.Pairs(node), func(key, value *yaml.Node) []Finding {
			if key.Value == strictRefLocationsRef || strings.HasPrefix(key.Value, strictRefLocationsExtPrefix) {
				return nil
			}

			child, ok := strictRefLocationsChild(children, key.Value)
			if !ok {
				return nil
			}

			return w.visit(child, value, oas.Join(path, key.Value), depth+1)
		}),
	)
}

func (w strictRefLocationsWalker) checkRef(kind string, node *yaml.Node, path string) []Finding {
	key, ref := oas.Entry(node, strictRefLocationsRef)

	return When(
		oas.IsScalar(ref) && !slices.Contains(strictRefLocationsReferenceable(), kind) &&
			(kind != strictRefLocationsMediaType || !w.allowMediaType),
		func() Finding {
			return Finding{
				Node:    ref,
				Path:    oas.Join(path, key.Value),
				Message: "Field $ref is not expected here.",
			}
		},
	)
}

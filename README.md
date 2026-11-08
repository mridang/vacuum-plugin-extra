# vacuum-plugin-extra

[![License](https://img.shields.io/badge/license-Apache--2.0-blue.svg)](LICENSE)

Extra lint rules for the [vacuum](https://github.com/daveshanley/vacuum) OpenAPI linter. The rules were collected by comparing vacuum with a survey of other OpenAPI linters (Spectral, Zally, Redocly, IBM openapi-validator, Oxide, specmatic, speakeasy and others) and keeping the checks vacuum does not ship. Every rule is a vacuum custom function with an `extra-` prefixed identifier, so none of them clash with vacuum's built-in rules.

> [!WARNING]
> Pre-release. Rule behaviour and options may shift before `v1.0.0`.

## Install

The easiest way to run the rules is the `vacuum-extra` command. It is one statically linked binary that embeds vacuum's engine and every rule, so there is no plugin to build and no version to match.

```bash
go install github.com/mridang/vacuum-plugin-extra/cmd/vacuum-extra@latest
```

## Usage

```bash
vacuum-extra openapi.yaml
vacuum-extra -format json -fail-on warn openapi.yaml
vacuum-extra -only extra-unique-tag-names,extra-operation-id-casing openapi.yaml
vacuum-extra -r my-ruleset.yaml openapi.yaml
vacuum-extra -list
vacuum-extra -debug openapi.yaml
```

The command exits with 0 when nothing at or above the `-fail-on` severity (default `error`; use `none` to never fail) is found, 1 when something is, and 2 for invalid usage. `-r` replaces the embedded ruleset with your own, which can configure any rule's severity and options.

`-debug` (or `VACUUM_EXTRA_DEBUG=1`) writes a `log/slog` trace to stderr: the ruleset that was loaded, every rule that is skipped and why (not selected by `-only`, `severity: off`, or formats that do not match the document), and for each rule that runs its id, `given` path, options, the number of nodes it visited and the number of findings. Nothing is logged and the output does not change without it.

```text
level=DEBUG msg="ruleset loaded" source=embedded rules=165 selected=2 openapi=3.0.3
level=DEBUG msg=run rule=extra-summary-no-trailing-period given=$ options=map[] nodes=1
level=DEBUG msg=visited rule=extra-summary-no-trailing-period kind=operation count=1
level=DEBUG msg=done rule=extra-summary-no-trailing-period findings=1
```

The rules read `$ref` nodes as written, so every rule of the embedded ruleset sets `resolved: false`, and the command forces it for `extra-` rules of a ruleset passed with `-r`. A document that splits itself over several files with relative `$ref`s is bundled first (file references are pulled in, references to components stay references), so the rules see the whole API; line numbers in the findings then refer to the bundled text, not to the files on disk. Remote `http(s)` references are not fetched.

`ruleset.yaml` in the repository root is the embedded ruleset. It lists every rule with its severity, `given` JSONPath, how-to-fix message and default options. A rule can be switched off with `severity: off`. Rules that need configuration to do anything, such as `extra-banned-words-in-docs`, are inert until their options are filled in. `ruleset.yaml` is generated with `make generate`; do not edit it by hand.

### Switching rules off and changing severities

A custom ruleset passed with `-r` lists only the rules it names; a rule that is not in it does not run. Copy the complete entries you want from `ruleset.yaml` (vacuum rejects an entry without `given` and `then`) and change their `severity`, which can be `error`, `warn`, `info`, `hint` or `off` (the values of vacuum's ruleset schema; anything else, including `none`, is rejected). A rule set to `off` is skipped: it does not run, reports nothing and, with `-debug`, logs one `rule skipped` line. This works for every rule, also when its `functionOptions` are not valid. The severity of the other rules is the one reported in the findings and is what `-fail-on` compares against; `-fail-on none` is the only place `none` is accepted, and it means the run never fails.

```yaml
rules:
  extra-summary-no-trailing-period:
    description: Operation summaries must not end with a period.
    message: "{{error}}"
    given: $
    resolved: false
    severity: off
    type: style
    then:
      function: extra-summary-no-trailing-period
  extra-no-eval-in-title:
    description: Titles and descriptions must not contain eval(.
    message: "{{error}}"
    given: $
    resolved: false
    severity: info
    type: validation
    then:
      function: extra-no-eval-in-title
```

### As a Go library

Programs that embed vacuum can register the rules in-process:

```go
import (
	"github.com/daveshanley/vacuum/plugin"

	extra "github.com/mridang/vacuum-plugin-extra"
)

manager := plugin.CreatePluginManager()
extra.Register(manager)
// manager.GetCustomFunctions() is what RuleSetExecution.CustomFunctions expects.
```

### As a vacuum Go plugin

vacuum's own CLI loads Go custom functions from a plugin file. This works, but a Go plugin must be built with exactly the same Go toolchain and exactly the same vacuum and dependency versions as the vacuum binary that loads it, and it is specific to the operating system and CPU. Prefer `vacuum-extra` unless you build vacuum yourself.

```bash
mkdir -p plugins
go build -buildmode=plugin -o plugins/extra.so ./plugin
vacuum lint -r ruleset.yaml -f ./plugins openapi.yaml
```

## How the rules were assembled

The source list had 192 rows. Several rows describe the same check, so they were merged into one rule, giving 165 distinct rules:

- `extra-array-requires-items` merges `array-items-required`, `array-requires-items`.
- `extra-component-schema-name-case` merges `component-schema-name-case`, `schema-name-upper-camel-case`.
- `extra-example-value-xor-external` merges `example-object-value-required`, `example-value-xor-external`.
- `extra-link-target-exists` merges `link-operation-id-resolves`, `link-target-exists`.
- `extra-no-default-for-required-parameter` merges `no-default-for-required-parameter`, `required-parameter-must-not-have-default`.
- `extra-no-inline-schema` merges `no-inline-parameter-schema`, `no-inline-request-body-schema`, `no-inline-response-schema`.
- `extra-no-reserved-header-parameter` merges `no-reserved-header-parameter-accept`, `no-reserved-header-parameter-authorization`, `no-reserved-header-parameter-content-type`, `no-reserved-header-parameter-if-modified-since`, `no-reserved-header-parameter-if-unmodified-since`.
- `extra-operation-id-casing` merges `operation-id-casing`, `operation-id-charset-strict`.
- `extra-operation-max-parameters` merges `operation-max-parameters`, `operation-max-required-parameters`.
- `extra-parameter-name-constraints` merges `parameter-name-charset`, `parameter-name-max-length`, `parameter-name-nonempty`.
- `extra-path-segment-plural` merges `path-segment-plural`, `resource-path-plural`.
- `extra-response-required-properties` merges `required-array-properties-in-response`, `required-enum-properties-in-response`.
- `extra-schema-type-keyword-mismatch` merges `schema-type-keyword-mismatch`, `schema-with-properties-must-be-type-object`.
- `extra-security-scopes-defined` merges `security-scope-consistency`, `security-scopes-defined`.
- `extra-server-variables-consistent` merges `server-variable-default-nonempty`, `server-variable-enum-default`, `server-variable-enum-defined`, `server-variables-consistent`.
- `extra-string-length-limits` merges `min-length-positive`, `string-max-length-ceiling`.
- `extra-summary-no-trailing-period` merges `operation-summary-no-trailing-period`, `summary-no-trailing-period`.
- `extra-unique-tag-names` merges `duplicate-tag-name`, `tags-unique-names`, `unique-tag-names`.
- `extra-webhook-restrictions` merges `webhook-no-callbacks`, `webhook-no-servers`.

The five `no-reserved-header-parameter-*` rows became one rule parameterised by header name, and the `server-variable-*` rows became one rule with a `requireEnum` option.

Some rules are a conservative subset of the check in the linter they come from, for example where the original needs a regular expression solver or a pluralisation library. Each rule's Go doc comment states what it checks, why, where it comes from, its options, a bad and a good example, and its known false positives or limitations.

## Rules

| Rule                                                | Severity | What it checks                                                                                                                 | Source                             | Options                                                                             |
| --------------------------------------------------- | -------- | ------------------------------------------------------------------------------------------------------------------------------ | ---------------------------------- | ----------------------------------------------------------------------------------- |
| `extra-action-verbs-in-path`                        | warn     | Path segments must be nouns, not action verbs such as create or delete.                                                        | ai-api-lint                        | `verbs`                                                                             |
| `extra-additional-properties-object-only`           | warn     | additionalProperties is only valid inside object schemas.                                                                      | specmatic                          | -                                                                                   |
| `extra-admin-security-differs`                      | error    | Admin endpoints must not share a security requirement with non-admin endpoints.                                                | Spectral                           | `adminUrl`                                                                          |
| `extra-allof-schema-type-intersection`              | error    | A schema and its immediate allOf branches must declare types with a non-empty intersection.                                    | speakeasy                          | -                                                                                   |
| `extra-allow-reserved-query-only`                   | warn     | allowReserved must only be set on query parameters.                                                                            | KaiZen                             | -                                                                                   |
| `extra-api-max-operations`                          | info     | The document must not define more operations than the configured limit.                                                        | ai-api-lint                        | `maxOperations`                                                                     |
| `extra-api-symmetry-get-vs-write-schemas`           | warn     | Summary, Prototype and Patch variants of a resource schema must be graph fragments of its canonical schema.                    | IBM                                | -                                                                                   |
| `extra-api-version-in-url`                          | info     | The API must expose a version in a server URL, a path or a version header.                                                     | apilint                            | -                                                                                   |
| `extra-array-bounds-and-items-consistency`          | warn     | Array schemas must define items and should define consistent minItems and maxItems.                                            | IBM                                | `requireBounds`                                                                     |
| `extra-array-parameter-serialization`               | warn     | Array parameters must declare both style and explode.                                                                          | Redocly                            | `in`                                                                                |
| `extra-array-requires-items`                        | error    | Schemas of type array must define items.                                                                                       | Spectral, scalar                   | `allowPrefixItems`                                                                  |
| `extra-avoid-inline-object-schemas`                 | warn     | Object schemas must be defined in components/schemas and referenced instead of being inlined.                                  | IBM                                | -                                                                                   |
| `extra-banned-words-in-docs`                        | warn     | Descriptions, summaries and titles must not contain the configured banned words.                                               | openapilint                        | `words`                                                                             |
| `extra-boolean-parameter-prefix`                    | warn     | Boolean parameter names must start with one of the configured prefixes.                                                        | Redocly                            | `prefixes`                                                                          |
| `extra-collection-response-has-named-array`         | warn     | List operation responses must contain an array property named after the last path segment.                                     | IBM                                | `operationIdPrefix`                                                                 |
| `extra-component-name-max-length`                   | info     | Component names must not be longer than a maximum length.                                                                      | RMOA                               | `maxLength`                                                                         |
| `extra-component-schema-name-case`                  | warn     | Component schema names must be upper camel case.                                                                               | IBM, Oxide                         | `style`                                                                             |
| `extra-composition-branches-same-type`              | warn     | All leaf schemas of a oneOf, allOf, anyOf or not must have the same type.                                                      | Oxide                              | -                                                                                   |
| `extra-consistent-property-name-and-type`           | warn     | Properties with the same name in different schemas must have the same type.                                                    | IBM                                | `ignore`                                                                            |
| `extra-content-must-have-schema`                    | error    | Every content entry of a request body, response or parameter must contain a schema.                                            | IBM                                | -                                                                                   |
| `extra-content-type-allow-list`                     | warn     | Request and response media types must be in the allowed list.                                                                  | apilint                            | `mimetypes`, `validateExamples`                                                     |
| `extra-cors-origin-header`                          | warn     | Every response must define the Access-Control-Allow-Origin header.                                                             | Spectral                           | `header`                                                                            |
| `extra-date-named-property-must-use-date-format`    | warn     | Date-based schemas and properties must be strings with format date or date-time.                                               | IBM                                | `names`                                                                             |
| `extra-date-time-property-suffix`                   | warn     | String properties with a date, time or date-time format must have a matching name.                                             | Zally                              | `patterns`                                                                          |
| `extra-deepobject-example-keys`                     | warn     | Examples of deepObject query parameters must use keys defined by the parameter schema.                                         | specmatic                          | -                                                                                   |
| `extra-default-equals-example-redundant`            | info     | A schema example must not be identical to the schema default.                                                                  | speccy                             | -                                                                                   |
| `extra-description-usage-hint-wording`              | info     | Operation descriptions should contain a usage hint word such as returns, allows or use.                                        | ai-api-lint                        | `hintWords`, `minLength`                                                            |
| `extra-discourage-http-422`                         | warn     | Operations should use status code 400 instead of 422.                                                                          | baloise-incubator/Spectral-ruleset | -                                                                                   |
| `extra-discriminator-property-exists`               | error    | The property named by a discriminator must exist in the schema.                                                                | IBM                                | -                                                                                   |
| `extra-discriminator-property-required`             | error    | The discriminator property must be listed in the required list of the schema.                                                  | KaiZen                             | -                                                                                   |
| `extra-document-has-operations`                     | warn     | The document must define at least one operation.                                                                               | oas-kit                            | -                                                                                   |
| `extra-empty-required-or-enum`                      | error    | The required and enum lists of a schema must be non-empty and free of duplicates.                                              | KaiZen                             | -                                                                                   |
| `extra-encoding-keys-match-properties`              | error    | Encoding map keys must name properties of the media type schema.                                                               | KaiZen                             | -                                                                                   |
| `extra-enum-value-case`                             | warn     | String enum values must be snake_case or SCREAMING_SNAKE_CASE.                                                                 | Oxide                              | `allowSnake`, `allowScreamingSnake`                                                 |
| `extra-error-container-schema-shape`                | warn     | Error response schemas must follow the API Handbook error container model.                                                     | IBM                                | -                                                                                   |
| `extra-error-response-content-type`                 | warn     | Error responses that define content must offer an application/json entry.                                                      | IBM                                | -                                                                                   |
| `extra-error-response-has-body`                     | warn     | 4xx responses must declare a response body with a schema.                                                                      | apilint                            | `requireSchema`                                                                     |
| `extra-etag-with-conditional-headers`               | warn     | A path item with If-Match or If-None-Match parameters must return an ETag header from GET.                                     | IBM                                | -                                                                                   |
| `extra-example-body-key-case`                       | warn     | Property names in example bodies must follow the configured case style.                                                        | apilint                            | `style`                                                                             |
| `extra-example-value-xor-external`                  | error    | An Example Object must define exactly one of value and externalValue.                                                          | scalar, Spectral, speccy, oas-kit  | `forbidBoth`, `requireOne`                                                          |
| `extra-exploded-object-query-required-property`     | warn     | A required exploded object query parameter needs at least one required property.                                               | specmatic                          | -                                                                                   |
| `extra-extensible-enum-string-type`                 | error    | Schemas with x-extensible-enum must have type string.                                                                          | Zally                              | -                                                                                   |
| `extra-format-matches-type`                         | warn     | Standard formats must be used with a compatible type.                                                                          | KaiZen                             | -                                                                                   |
| `extra-global-security-required`                    | warn     | The root security property must exist and hold at least one security requirement.                                              | RMOA                               | -                                                                                   |
| `extra-header-name-max-length`                      | warn     | Response header names must not exceed the maximum length.                                                                      | RMOA                               | `maxLength`                                                                         |
| `extra-hoist-repeated-path-parameters`              | warn     | Identical path parameters repeated across operations must move to the path item.                                               | IBM                                | -                                                                                   |
| `extra-ignored-header-names`                        | warn     | Header parameters named Accept, Content-Type or Authorization are ignored.                                                     | KaiZen                             | `names`                                                                             |
| `extra-illogical-composition-keywords`              | warn     | allOf, anyOf and oneOf must combine schemas that a value can actually resolve against.                                         | Redocly                            | -                                                                                   |
| `extra-integer-bounds-in-format-range`              | warn     | Integer schemas must define an int32 or int64 format and bounds inside its range.                                              | IBM                                | `requireFormat`, `requireBounds`                                                    |
| `extra-link-parameter-exists`                       | error    | Links must identify an existing operation and use parameters that exist in it.                                                 | KaiZen                             | `reportRemoteOperationRef`                                                          |
| `extra-link-target-exists`                          | error    | Link objects must reference an operation that exists in the document.                                                          | KaiZen, speakeasy                  | `operationRefMethods`                                                               |
| `extra-major-version-in-path`                       | warn     | The major version must be a v<n> segment in a server URL or at the start of every path.                                        | IBM                                | `allowServerVersion`, `consistent`                                                  |
| `extra-media-type-allow-list`                       | error    | Media types must be in the configured allow-list.                                                                              | Redocly                            | `allowed`, `includeResponses`                                                       |
| `extra-merge-patch-no-required`                     | warn     | Schemas of merge patch request bodies must not define required properties.                                                     | IBM                                | -                                                                                   |
| `extra-multiple-path-params`                        | info     | Paths should not contain more than one path parameter.                                                                         | Zally                              | `max`                                                                               |
| `extra-no-array-of-arrays`                          | warn     | Array schemas must not use another array schema as their items.                                                                | IBM                                | -                                                                                   |
| `extra-no-binary-in-json-api`                       | warn     | Binary string schemas must not be used for parameters or JSON request and response bodies.                                     | IBM                                | -                                                                                   |
| `extra-no-body-on-safe-methods`                     | error    | Operations with a method such as GET, HEAD, DELETE or OPTIONS must not define a request body.                                  | IBM                                | `httpMethods`                                                                       |
| `extra-no-circular-schema-refs`                     | warn     | Component schemas must not reference themselves through $ref cycles.                                                           | IBM                                | `ignoreProperties`                                                                  |
| `extra-no-consecutive-path-parameter-segments`      | warn     | A path must not contain two consecutive path parameter segments.                                                               | IBM                                | -                                                                                   |
| `extra-no-default-for-required-parameter`           | warn     | A required parameter must not have a default value.                                                                            | IBM, openapilint                   | `onlyTruthy`                                                                        |
| `extra-no-eval-in-title`                            | error    | Titles and descriptions must not contain eval(.                                                                                | speccy, oas-kit                    | -                                                                                   |
| `extra-no-inline-schema`                            | info     | Parameter, request body and response schemas must be $refs to reusable schemas.                                                | RMOA                               | `parameters`, `requestBodies`, `responses`, `allowPrimitives`                       |
| `extra-no-link-header`                              | warn     | Header parameters and response headers must not be named Link.                                                                 | Zally                              | -                                                                                   |
| `extra-no-mixed-number-range-constraints`           | warn     | Schemas must not define both an inclusive and a numeric exclusive bound on the same side.                                      | Redocly                            | -                                                                                   |
| `extra-no-nested-callbacks`                         | warn     | A callback must not contain its own callbacks.                                                                                 | Spectral                           | -                                                                                   |
| `extra-no-nullable-properties`                      | warn     | Properties must not be nullable, except in merge-patch request bodies.                                                         | IBM                                | -                                                                                   |
| `extra-no-path-item-level-parameters`               | warn     | Path items must not define a parameters list.                                                                                  | openapilint                        | -                                                                                   |
| `extra-no-ref-in-example`                           | error    | A $ref must not be used inside an example field; use examples instead.                                                         | IBM                                | -                                                                                   |
| `extra-no-reserved-header-parameter`                | warn     | Header parameters must not be named Accept, Authorization, Content-Type or similar reserved headers.                           | IBM                                | `headers`                                                                           |
| `extra-no-uuid-suffix-property`                     | warn     | UUID properties must not have names ending in _uuid.                                                                           | Oxide                              | -                                                                                   |
| `extra-no-whitespace-in-example-keys`               | warn     | Keys of response examples maps must not contain whitespace.                                                                    | IBM                                | -                                                                                   |
| `extra-no-wildcard-media-type`                      | warn     | Content maps must not use the _/_ media type.                                                                                  | IBM                                | -                                                                                   |
| `extra-nullable-body-schema`                        | warn     | Request body schemas must not be nullable.                                                                                     | Oxide                              | -                                                                                   |
| `extra-nullable-requires-type`                      | warn     | A schema using nullable must declare a sibling type.                                                                           | Redocly, speakeasy                 | -                                                                                   |
| `extra-oauth2-refresh-url`                          | error    | OAuth2 flows other than clientCredentials must define a refreshUrl.                                                            | Spectral                           | -                                                                                   |
| `extra-oauth2-scope-name-pattern`                   | error    | OAuth2 scope names must match the configured pattern and every endpoint must be secured by scopes.                             | Zally                              | `pattern`, `allowed`, `pathWhitelist`                                               |
| `extra-openapi31-allowed-schema-keywords`           | error    | OpenAPI 3.1 schemas may only use allow-listed keywords or x- extensions.                                                       | IBM                                | `allowedKeywords`                                                                   |
| `extra-operation-2xx-required`                      | warn     | Every operation must declare a 2xx response.                                                                                   | ai-api-lint                        | -                                                                                   |
| `extra-operation-description-min-chars`             | info     | The longer of an operation's description and summary must reach a minimum length.                                              | ai-api-lint                        | `minLength`                                                                         |
| `extra-operation-description-opens-with-sentence`   | warn     | Operation descriptions must open with a sentence, not a heading, quote, list, table, code fence or HTML.                       | Redocly                            | -                                                                                   |
| `extra-operation-description-restates-path`         | warn     | Operation descriptions and summaries must not merely restate the method and path.                                              | ai-api-lint                        | -                                                                                   |
| `extra-operation-id-casing`                         | warn     | operationId must follow the configured case convention.                                                                        | IBM, Oxide, ai-api-lint            | `style`, `strictCharset`                                                            |
| `extra-operation-id-max-length`                     | warn     | The operationId must not be longer than the configured limit.                                                                  | ai-api-lint, RMOA                  | `maxLength`                                                                         |
| `extra-operation-id-max-prefixes`                   | info     | The document must not use more than a few distinct operationId verb prefixes.                                                  | ai-api-lint                        | `max`                                                                               |
| `extra-operation-id-reserved-word`                  | warn     | The operationId must not be a reserved word of the target language.                                                            | ai-api-lint                        | `words`                                                                             |
| `extra-operation-id-verb-matches-method`            | warn     | The verb and resource of an operationId must match the HTTP method and path of the operation.                                  | IBM                                | `strict`                                                                            |
| `extra-operation-id-verb-prefix`                    | warn     | An operationId must start with a common verb followed by an upper-case letter.                                                 | ai-api-lint                        | `prefixes`                                                                          |
| `extra-operation-max-parameters`                    | info     | Operations must not have too many parameters or required parameters.                                                           | ai-api-lint                        | `maxParameters`, `maxRequiredParameters`                                            |
| `extra-operation-summary-max-length`                | warn     | The operation summary must not be longer than the configured limit.                                                            | IBM, speccy                        | `maxLength`                                                                         |
| `extra-operation-summary-required`                  | warn     | Every operation must have a non-empty summary.                                                                                 | IBM, Redocly, RMOA                 | -                                                                                   |
| `extra-optional-body-with-required-properties`      | warn     | An optional request body must not have a schema with required properties.                                                      | IBM                                | -                                                                                   |
| `extra-pagination-parameters-shape`                 | warn     | Paginated list operations must define limit, offset or page token parameters, response properties and page links consistently. | IBM                                | -                                                                                   |
| `extra-param-name-vs-body-property-name`            | warn     | Parameter names must not equal top-level property names of the JSON request body.                                              | IBM                                | -                                                                                   |
| `extra-parameter-name-constraints`                  | warn     | Parameter names must be non-empty, short and match the configured pattern.                                                     | RMOA, speccy, oas-kit              | `pattern`, `maxLength`, `minLength`                                                 |
| `extra-parameter-schema-type-required`              | warn     | Every parameter must have a schema with a declared type.                                                                       | ai-api-lint                        | -                                                                                   |
| `extra-patch-content-type`                          | warn     | PATCH request bodies must use a JSON Patch or JSON Merge Patch media type.                                                     | IBM                                | `allowed`                                                                           |
| `extra-path-http-verbs-order`                       | info     | Operations within a path item must follow the configured HTTP method order.                                                    | Redocly                            | `order`                                                                             |
| `extra-path-item-summary-or-description`            | info     | Each path item must have a summary or a description.                                                                           | oas-kit                            | -                                                                                   |
| `extra-path-param-placement`                        | warn     | Paths must not start with a path parameter, and path parameters must fill a whole segment.                                     | Zally                              | -                                                                                   |
| `extra-path-parameter-case`                         | warn     | Path template variables must use the configured case style.                                                                    | apilint, Oxide                     | `style`                                                                             |
| `extra-path-segment-case`                           | warn     | Paths must start with a slash, have no trailing slash or empty segments and use one casing.                                    | openapilint, apilint               | `case`                                                                              |
| `extra-path-segment-param-only`                     | warn     | A path segment with a parameter must contain only that parameter.                                                              | IBM                                | -                                                                                   |
| `extra-path-segment-plural`                         | warn     | Literal path segments must be plural nouns.                                                                                    | Redocly, apilint                   | `ignoreLastPathSegment`, `exceptions`, `onlyCollectionSegments`                     |
| `extra-pattern-anchored`                            | warn     | String schema patterns must be anchored with ^ and $.                                                                          | IBM                                | `requireStart`, `requireEnd`                                                        |
| `extra-pattern-length-satisfiable`                  | error    | A pattern must be able to match a string within the schema minLength and maxLength.                                            | specmatic                          | -                                                                                   |
| `extra-pattern-properties-restrictions`             | warn     | patternProperties must have one anchored entry and no additionalProperties.                                                    | IBM                                | -                                                                                   |
| `extra-precondition-headers-on-412`                 | warn     | Operations that return a 412 response must define a conditional request header.                                                | IBM                                | `headers`                                                                           |
| `extra-prefer-token-pagination-over-offset`         | warn     | Paginated list operations should use token-based pagination, rather than offset/limit pagination.                              | IBM                                | -                                                                                   |
| `extra-property-name-collision-after-normalising`   | error    | Property names within a schema must be unique after normalising their case convention.                                         | IBM                                | -                                                                                   |
| `extra-put-body-matches-get-response`               | warn     | The PUT request body schema must equal the GET success response schema of the same path.                                       | openapilint                        | `responseCode`                                                                      |
| `extra-query-array-style-form`                      | warn     | Array query parameters must set style to form explicitly.                                                                      | Zally                              | -                                                                                   |
| `extra-query-param-wire-key-collision`              | warn     | Query parameters that serialise to the same wire key must have compatible schemas.                                             | specmatic                          | -                                                                                   |
| `extra-readonly-and-writeonly-exclusive`            | error    | A schema must not be both readOnly and writeOnly.                                                                              | KaiZen                             | -                                                                                   |
| `extra-ref-pattern-component-location`              | warn     | A $ref must start with the components prefix that matches the kind of object it sits in.                                       | IBM                                | -                                                                                   |
| `extra-ref-sibling-description-not-duplicate`       | info     | A description beside a $ref must not duplicate the description of the referenced schema.                                       | IBM                                | -                                                                                   |
| `extra-request-body-json-media-type`                | info     | Request bodies must offer the application/json media type.                                                                     | ai-api-lint                        | `mediaType`                                                                         |
| `extra-request-body-required-list`                  | info     | A JSON request body schema with properties must list its required properties.                                                  | ai-api-lint                        | `mediaTypes`                                                                        |
| `extra-request-body-schema-has-properties`          | warn     | JSON request body schemas must define properties.                                                                              | ai-api-lint                        | -                                                                                   |
| `extra-request-response-must-be-models`             | error    | JSON request and response body object schemas must define static properties (models, not dictionaries).                        | IBM                                | -                                                                                   |
| `extra-required-parameters-before-optional`         | warn     | Required parameters must be listed before optional parameters.                                                                 | IBM                                | -                                                                                   |
| `extra-required-string-min-length`                  | info     | Required string properties must declare minLength.                                                                             | Redocly                            | `shapedFormats`                                                                     |
| `extra-reserved-header-parameter`                   | warn     | Header parameters must not use the reserved names Accept, Content-Type or Authorization.                                       | specmatic                          | `names`                                                                             |
| `extra-resource-response-consistency`               | warn     | Create and update operations must return the same schema as the GET of the resource.                                           | IBM                                | -                                                                                   |
| `extra-resource-schema-naming-convention`           | warn     | Schema names of resource paths must derive from the canonical schema name.                                                     | IBM                                | -                                                                                   |
| `extra-response-contains-header`                    | warn     | Responses must declare the headers configured for their status code.                                                           | Redocly                            | `headers`                                                                           |
| `extra-response-contains-property`                  | warn     | Configured response codes must define the configured top-level properties in their schema.                                     | Redocly                            | `names`                                                                             |
| `extra-response-request-content-required`           | warn     | Request bodies and responses other than 204 must have a content field.                                                         | IBM                                | `ignoreCodes`                                                                       |
| `extra-response-required-properties`                | warn     | Array and enum properties of response schemas must be listed as required.                                                      | IBM                                | `arrays`, `enums`                                                                   |
| `extra-response-status-code-semantics`              | warn     | Operations must use status codes consistently with their meaning.                                                              | IBM                                | -                                                                                   |
| `extra-schema-description-required`                 | warn     | Schemas must have a description.                                                                                               | IBM                                | `scope`                                                                             |
| `extra-schema-keyword-type-consistency`             | warn     | Schema keywords must match the declared type and lower bounds must not exceed upper bounds.                                    | IBM                                | -                                                                                   |
| `extra-schema-property-description`                 | warn     | Every schema property must have a non-empty description.                                                                       | IBM                                | -                                                                                   |
| `extra-schema-type-format-whitelist`                | warn     | Schema types and formats must come from the allowed lists.                                                                     | IBM                                | `types`, `integerFormats`, `numberFormats`, `stringFormats`                         |
| `extra-schema-type-keyword-mismatch`                | warn     | A schema must not carry items or properties that contradict its type, and properties require type object.                      | Redocly, oas-kit                   | `requireExplicitType`                                                               |
| `extra-security-scopes-declared`                    | error    | Use OAuth2 or bearer security schemes and declare every scope that is used.                                                    | Zally                              | `allowedTypes`                                                                      |
| `extra-security-scopes-defined`                     | error    | Security requirements must use defined schemes and declared scopes.                                                            | IBM, Redocly                       | `reportUnused`                                                                      |
| `extra-server-environment-description`              | error    | Every server description must name the environment of the server.                                                              | Spectral                           | `pattern`, `requireDescription`                                                     |
| `extra-server-variables-consistent`                 | warn     | Server variables must have a default, which must be one of the enum values.                                                    | IBM, RMOA, Redocly, Spectral       | `requireEnum`                                                                       |
| `extra-server-variables-defined`                    | error    | Server URL variables must be defined, and defined variables must be used.                                                      | Redocly, RMOA                      | -                                                                                   |
| `extra-server-x-internal`                           | warn     | Every server object must define x-internal as true or false.                                                                   | Spectral                           | `extension`                                                                         |
| `extra-single-type-plus-null`                       | warn     | A schema type list must not hold more than one type besides null.                                                              | IBM                                | -                                                                                   |
| `extra-single-uri-version-segment`                  | warn     | Paths must not contain more than one version segment.                                                                          | baloise-incubator/Spectral-ruleset | -                                                                                   |
| `extra-ssrf-parameter-name`                         | info     | Parameter names must not suggest a caller-supplied URL.                                                                        | Spectral                           | `pattern`                                                                           |
| `extra-strict-ref-locations`                        | error    | A $ref may only appear in objects that the specification allows to be references.                                              | Redocly                            | -                                                                                   |
| `extra-string-length-and-pattern-required`          | warn     | String schemas must define pattern, minLength and maxLength, and other schemas must not.                                       | IBM                                | `requirePattern`, `requireMinLength`, `requireMaxLength`, `shapedFormats`           |
| `extra-string-length-limits`                        | warn     | minLength must be a positive integer and string lengths must not exceed 4 MiB.                                                 | specmatic                          | `minLengthFloor`, `lengthCeiling`                                                   |
| `extra-success-response-example`                    | warn     | Success response media types should include an example.                                                                        | IBM                                | `mediaTypes`                                                                        |
| `extra-success-response-has-schema`                 | warn     | The first success response of an operation must define a body schema.                                                          | ai-api-lint                        | `ignoredCodes`                                                                      |
| `extra-summary-no-trailing-period`                  | warn     | Operation summaries must not end with a period.                                                                                | IBM, Oxide                         | -                                                                                   |
| `extra-tracing-headers-required`                    | error    | Requests must provide the B3 or W3C tracing headers.                                                                           | baloise-incubator/Spectral-ruleset | `requireOnAllOperations`                                                            |
| `extra-unevaluated-properties-must-be-false`        | warn     | unevaluatedProperties, when set, must be false.                                                                                | IBM                                | -                                                                                   |
| `extra-unique-tag-names`                            | error    | Tag names in the top-level tags list must be unique.                                                                           | RMOA, Spectral, spec-validator     | `caseInsensitive`                                                                   |
| `extra-unreachable-405`                             | warn     | A 405 response must not be declared when all HTTP methods are already declared for the path.                                   | specmatic                          | `methods`                                                                           |
| `extra-unsupported-3-1-keywords`                    | warn     | OpenAPI 3.1 documents must not use the top-level keywords jsonSchemaDialect or webhooks.                                       | IBM                                | `keywords`                                                                          |
| `extra-unused-global-tags`                          | warn     | Every global tag must be used by at least one operation.                                                                       | IBM                                | -                                                                                   |
| `extra-url-field-format`                            | error    | URL fields must hold well formed URLs.                                                                                         | KaiZen                             | -                                                                                   |
| `extra-valid-media-type-syntax`                     | warn     | Request body content keys must be valid media types.                                                                           | RMOA                               | `types`                                                                             |
| `extra-webhook-restrictions`                        | warn     | Webhooks must not define callbacks, and webhook path items and operations must not define servers.                             | Spectral                           | `noCallbacks`, `noServers`                                                          |
| `extra-well-defined-dictionaries`                   | warn     | A schema must be either a model or a dictionary, and dictionaries need a well-defined value type.                              | IBM                                | `requireValueType`, `forbidNestedDictionaries`, `requireObjectDefinition`           |
| `extra-well-known-property-types`                   | warn     | Properties named id, created and modified must have the common types.                                                          | Zally                              | `idProperties`, `idType`, `timestampProperties`, `timestampType`, `timestampFormat` |
| `extra-wire-enum-no-camel-case`                     | warn     | Enum values must not be written in camelCase.                                                                                  | Redocly                            | -                                                                                   |
| `extra-wire-parameter-no-camel-case`                | warn     | Parameter names must not be camelCase.                                                                                         | Redocly                            | `pattern`                                                                           |
| `extra-write-operation-requires-request-body`       | warn     | POST, PUT and PATCH operations must declare a request body.                                                                    | ai-api-lint                        | `methods`                                                                           |
| `extra-write-response-returns-resource-or-location` | info     | POST, PUT and PATCH 2xx responses must return the resource or a Location header.                                               | apilint                            | `ignoreCodes`                                                                       |

## Development

```bash
devbox shell
make build
make test
make lint
make format
make generate
```

See [CONTRIBUTING](CONTRIBUTING.md) for how to add a rule.

## License

[Apache-2.0](LICENSE)

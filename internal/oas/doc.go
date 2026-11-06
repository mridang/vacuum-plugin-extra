// Package oas holds the small, dependency-light helpers that the rules use to
// read an OpenAPI document as raw YAML nodes: safe accessors, JSONPath
// building, local $ref resolution and iterators over operations, parameters and
// schemas.
//
// Nothing in this package reports violations; it only navigates documents. All
// accessors accept nil nodes and return zero values for them, so a check can
// chain lookups without testing every step.
package oas

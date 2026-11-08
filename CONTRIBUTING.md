# Contributing to vacuum-plugin-extra

Thank you for your interest in contributing!

## Before you start

- Open an issue first for non-trivial changes so the approach can be agreed on
  before you invest time writing code.
- For typo fixes, doc improvements, and small bug fixes a PR without an issue
  is fine.

## Development setup

```bash
# Install devbox (https://www.jetify.com/devbox/)
devbox shell          # activates Go 1.27, golangci-lint, dprint, etc.
make build            # build every package
make test             # run unit tests
make lint             # golangci-lint
make format           # dprint + gofmt
make generate         # regenerate ruleset.yaml
```

See the [README](README.md) for how to use the rules with vacuum.

## Adding a rule

Every rule is one file, one test file and one entry in the catalogue.

1. Pick a kebab-case name without the `extra-` prefix, for example
   `operation-summary-max-length`. The framework adds the prefix, so the rule
   identifier and the custom function name become `extra-operation-summary-max-length`.
2. Create `rules/operation_summary_max_length.go` with a constructor named
   after the rule, `OperationSummaryMaxLength() Rule`, returning `New(Spec{...})`.
   Fill in the description, the how-to-fix message, the severity, the `Given`
   JSONPath, the rule type, the options with their defaults and the check.
3. Write the exhaustive doc comment on the constructor. It states what the rule
   checks, why, which linter it comes from, the options and their defaults, a
   bad and a good YAML example, and the known false positives.
4. Create `rules/operation_summary_max_length_test.go` in the `rules_test`
   package. Use `ruletest.Run`, `ruletest.RequirePaths` and `ruletest.RequireClean`
   so the rule runs through the real vacuum engine. Cover at least a failing
   document, a passing document and the edge cases (options, `$ref`, OpenAPI
   3.0 against 3.1, missing fields). A test must fail if the rule is a no-op,
   so always assert the exact paths of the violations.
5. Add the constructor to `All()` in `rules/all.go`, keeping the list ordered by
   identifier.
6. Run `make generate` to regenerate `ruleset.yaml`, and add a row to the table
   in the README.

A rule that checks a naming convention must not invent its own option. Declare
the shared one with `casingOption(subject, defaults...)`, build the check with
`WithCasing`, and compare names through the `casing.Set` it receives (see
`internal/casing`). That gives every casing rule the same `casing` option, the
same values, the same validation error and a message that names the convention.
Add the rule to the table in `rules/casing_test.go` so that every casing value
is proven against it.

Use the helpers in `internal/oas` to read the document instead of walking YAML
nodes by hand. Keep functions small, brace every `if` over multiple lines, and
avoid package-level variables.

## Pull request checklist

- [ ] `make test` passes
- [ ] `make lint` is clean
- [ ] `make format` produces no diff
- [ ] `make generate` produces no diff
- [ ] Commit messages follow [Conventional Commits](https://www.conventionalcommits.org/)
      (`feat:`, `fix:`, `docs:`, `chore:` etc.) — semantic-release uses these to
      determine the next version

## Code style

- Standard `gofmt`/`goimports` formatting (enforced by CI)
- All exported symbols must have a doc comment
- `ruleset.yaml` is generated — do not edit it by hand; run `go generate ./...`
  instead

## License

By contributing, you agree your contributions will be licensed under the
[Apache-2.0 License](LICENSE).

package cli

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"slices"
	"strings"

	"github.com/pb33f/go-yaml"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
)

// bundleExternalRefs returns the document with every $ref to another file
// brought into it, so that the rules, which only follow local pointers, see the
// whole API. A document without file references is returned unchanged.
//
// The references to components stay references (the composed form), so the
// result still reads like the original. Line numbers in the findings refer to
// the bundled text, not to the files on disk.
func bundleExternalRefs(spec []byte, path string) ([]byte, error) {
	if !hasFileRefs(spec) {
		return spec, nil
	}

	absolute, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve spec path: %w", err)
	}

	bundled, err := bundler.BundleBytesComposed(spec, &datamodel.DocumentConfiguration{
		BasePath:            filepath.Dir(absolute),
		SpecFilePath:        absolute,
		AllowFileReferences: true,
		Logger:              slog.New(slog.DiscardHandler),
	}, &bundler.BundleCompositionConfig{Delimiter: "__"})
	if err != nil {
		return nil, fmt.Errorf("bundle file references: %w", err)
	}

	slog.Default().Debug("bundled file references", "spec", path, "before", len(spec), "after", len(bundled))

	return bundled, nil
}

// hasFileRefs reports whether any $ref of the document points outside it and
// is not an http or https URL.
func hasFileRefs(spec []byte) bool {
	var root yaml.Node
	if err := yaml.Unmarshal(spec, &root); err != nil {
		return false
	}

	return anyFileRef(&root)
}

func anyFileRef(node *yaml.Node) bool {
	if node == nil {
		return false
	}

	if node.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(node.Content); i += 2 {
			if node.Content[i].Value == "$ref" && isFileRef(node.Content[i+1].Value) {
				return true
			}
		}
	}

	return slices.ContainsFunc(node.Content, anyFileRef)
}

func isFileRef(ref string) bool {
	return ref != "" && !strings.HasPrefix(ref, "#") &&
		!strings.HasPrefix(ref, "http://") && !strings.HasPrefix(ref, "https://")
}

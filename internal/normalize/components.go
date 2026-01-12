package normalize

import (
	"fmt"
	"sort"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

func collectComponentSchemas(doc *openapi3.T) (map[string]ir.TypeDecl, error) {
	out := map[string]ir.TypeDecl{}

	if doc == nil {
		return nil, fmt.Errorf("nil OpenAPI doc")
	}
	if doc.Components.Schemas == nil {
		return out, nil // allowed: empty
	}

	// deterministic iteration
	names := make([]string, 0, len(doc.Components.Schemas))
	for name := range doc.Components.Schemas {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			return nil, fmt.Errorf("components.schemas contains empty name")
		}

		sr := doc.Components.Schemas[name]
		if sr == nil {
			return nil, fmt.Errorf("components.schemas.%s is nil", name)
		}

		// accept either inline schema or $ref, but in practice components.schemas should be inline.
		// if it's a $ref, resolve it through SchemaRefToTypeRef.
		if sr.Ref != "" {
			_, err := SchemaRefToTypeRef(sr)
			if err != nil {
				return nil, fmt.Errorf("components.schemas.%s: %w", name, err)
			}
			// keep it strict: components.schemas entries must be inline (Value != nil).
			return nil, fmt.Errorf("components.schemas.%s: $ref schema entries are not supported; define schema inline", name)

		}

		if sr.Value == nil {
			return nil, fmt.Errorf("components.schemas.%s has no schema value", name)
		}

		t, err := schemaValueToType(sr.Value)
		if err != nil {
			return nil, fmt.Errorf("components.schemas.%s: %w", name, err)
		}

		out[name] = ir.TypeDecl{
			Name: name,
			Type: t,
		}
	}

	return out, nil
}

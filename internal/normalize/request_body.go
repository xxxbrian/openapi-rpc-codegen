package normalize

import (
	"fmt"
	"strings"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

func normalizeRequestBody(op *openapi3.Operation) (*ir.Body, error) {
	if op.RequestBody == nil || op.RequestBody.Value == nil {
		return nil, nil // body optional for POST
	}

	mt := "application/json"
	rb := op.RequestBody.Value
	if rb.Content == nil || rb.Content[mt] == nil {
		alt, schema := findJSONContent(rb.Content)
		if schema == nil {
			return nil, fmt.Errorf("requestBody must have %q content", mt)
		}
		typ, err := SchemaRefToTypeRef(schema)
		if err != nil {
			return nil, fmt.Errorf("requestBody %q schema: %w", alt, err)
		}
		return &ir.Body{Required: rb.Required, Type: typ}, nil
	}

	if rb.Content[mt].Schema == nil {
		return nil, fmt.Errorf("requestBody %q must define schema", mt)
	}

	typ, err := SchemaRefToTypeRef(rb.Content[mt].Schema)
	if err != nil {
		return nil, fmt.Errorf("requestBody %q schema: %w", mt, err)
	}

	// Optional sanity: reject empty schema
	if strings.TrimSpace(typ.RefName) == "" && typ.Inline == nil {
		return nil, fmt.Errorf("requestBody schema is empty/unsupported")
	}

	return &ir.Body{
		Required: rb.Required,
		Type:     typ,
	}, nil
}

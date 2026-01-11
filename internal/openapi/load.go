package openapi

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
)

func LoadAndValidate(specPath string) (*openapi3.T, error) {
	loader := openapi3.NewLoader()
	loader.IsExternalRefsAllowed = true

	doc, err := loader.LoadFromFile(specPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load OpenAPI spec: %w", err)
	}

	if err := doc.Validate(loader.Context); err != nil {
		return nil, fmt.Errorf("OpenAPI validation error: %w", err)
	}

	if len(doc.Servers) == 0 || doc.Servers[0] == nil || doc.Servers[0].URL == "" {
		return nil, fmt.Errorf("OpenAPI spec must define servers[0].url")
	}

	return doc, nil
}

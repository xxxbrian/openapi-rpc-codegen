package normalize

import (
	"fmt"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type Options struct {
	BaseURLOverride string
}

func ToIR(doc *openapi3.T, opt Options) (*ir.Spec, error) {
	if doc == nil {
		return nil, fmt.Errorf("nil OpenAPI doc")
	}
	baseURL := doc.Servers[0].URL
	if opt.BaseURLOverride != "" {
		baseURL = opt.BaseURLOverride
	}

	// TODO: implement rules here:
	// - Only GET/POST
	// - Must have operationId unique
	// - Must have responses["200"].content["application/json"].schema
	// - No oneOf/anyOf/allOf/additionalProperties, etc.

	return &ir.Spec{
		Meta:  ir.Meta{BaseURL: baseURL},
		Types: map[string]ir.TypeDecl{},
		// TODO: Routes
		Routes: []ir.Route{},
	}, nil
}

package normalize

import (
	"errors"

	"github.com/getkin/kin-openapi/openapi3"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type Options struct {
	BaseURLOverride string
}

func ToIR(doc *openapi3.T, opt Options) (*ir.Spec, error) {
	// TODO
	return nil, errors.ErrUnsupported
}

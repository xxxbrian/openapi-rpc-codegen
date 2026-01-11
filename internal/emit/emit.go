package emit

import (
	"errors"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/ir"
)

type Options struct {
	OutDir  string
	Targets []string
	Check   bool
	Verbose bool
}

func Dispatch(spec *ir.Spec, opts Options) ([]string, error) {
	// TODO
	return nil, errors.ErrUnsupported
}

package codegen

import (
	"fmt"

	"github.com/xxxbrian/openapi-rpc-codegen/internal/emit"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/normalize"
	"github.com/xxxbrian/openapi-rpc-codegen/internal/openapi"
)

func Generate(opts Options) (*Result, error) {
	if opts.SpecPath == "" {
		return nil, fmt.Errorf("spec path is required")
	}
	if opts.OutDir == "" {
		return nil, fmt.Errorf("output directory is required")
	}

	doc, err := openapi.LoadAndValidate(opts.SpecPath)
	if err != nil {
		return nil, err
	}

	irSpec, err := normalize.ToIR(doc, normalize.Options{
		BaseURLOverride: opts.BaseURL,
	})

	if err != nil {
		return nil, err
	}

	files, err := emit.Dispatch(irSpec, emit.Options{
		OutDir:  opts.OutDir,
		Targets: opts.Targets,
		Check:   opts.Check,
		Verbose: opts.Verbose,
	})
	if err != nil {
		return nil, err
	}

	return &Result{Files: files}, nil
}

package codegen

type Options struct {
	SpecPath string
	OutDir   string
	BaseURL  string
	Targets  []string
	Check    bool
	Verbose  bool
}

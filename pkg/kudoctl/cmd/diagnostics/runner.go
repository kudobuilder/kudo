package diagnostics

// collector - generic interface for diagnostic data collection
// implementors are expected to return only fatal errors and handle non-fatal ones themselves
type collector interface {
	collect() error
}

// runner - sequential runner for Collectors reducing error checking boilerplate code
type runner struct {
	fatalErr error
}

func (r *runner) run(c collector) *runner {
	if r.fatalErr == nil {
		r.fatalErr = c.collect()
	}
	return r
}

func (r *runner) dumpToYaml(v interface{}, dir stringGetter, name string, p *nonFailingPrinter) *runner {
	p.printYaml(v, dir(), name)
	return r
}

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

func (r *runner) runForEach(names []string, fn func(string) collector) *runner {
	for _, name := range names {
		collector := fn(name)
		r.run(collector)
	}
	return r
}

func (r *runner) dumpToYaml(v interface{}, dir stringGetter, name string, p *nonFailingPrinter) *runner {
	p.printYaml(v, dir(), name)
	return r
}

type collectorForRunner struct {
	runnerFn func() *runner
}

func (c *collectorForRunner) collect() error {
	r := c.runnerFn()
	return r.fatalErr
}

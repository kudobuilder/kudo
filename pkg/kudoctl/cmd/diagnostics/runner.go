package diagnostics

type Runner struct {
	fatalErr error
}

func (r *Runner) Run(c Collector) *Runner {
	if r.fatalErr == nil {
		r.fatalErr = c.Collect()
	}
	return r
}

func (r *Runner) RunForEach(names []string, fn func(string) Collector) *Runner {
	for _, name := range names {
		collector := fn(name)
		r.Run(collector)
	}
	return r
}

func (r *Runner) DumpToYaml(v interface{}, dir stringGetter, name string, p *ObjectPrinter) *Runner {
	p.printYaml(v, dir(), name)
	return r
}

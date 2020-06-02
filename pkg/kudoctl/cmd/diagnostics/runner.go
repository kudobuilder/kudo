package diagnostics

// collector - generic interface for diagnostic data collection
// implementors are expected to return only fatal errors and handle non-fatal ones themselves
type collector interface {
	collect(printer *nonFailingPrinter) error
}

// runner - sequential runner for Collectors reducing error checking boilerplate code
type runner struct {
	collectors []collector
}

func (r *runner) addCollector(c collector) {
	r.collectors = append(r.collectors, c)
}

func (r *runner) addObjDump(v interface{}, dir stringGetter, name string) {
	r.addCollector(&objCollector{
		obj:       v,
		parentDir: dir,
		name:      name,
	})
}

func (r *runner) run(printer *nonFailingPrinter) error {
	for _, c := range r.collectors {
		if err := c.collect(printer); err != nil {
			return err
		}
	}
	return nil
}

package diagnostics

import (
	"github.com/ghodss/yaml"
)

// helper collector to print any object as yaml
type dumpingCollector struct {
	s interface{}
}

func (c *dumpingCollector) Collect(f writerFactory) error {
	w, err := f(c.s)
	if err != nil {
		return err
	}
	b, err := yaml.Marshal(c.s)
	if err != nil {
		return err
	}
	_, err = w.Write(b)
	if err != nil {
		return err
	}
	return nil
}

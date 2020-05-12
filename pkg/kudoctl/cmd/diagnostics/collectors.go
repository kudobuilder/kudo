package diagnostics

type ResourceCollector struct {
	r          *ResourceFuncsConfig
	resourceFn ResourceFn
	printMode  printMode
	parentDir  func() string
	errName    string
	failOnErr  bool
}

func (c *ResourceCollector) Collect() (Printable, error) {
	obj, err := c.resourceFn(c.r)
	if err != nil {
		if c.failOnErr {
			return nil, err
		}
		return &PrintableError{
			error: err,
			Fatal: false,
			name:  c.errName,
			dir:   c.parentDir,
		}, nil
	}
	switch c.printMode {
	case ObjectWithDir:
		return NewPrintableObject(obj, c.parentDir)
	case ObjectsWithDir:
		return NewPrintableObjectList(obj, c.parentDir)
	case RuntimeObject:
		fallthrough
	default:
		return NewPrintableRuntimeObject(obj, c.parentDir)
	}
}

type LogCollector struct {
	r         *ResourceFuncsConfig
	podNames  func() []string
	parentDir func() string
}

func (c *LogCollector) Collect() (Printable, error) {
	var ret PrintableList
	for _, podName := range c.podNames() {
		log, err := Log(c.r, podName)
		if err != nil {
			return &PrintableError{
				error: err,
				Fatal: false,
				name:  podName,
				dir:   c.parentDir,
			}, nil
		}
		ret = append(ret, &PrintableLog{
			name:      podName,
			log:       log,
			parentDir: c.parentDir,
		})
	}
	return ret, nil
}

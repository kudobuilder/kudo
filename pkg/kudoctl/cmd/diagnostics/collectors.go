package diagnostics

type ResourceCollector struct {
	r          *ResourceFuncsConfig
	resourceFn ResourceFn
	printMode  printMode
	baseDir    func() string
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
			dir:   c.baseDir,
		}, nil
	}
	switch c.printMode {
	case ObjectWithDir:
		return NewPrintableObject(obj, c.baseDir)
	case ObjectsWithDir:
		return NewPrintableObjectList(obj, c.baseDir)
	case RuntimeObject:
		fallthrough
	default:
		return NewPrintableRuntimeObject(obj, c.baseDir)
	}
}

type LogCollector struct {
	r        *ResourceFuncsConfig
	podNames func() []string
	baseDir  func() string
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
				dir:   c.baseDir,
			}, nil
		}
		ret = append(ret, &PrintableLog{
			name:      podName,
			log:       log,
			parentDir: c.baseDir,
		})
	}
	return ret, nil
}

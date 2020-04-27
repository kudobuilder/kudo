package diagnostics

type Builder struct {
	collectors []Collector
	ctx map[string][]NameHolder
}

func NewBuilder() *Builder {
	return &Builder{ctx: make(map[string][]NameHolder)}
}

func (b *Builder) nameStorer(key string, f func(*ObjectWithParent) NameHolder) func (*ObjectWithParent) {
	return func(o *ObjectWithParent) {
		b.ctx[key] = append(b.ctx[key], f(o))
	}
}

func (b *Builder) namesStorer(key string, f func([]ObjectWithParent) []NameHolder) func (o []ObjectWithParent) {
	return func(o []ObjectWithParent) {
		b.ctx[key] = append(b.ctx[key], f(o)...)
	}
}

type nameExtractorWithKey struct {
	key string
	extractor func(parent *ObjectWithParent) NameHolder
}

type namesExtractorWithKey struct {
	key string
	extractor func(parent []ObjectWithParent) []NameHolder
}

func (b *Builder) AddResource(getter objectGetter, nameExtractors... nameExtractorWithKey ) *Builder {
	for _, e := range nameExtractors {
		getter = getterWithCallback(getter, b.nameStorer(e.key, e.extractor ))
	}
	b.collectors = append(b.collectors, &resourceCollector{getResource: getter})
	return b
}

func (b *Builder) AddResources(lister objectLister, namesExtractors... namesExtractorWithKey ) *Builder {
	for _, e := range namesExtractors {
		lister = listerWithCallback(lister, b.namesStorer(e.key, e.extractor ))
	}
	b.collectors = append(b.collectors, &resourceListCollector{getResources: lister})
	return b
}

func (b *Builder) GetNames(key string) NameLister {
	return func() []NameHolder {
		return b.ctx[key]
	}
}

func (b *Builder) GetName(key string) NameGetter {
	return func() *NameHolder {
		if nhs := b.ctx[key]; len(nhs) > 0 {
			return &nhs[0]
		}
		return nil
	}
}

func (b *Builder) Build() []Collector{
	return b.collectors
}

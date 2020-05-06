package diagnostics

import (
	"io"
	"strings"

	"k8s.io/cli-runtime/pkg/printers"
)

// TODO: rename callbacks
// builder should build a sequence of resource providers based on the AddResource
// it should add missing dependencies based on the provided name extraction rules
// probably extraction rules should make sure there is no loop, not the builder
type otherBuilder struct {
	n *nameProviders
	factory *resourceProviderFactory
	// one day more sophisticated data may be needed than just a resource names list
	namesCache  map[string][]string // kind: <resource names>
	isCollected map[string]struct{}
	isExposed   map[string]struct{}
	callbacks   map[string]map[string]nameExtractorFn // name provider -> collected resources
	collectors  []objectLister
}

func NewOtherBuilder(factory *resourceProviderFactory, providers *nameProviders) *otherBuilder {
	return &otherBuilder{
		n:           providers,
		factory:     factory,
		namesCache:  make(map[string][]string),
		isCollected: make(map[string]struct{}),
		isExposed:   make(map[string]struct{}),
		callbacks:   make(map[string]map[string]nameExtractorFn),
	}
}

func (b *otherBuilder) AddResource(kind string) {
	b.addResource(kind)
	b.isExposed[kind] = struct{}{}
}

func (b *otherBuilder) addResource(kind string) {
	if _, ok := b.isCollected[kind]; ok {
		return
	}
	if fromKind, extractorFn := b.n.NameProviderFor(kind); fromKind != "" {
		b.addResource(fromKind)
		if b.callbacks[fromKind] == nil {
			b.callbacks[fromKind] = make(map[string]nameExtractorFn)
		}
		b.callbacks[fromKind][kind] = extractorFn // collector p should cache names for kind using rule from nameProviders
		b.addMultiGetter(kind)
	} else {
		b.addLister(kind)
	}
	b.isCollected[kind] = struct{}{}
}

func (b *otherBuilder) addMultiGetter(kind string) {
	f := func() ([]Object, error) {
		names, _ := b.namesCache[kind] // TODO: handle not found
		getterFn := b.factory.MultiGetter(kind, names)
		ret, _ := getterFn() // TODO: handle error
		return ret, nil
	}
	b.collectors = append(b.collectors, f)
}

func (b *otherBuilder) addLister(kind string) {
	lister := b.factory.Lister(kind)
	b.collectors = append(b.collectors, lister)
}

func (b *otherBuilder) Collect() *ResourceCache {
	resourceListsByKind := make(map[string][]Object)
	resourcesByNameKind := make(map[NameKind]Object)
	for _, collectFn := range b.collectors {
		objs, err := collectFn()
		if err != nil { //TODO: handle error
			continue
		}
		if len(objs) == 0 {
			continue
		}
		kind := objs[0].GetObjectKind().GroupVersionKind().Kind //TODO: fix this ugly hack
		kind = strings.ToLower(kind)
		resourceListsByKind[kind] = objs
		for _, obj := range objs {
			resourcesByNameKind[NameKind{
				Name: obj.GetName(),
				Kind: kind,
			}] = obj
		}
		for dependentKind, callback := range b.callbacks[kind] {
			isNameAdded:= make(map[string]struct{})
			for _, o := range objs {
				name := callback(o)
				if _, ok := isNameAdded[name]; ok {
					continue
				}
				isNameAdded[name] = struct{}{}
				b.namesCache[dependentKind] = append(b.namesCache[dependentKind], name)
			}
		}

	}
	return &ResourceCache{
		resourceListsByKind: resourceListsByKind,
		resourcesByNameKind: resourcesByNameKind,
	}
}
// The ultimate provider of the resource
// we should be able to build the printable directory tree based on the resources and config
type ResourceCache struct {
	resourceListsByKind map[string][]Object
	resourcesByNameKind map[NameKind]Object
}

func (r *ResourceCache) Resources(kind string) []Object {
	return r.resourceListsByKind[kind]
}
func (r *ResourceCache) Resource(name, kind string) Object {
	return r.resourcesByNameKind[NameKind{name, kind}]
}

func BuildPrintableTree (r *ResourceCache, a *attachmentRules, kinds []string) []*printableTree {
	m := make(map[NameKind]*printableTree)
	var ret []*printableTree
	for _, kind := range kinds {
		if _, ok :=a.m[kind]; !ok {
			rootResources := r.resourceListsByKind[kind]
			for _, resource := range rootResources {
				nk := NameKind{
					Name: resource.GetName(),
					Kind: kind,
				}
				ret = append(ret, getOrCreate(nk, m, r))
			}
			continue
		}
		resources := r.Resources(kind)
		for _, resource := range resources {
			nk := a.AttachmentFor(resource) // TODO: consider attaching to root if nil
			if nk == nil {
				continue
			}
			parent := getOrCreate(*nk, m, r) // TODO: handle parent not collected error!
			key := NameKind{
				Name: resource.GetName(),
				Kind: strings.ToLower(resource.GetObjectKind().GroupVersionKind().Kind),
			}
			child := getOrCreate(key, m, r)
			parent.children = append(parent.children, child)
		}
	}
	return ret
}

func getOrCreate(key NameKind, m map[NameKind]*printableTree, r *ResourceCache) *printableTree {
	pt, ok := m[key]
	if !ok {
		resource := r.Resource(key.Name, key.Kind) // TODO: what if missing?
		pt = &printableTree{o: toPrintableObject(resource)}
		m[key] = pt
	}
	return pt
}

type printableObject struct {
	Object
	printer printers.YAMLPrinter // TODO: allow other printers
}

func (o *printableObject) Print(w io.Writer) error {
	return o.printer.PrintObj(o, w)
}

func toPrintableObject(o Object) *printableObject {
	return &printableObject{o, printers.YAMLPrinter{}}
}
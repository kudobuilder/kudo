package diagnostics

// configuration of the output hierarchical directory structure

import (
	corev1 "k8s.io/api/core/v1"
	"strings"
)

type NameKind struct {
	Name, Kind string
}

type attachmentFn func (Object) *NameKind

// configuration. essentially, a directory tree definition
type attachmentRules struct {
	rootFn       attachmentFn // TODO: a hack allowing to attach to a certain "root". fix properly.
	m            map[string]attachmentFn
}

func (a *attachmentRules) Add(kind string, rule attachmentFn) *attachmentRules {
	a.m[kind] = rule
	return a
}

// TODO: handle missing
func (a *attachmentRules) AttachmentFor(o Object) *NameKind {
	kind := strings.ToLower(o.GetObjectKind().GroupVersionKind().Kind)
	if fn := a.m[kind]; fn != nil {
		return fn(o)
	}
	return nil
}

func podOwner(o Object) *NameKind {
	obj := o.(*corev1.Pod)
	owners := obj.GetObjectMeta().GetOwnerReferences()
	for _, owner := range owners {
		kind := strings.ToLower(owner.Kind)
		if kind == "statefulset" || kind == "replicaset"{
			return &NameKind{
				Name: strings.ToLower(owner.Name),
				Kind: strings.ToLower(owner.Kind),
			}
		}
	}
	return nil
}

func (a *attachmentRules) setLocalRoot(fn attachmentFn) attachmentFn {
	return func(o Object) *NameKind {
		ret := fn(o)
		a.rootFn = func(o1 Object) *NameKind {
			return ret
		}
		return ret
	}
}

func nameKindFor(fn nameExtractorFn, kind string) attachmentFn {
	return func(o Object) *NameKind{
		return &NameKind{
			Name: fn(o),
			Kind: kind,
		}
	}
}

func DefaultAttachmentRules() *attachmentRules {
	a := &attachmentRules{m: make(map[string]attachmentFn)}
	a.
		Add("operatorversion", nameKindFor(operatorForOperatorVersion, "operator")). // OV child for O
		Add("instance", a.setLocalRoot(nameKindFor(operatorVersionForInstance, "operatorversion"))). // I child for OV
		Add("pod", podOwner) // TODO: add non-owned pods
	for _, kind := range []string{"statefulset", "replicaset", "event", "deployment", "service", "serviceaccount", "role", "clusterrole"} {
		a.Add(kind, func(o Object) *NameKind {return a.rootFn(nil)})
	}
	return a
}

func DefaultKudoAttachmentRules() *attachmentRules {
	return (&attachmentRules{m: make(map[string]attachmentFn)}).Add("pod", podOwner)
}
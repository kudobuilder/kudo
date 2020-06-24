package renderer

import (
	"strings"

	"k8s.io/apimachinery/pkg/runtime/schema"
)

type ResourcePath struct {
	Group   string
	Version string
	Kind    string
	path    string
}

func (lp *ResourcePath) pathFields() []string {
	return strings.Split(lp.path, "/")
}

func (lp *ResourcePath) matches(gvk schema.GroupVersionKind) bool {
	if lp.Group != "" && lp.Group != gvk.Group {
		return false
	}
	if lp.Version != "" && lp.Version != gvk.Version {
		return false
	}
	if lp.Kind != "" && lp.Kind != gvk.Kind {
		return false
	}
	return true
}

var (
	// CommonLabelPaths is a list of locations in an object type where labels should be added.
	// Taken from here: https://github.com/kubernetes-sigs/kustomize/blob/master/api/konfig/builtinpluginconsts/commonlabels.go
	CommonLabelPaths = []ResourcePath{
		{Group: "", Kind: "", path: "metadata/labels"},
		{Group: "apps", Kind: "StatefulSet", path: "spec/template/metadata/labels"},
		{Group: "apps", Kind: "StatefulSet", path: "spec/volumeClaimTemplates[]/metadata/labels"},
		{Group: "apps", Kind: "Deployment", path: "spec/template/metadata/labels"},
		{Group: "", Kind: "ReplicaSet", path: "spec/template/metadata/labels"},
		{Group: "", Kind: "ReplicationController", path: "spec/template/metadata/labels"},
		{Group: "", Kind: "DaemonSet", path: "spec/template/metadata/labels"},
		{Group: "batch", Kind: "Job", path: "spec/template/metadata/labels"},
		{Group: "batch", Kind: "CronJob", path: "spec/template/metadata/labels"},
		{Group: "batch", Kind: "CronJob", path: "spec/jobTemplate/spec/template/metadata/labels"},
	}

	// CommonAnnotationPaths is a list of locations for annotations to add in all resources
	CommonAnnotationPaths = []ResourcePath{
		{Group: "", Kind: "", path: "metadata/annotations"},
	}

	// TemplateAnnotationPaths is a list of locations specific to objects where annotations should be added in
	// templates. Taken from here https://github.com/kubernetes-sigs/kustomize/blob/master/api/konfig/builtinpluginconsts/commonannotations.go
	TemplateAnnotationPaths = []ResourcePath{
		{Group: "", Kind: "ReplicationController", path: "spec/template/metadata/annotations"},
		{Group: "apps", Kind: "StatefulSet", path: "spec/template/metadata/annotations"},
		{Group: "apps", Kind: "Deployment", path: "spec/template/metadata/annotations"},
		{Group: "", Kind: "ReplicaSet", path: "spec/template/metadata/annotations"},
		{Group: "", Kind: "DaemonSet", path: "spec/template/metadata/annotations"},
		{Group: "batch", Kind: "Job", path: "spec/template/metadata/annotations"},
		{Group: "batch", Kind: "CronJob", path: "spec/template/metadata/annotations"},
		{Group: "batch", Kind: "CronJob", path: "spec/jobTemplate/spec/template/metadata/annotations"},
	}
)

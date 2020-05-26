package diagnostics

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/runtime"
)

// processingContext - shared data for the resource collectors
// provides property accessors allowing to define a collector before the data it needs is available
// provides update callback functions. callbacks panic if called on a wrong type of runtime.object
type processingContext struct {
	podNames      []string
	root          string
	opName        string
	opVersionName string
	instanceName  string
}

func (ctx *processingContext) rootDirectory() string {
	return ctx.root
}

func (ctx *processingContext) operatorDirectory() string {
	return fmt.Sprintf("%s/operator_%s", ctx.root, ctx.opName)
}

func (ctx *processingContext) instanceDirectory() string {
	return fmt.Sprintf("%s/instance_%s", ctx.operatorDirectory(), ctx.instanceName)
}

func (ctx *processingContext) mustSetOperatorNameFromOperatorVersion(obj runtime.Object) {
	ctx.opName = obj.(*v1beta1.OperatorVersion).Spec.Operator.Name
}

func (ctx *processingContext) mustSetOperatorVersionNameFromInstance(obj runtime.Object) {
	ctx.opVersionName = obj.(*v1beta1.Instance).Spec.OperatorVersion.Name
}

func (ctx *processingContext) mustAddPodNames(obj runtime.Object) {
	_ = meta.EachListItem(obj, func(obj runtime.Object) error {
		ctx.podNames = append(ctx.podNames, obj.(*v1.Pod).Name)
		return nil
	})
}

func (ctx *processingContext) operatorVersionName() string {
	return ctx.opVersionName
}

func (ctx *processingContext) operatorName() string {
	return ctx.opName
}

package diagnostics

import (
	"fmt"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// processingContext - shared data for the resource collectors
// provides property accessors allowing to define a collector before the data it needs is available
// provides update callback functions. callbacks panic if called on a wrong type of runtime.object
type processingContext struct {
	root          string
	opName        string
	opVersionName string
	instanceName  string
	pods          []v1.Pod
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

func (ctx *processingContext) mustSetPods(o runtime.Object) {
	ctx.pods = o.(*v1.PodList).Items
}

func (ctx *processingContext) operatorVersionName() string {
	return ctx.opVersionName
}

func (ctx *processingContext) operatorName() string {
	return ctx.opName
}

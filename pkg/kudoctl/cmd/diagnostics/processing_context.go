package diagnostics

import (
	"fmt"

	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
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

func (ctx *processingContext) setOperatorNameFromOperatorVersion(obj runtime.Object) {
	ctx.opName = obj.(*kudoapi.OperatorVersion).Spec.Operator.Name
}

func (ctx *processingContext) setOperatorVersionNameFromInstance(obj runtime.Object) {
	ctx.opVersionName = obj.(*kudoapi.Instance).Spec.OperatorVersion.Name
}

func (ctx *processingContext) setPods(o runtime.Object) {
	ctx.pods = o.(*v1.PodList).Items
}

func (ctx *processingContext) operatorVersionName() string {
	return ctx.opVersionName
}

func (ctx *processingContext) operatorName() string {
	return ctx.opName
}

func (ctx *processingContext) podList() []v1.Pod {
	return ctx.pods
}

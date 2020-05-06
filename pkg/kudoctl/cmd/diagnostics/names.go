package diagnostics

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

/* helpers to get names for resources to be collected based on other collected resources */

type nameExtractorFn func (Object) string

func serviceAccountsForPods(obj Object) string {
	pod := obj.(*corev1.Pod)
	return pod.Spec.ServiceAccountName
}

func operatorForOperatorVersion(obj Object) string {
	ov := obj.(*v1beta1.OperatorVersion)
	return ov.Spec.Operator.Name
}

func operatorVersionForInstance(obj Object) string {
	i := obj.(*v1beta1.Instance)
	return i.Spec.OperatorVersion.Name
}

func clusterRoleForBinding (obj Object) string {
	binding := obj.(*rbacv1.ClusterRoleBinding)
	return binding.RoleRef.Name
}

func roleForBinding(obj Object) string {
	binding := obj.(*rbacv1.RoleBinding)
	return binding.RoleRef.Name
}

type nameExtractorHolder struct{
	p string
	extractor nameExtractorFn
}

type nameProviders struct {
	m map[string]nameExtractorHolder
}

// Add rule how to obtain names for resource kind2 from resource kind1
// TODO: expose holder, pass holder - otherwise it's easy to confuse kinds' order
func (p *nameProviders) Add(kind, namesProviderKind string, rule nameExtractorFn) *nameProviders {
	p.m[kind] = nameExtractorHolder{
		p:         namesProviderKind,
		extractor: rule,
	}
	return p
}

func (p *nameProviders) NameProviderFor(kind string) (string, nameExtractorFn) {
	h := p.m[kind]
	return h.p, h.extractor
}

func DefaultNameProviders() *nameProviders {
	return (&nameProviders{make(map[string]nameExtractorHolder)}).
		Add("serviceaccount", "pod", serviceAccountsForPods).
		Add("operator", "operatorversion", operatorForOperatorVersion).
		Add("operatorversion", "instance", operatorVersionForInstance).
		Add("clusterrole", "clusterrolebinding", clusterRoleForBinding).
		Add("role", "rolebinding", roleForBinding).
		Add("clusterrolebinding", "pod", serviceAccountsForPods).
		Add("rolebinding", "pod", serviceAccountsForPods)
}
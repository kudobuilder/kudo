package diagnostics

import (
	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
)

/* helpers to get names for resources to be collected based on other collected resources */

func nameDumper(o *NameHolder) NameGetter {
	return func() *NameHolder {
		return o
	}
}

func serviceAccountsForPods(objs []ObjectWithParent) []NameHolder {
	var ret []NameHolder
	for i := range objs {
		obj := &objs[i]
		pod := obj.Object.(*corev1.Pod)
		ret = append(ret, NameHolder{
			name:   pod.Spec.ServiceAccountName,
			parent: obj,
		})
	}
	return ret
}

func operatorForOperatorVersion(o *ObjectWithParent) NameHolder {
	ov := o.Object.(*v1beta1.OperatorVersion)
	return NameHolder{
		name: ov.Spec.Operator.Name,
		parent: o.parent, // keep Operator and OperatorVersion on the same level
	}
}

func operatorVersionForInstance(o *ObjectWithParent) NameHolder {
	i := o.Object.(*v1beta1.Instance)
	return NameHolder{
		name: i.Spec.OperatorVersion.Name,
		parent: o,
	}
}

func clusterRolesForBindings (objs []ObjectWithParent) []NameHolder {
		var ret []NameHolder
		for i := range objs {
			obj := &objs[i]
			binding := obj.Object.(*rbacv1.ClusterRoleBinding)
			ret = append(ret, NameHolder{
				name:   binding.RoleRef.Name,
				parent: obj,
			})
		}
		return ret
}

func rolesForBindings(objs []ObjectWithParent) []NameHolder {
	var ret []NameHolder
	for i := range objs {
		obj := &objs[i]
		binding := obj.Object.(*rbacv1.RoleBinding)
		ret = append(ret, NameHolder{
			name:   binding.RoleRef.Name,
			parent: obj,
		})
	}
	return ret
}

func pods (objs []ObjectWithParent) []NameHolder {
	var ret []NameHolder
	for i := range objs {
		obj := &objs[i]
		pod := obj.Object.(*corev1.Pod)
		ret = append(ret, NameHolder{
			name:   pod.Name,
			parent: obj.parent,
		})
	}
	return ret
}
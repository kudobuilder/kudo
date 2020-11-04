package renderer

import (
	"testing"

	"github.com/kudobuilder/kuttl/pkg/test/utils"
	"github.com/stretchr/testify/assert"
	"github.com/thoas/go-funk"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/batch/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/kubectl/pkg/scheme"
	clientfake "sigs.k8s.io/controller-runtime/pkg/client/fake"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/test/fake"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func TestEnhancerApply_embeddedMetadataStatefulSet(t *testing.T) {

	tpls := []runtime.Object{
		statefulSet("sfs1", "default"),
	}

	meta := metadata()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: fake.CachedDiscoveryClient(),
	}

	objs, err := e.Apply(tpls, meta)
	if err != nil {
		t.Errorf("failed to apply template %s", err)
	}

	for _, o := range objs {
		unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			t.Errorf("failed to parse object to unstructured: %s", err)
		}
		sfs := &appsv1.StatefulSet{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructMap, sfs); err != nil {
			t.Errorf("failed to parse unstructured to StatefulSet: %s", err)
		}

		// Verify that labels are added
		assert.Equal(t, meta.InstanceNamespace, sfs.GetNamespace())
		assert.Equal(t, string(meta.PlanUID), sfs.Annotations[kudo.PlanUIDAnnotation])

		// Verify that annotations are added
		assert.Equal(t, "kudo", sfs.Labels[kudo.HeritageLabel])
		assert.Equal(t, "kudo", sfs.Spec.Template.Labels[kudo.HeritageLabel])
		assert.Equal(t, "kudo", sfs.Spec.VolumeClaimTemplates[0].Labels[kudo.HeritageLabel])
		assert.Equal(t, "kudo", sfs.Spec.VolumeClaimTemplates[1].Labels[kudo.HeritageLabel])

		// Verify that existing labels are not removed
		assert.Equal(t, "app-type", sfs.Spec.Template.Labels["app"])
		assert.Equal(t, "vct1label", sfs.Spec.VolumeClaimTemplates[0].Labels["vct1"])
		assert.Equal(t, "vct2label", sfs.Spec.VolumeClaimTemplates[1].Labels["vct2"])
	}
}

func TestEnhancerApply_embeddedMetadataCronjob(t *testing.T) {

	tpls := []runtime.Object{
		cronjob("cronjob", "default"),
	}

	meta := metadata()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: fake.CachedDiscoveryClient(),
	}

	objs, err := e.Apply(tpls, meta)
	if err != nil {
		t.Errorf("failed to apply template %s", err)
	}

	for _, o := range objs {
		unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			t.Errorf("failed to parse object to unstructured: %s", err)
		}
		cron := &v1beta1.CronJob{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructMap, cron); err != nil {
			t.Errorf("failed to parse unstructured to CronJob: %s", err)
		}

		// Verify that labels are added directly and on template
		assert.Equal(t, meta.InstanceNamespace, cron.GetNamespace())
		assert.Equal(t, string(meta.PlanUID), cron.Annotations[kudo.PlanUIDAnnotation])
		assert.Equal(t, "kudo", cron.Labels[kudo.HeritageLabel])

		assert.Equal(t, "kudo", cron.Spec.JobTemplate.Spec.Template.Labels[kudo.HeritageLabel])

		// Verify that existing labels are not removed
		assert.Equal(t, "labelvalue", cron.Spec.JobTemplate.Labels["additional"])
		assert.Equal(t, "app-type", cron.Spec.JobTemplate.Spec.Template.Labels["app"])
	}
}

func TestEnhancerApply_noAdditionalMetadata(t *testing.T) {

	tpls := []runtime.Object{
		pod("pod", "default"),
		unstructuredCrd("crd", "default"),
	}

	meta := metadata()

	crdType := &metav1.APIResourceList{
		GroupVersion: "install.istio.io/v1alpha1",
		APIResources: []metav1.APIResource{
			{Name: "istiooperator", Namespaced: true, Kind: "IstioOperator"},
		},
	}

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: fake.CustomCachedDiscoveryClient(crdType),
	}

	objs, err := e.Apply(tpls, meta)
	if err != nil {
		t.Errorf("failed to apply template %s", err)
	}

	for _, o := range objs {
		unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(o)
		if err != nil {
			t.Errorf("failed to parse object to unstructured: %s", err)
		}

		// Make sure the top-level metadata is there
		uo := unstructured.Unstructured{Object: unstructMap}
		assert.Equal(t, string(meta.PlanUID), uo.GetAnnotations()[kudo.PlanUIDAnnotation])
		assert.Equal(t, "kudo", uo.GetLabels()[kudo.HeritageLabel])

		// But no nested fields
		f, ok, _ := unstructured.NestedFieldNoCopy(unstructMap, "spec", "template")
		assert.Nil(t, f)
		assert.False(t, ok, "%s struct contains template field", o.GetObjectKind())
	}
}
func TestEnhancerApply_dependencyHash_noDependencies(t *testing.T) {
	ss := statefulSet("statefulset", "default")

	tpls := []runtime.Object{ss}

	meta := metadata()
	meta.PlanUID = uuid.NewUUID()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: fake.CachedDiscoveryClient(),
	}

	objs, err := e.Apply(tpls, meta)
	if err != nil {
		t.Errorf("failed to apply template %s", err)
	}

	ssApplied := funk.Find(objs, func(o runtime.Object) bool {
		return o.GetObjectKind().GroupVersionKind() == schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	})

	unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ssApplied)
	assert.Nil(t, err, "failed to parse object to unstructured: %s", err)

	annotations, _, _ := unstructured.NestedMap(unstructMap, "spec", "template", "metadata", "annotations")
	assert.NotNil(t, annotations, "Statefulset pod template spec contains no annotations")

	hash := annotations[kudo.DependenciesHashAnnotation]
	assert.Nil(t, hash, "Pod template spec annotations contains a dependency hash but no dependencies")
}

func TestEnhancerApply_dependencyHash_unavailableResource(t *testing.T) {
	// Test that the dependency calculation does not error out on a resource that is NotAvailable at the moment

	ss := statefulSet("statefulset", "default")

	ss.Spec.Template.Spec.Volumes = append(ss.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "configMap",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: "unvailableConfigMap"},
			},
		},
	})

	tpls := []runtime.Object{ss}

	meta := metadata()
	meta.PlanUID = uuid.NewUUID()

	e := &DefaultEnhancer{
		Client:    clientfake.NewFakeClientWithScheme(scheme.Scheme),
		Scheme:    utils.Scheme(),
		Discovery: fake.CachedDiscoveryClient(),
	}

	objs, err := e.Apply(tpls, meta)
	if err != nil {
		t.Errorf("failed to apply template %s", err)
	}

	ssApplied := funk.Find(objs, func(o runtime.Object) bool {
		return o.GetObjectKind().GroupVersionKind() == schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	})

	unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ssApplied)
	assert.Nil(t, err, "failed to parse object to unstructured: %s", err)

	annotations, _, _ := unstructured.NestedMap(unstructMap, "spec", "template", "metadata", "annotations")
	assert.NotNil(t, annotations, "Statefulset pod template spec contains no annotations")

	hash := annotations[kudo.DependenciesHashAnnotation]
	assert.NotNil(t, hash, "Pod template spec annotations contains no dependency hash field")
}

func TestEnhancerApply_dependencyHash_calculatedOnResourceWithoutLastAppliedConfigAnnotation(t *testing.T) {
	// We may encounter references to resources that are not deployed by KUDO and do not have the
	// LastAppliedConfigAnnotation. We still need to calculate a mostly stable hash from the resource

	ss := statefulSet("statefulset", "default")
	cm := configMap("configmap", "default")

	ss.Spec.Template.Spec.Volumes = append(ss.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "configMap",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name},
			},
		},
	})

	tpls := []runtime.Object{ss}

	meta := metadata()
	meta.PlanUID = uuid.NewUUID()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: fake.CachedDiscoveryClient(),
		Client:    clientfake.NewFakeClientWithScheme(scheme.Scheme, cm),
	}

	objs, err := e.Apply(tpls, meta)
	if err != nil {
		t.Errorf("failed to apply template %s", err)
	}

	ssApplied := funk.Find(objs, func(o runtime.Object) bool {
		return o.GetObjectKind().GroupVersionKind() == schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	})

	unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ssApplied)
	assert.Nil(t, err, "failed to parse object to unstructured: %s", err)

	annotations, _, _ := unstructured.NestedMap(unstructMap, "spec", "template", "metadata", "annotations")
	assert.NotNil(t, annotations, "Statefulset pod template spec contains no annotations")

	hash := annotations[kudo.DependenciesHashAnnotation]
	assert.NotNil(t, hash, "Pod template spec annotations contains no dependency hash field")
}

func TestEnhancerApply_dependencyHash_changes(t *testing.T) {
	ss := statefulSet("statefulset", "default")
	cm := configMap("configmap", "default")

	ss.Spec.Template.Spec.Volumes = append(ss.Spec.Template.Spec.Volumes, corev1.Volume{
		Name: "configMap",
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{Name: cm.Name},
			},
		},
	})

	tpls := []runtime.Object{ss, cm}

	meta := metadata()
	meta.PlanUID = uuid.NewUUID()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: fake.CachedDiscoveryClient(),
	}

	objs, err := e.Apply(tpls, meta)
	if err != nil {
		t.Errorf("failed to apply template %s", err)
	}

	ssApplied := funk.Find(objs, func(o runtime.Object) bool {
		return o.GetObjectKind().GroupVersionKind() == schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	})

	unstructMap, err := runtime.DefaultUnstructuredConverter.ToUnstructured(ssApplied)
	assert.Nil(t, err, "failed to parse object to unstructured: %s", err)

	annotations, _, _ := unstructured.NestedMap(unstructMap, "spec", "template", "metadata", "annotations")
	assert.NotNil(t, annotations, "Statefulset pod template spec contains no annotations")

	hash := annotations[kudo.DependenciesHashAnnotation]
	assert.NotNil(t, hash, "Pod template spec annotations contains no dependency hash field")

	cm.Data["newkey"] = "newvalue"
	tpls = []runtime.Object{ss, cm}

	objs, err = e.Apply(tpls, meta)
	assert.Nil(t, err)
	ssApplied = funk.Find(objs, func(o runtime.Object) bool {
		return o.GetObjectKind().GroupVersionKind() == schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"}
	})

	unstructMap, err = runtime.DefaultUnstructuredConverter.ToUnstructured(ssApplied)
	assert.Nil(t, err, "failed to parse object to unstructured: %s", err)

	annotations, _, _ = unstructured.NestedMap(unstructMap, "spec", "template", "metadata", "annotations")
	assert.NotNil(t, annotations, "Statefulset pod template spec contains no annotations")

	newHash := annotations[kudo.DependenciesHashAnnotation]
	assert.NotNil(t, newHash, "Pod template spec annotations contains no dependency hash field")
	assert.NotEqual(t, hash, newHash, "Hashes are the same after the config map changed")
}

func metadata() Metadata {
	return Metadata{
		Metadata: engine.Metadata{
			InstanceName:        "instance",
			InstanceNamespace:   "namespace",
			OperatorName:        "operator",
			OperatorVersionName: "versionname",
			OperatorVersion:     "1.0.0",
			AppVersion:          "2.0.0",
			ResourcesOwner:      owner(),
		},
		PlanName:  "deploy",
		PlanUID:   "uid",
		PhaseName: "phase",
		StepName:  "step",
		TaskName:  "task",
	}
}

func owner() *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "core/v1",
		},
		ObjectMeta: metav1.ObjectMeta{},
		Spec:       corev1.PodSpec{},
		Status:     corev1.PodStatus{},
	}
}

func configMap(name string, namespace string) *corev1.ConfigMap {
	configMap := &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Data: map[string]string{
			"key": "value",
		},
	}
	return configMap
}

func statefulSet(name string, namespace string) *appsv1.StatefulSet {
	statefulSet := &appsv1.StatefulSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "StatefulSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: appsv1.StatefulSetSpec{
			ServiceName: name + "Service",
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"app": "app-type",
					},
					Annotations: map[string]string{},
				},
				Spec: corev1.PodSpec{},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"vct1": "vct1label",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{},
				},
				{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"vct2": "vct2label",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{},
				},
			},
		},
	}
	return statefulSet
}

func cronjob(name string, namespace string) *v1beta1.CronJob {
	cronjob := &v1beta1.CronJob{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CronJob",
			APIVersion: "batch/v1beta1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: v1beta1.CronJobSpec{
			Schedule:                "",
			StartingDeadlineSeconds: nil,
			ConcurrencyPolicy:       "",
			Suspend:                 nil,
			JobTemplate: v1beta1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: map[string]string{
						"additional": "labelvalue",
					},
				},
				Spec: v1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"app": "app-type",
							},
						},
						Spec: corev1.PodSpec{},
					},
				},
			},
			SuccessfulJobsHistoryLimit: nil,
			FailedJobsHistoryLimit:     nil,
		},
		Status: v1beta1.CronJobStatus{},
	}
	return cronjob
}

func pod(name string, namespace string) *corev1.Pod {
	pod := &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: corev1.PodSpec{},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
		},
	}
	return pod
}

func unstructuredCrd(name string, namespace string) runtime.Object {
	data := `apiVersion: install.istio.io/v1alpha1
kind: IstioOperator
metadata:
  namespace: ` + namespace + `
  name: ` + name + `
spec:
  profile: default`

	parsed, _ := YamlToObject(data)

	return parsed[0]
}

package renderer

import (
	"testing"

	v1 "k8s.io/api/batch/v1"

	"k8s.io/api/batch/v1beta1"

	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/engine"
	"github.com/kudobuilder/kudo/pkg/test/utils"
	"github.com/kudobuilder/kudo/pkg/util/kudo"
)

func TestEnhancerApply_embeddedMetadataStatefulSet(t *testing.T) {

	tpls := map[string]string{
		"deployment": resourceAsString(statefulSet("sfs1", "default")),
	}

	meta := metadata()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: utils.FakeDiscoveryClient(),
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
		assert.Equal(t, string(meta.PlanUID), sfs.Spec.Template.Annotations[kudo.PlanUIDAnnotation])

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

	tpls := map[string]string{
		"cron": resourceAsString(cronjob("cronjob", "default")),
	}

	meta := metadata()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: utils.FakeDiscoveryClient(),
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

		assert.Equal(t, string(meta.PlanUID), cron.Spec.JobTemplate.Spec.Template.Annotations[kudo.PlanUIDAnnotation])
		assert.Equal(t, "kudo", cron.Spec.JobTemplate.Spec.Template.Labels[kudo.HeritageLabel])

		// Verify that existing labels are not removed
		assert.Equal(t, "labelvalue", cron.Spec.JobTemplate.Labels["additional"])
		assert.Equal(t, "app-type", cron.Spec.JobTemplate.Spec.Template.Labels["app"])
	}
}

func TestEnhancerApply_noAdditionalMetadata(t *testing.T) {

	tpls := map[string]string{
		"pod": resourceAsString(pod("pod", "default")),
	}

	meta := metadata()

	e := &DefaultEnhancer{
		Scheme:    utils.Scheme(),
		Discovery: utils.FakeDiscoveryClient(),
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

		f, ok, _ := unstructured.NestedFieldNoCopy(unstructMap, "spec", "template")

		assert.Nil(t, f)
		assert.False(t, ok, "Pod struct contains template field")
	}
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

func resourceAsString(resource metav1.Object) string {
	bytes, _ := yaml.Marshal(resource)
	return string(bytes)
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
				},
				Spec: corev1.PodSpec{},
			},
			VolumeClaimTemplates: []corev1.PersistentVolumeClaim{
				corev1.PersistentVolumeClaim{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							"vct1": "vct1label",
						},
					},
					Spec: corev1.PersistentVolumeClaimSpec{},
				},
				corev1.PersistentVolumeClaim{
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

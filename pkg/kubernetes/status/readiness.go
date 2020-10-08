package status

import (
	"context"
	"fmt"
	"log"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	label "github.com/kudobuilder/kudo/pkg/util/kudo"
)

// IsReady computes instance readiness based on current state of underlying resources
// currently readiness examines the following types: Pods, StatefulSets, Deployments, ReplicaSets and DaemonSets
// Instance is considered ready if all the resources linked to this instance are also ready (healthy)
func IsReady(i kudoapi.Instance, c client.Client) (bool, string, error) {
	resources, err := healthResources(c, i.Name, i.Namespace)
	if err != nil {
		return false, "", err
	}
	ready := true
	readinessMessage := ""
	for _, res := range resources {
		healthy, msg, err := IsHealthy(res)
		if err != nil {
			return false, "", err
		}
		if !healthy {
			if readinessMessage == "" {
				readinessMessage = msg
			} else {
				readinessMessage += fmt.Sprintf(", %s", msg)
			}
			ready = false
		}
	}

	return ready, readinessMessage, nil
}

func healthResources(c client.Client, instanceName, instanceNamespace string) ([]runtime.Object, error) {
	instanceLabels, err := labels.Parse(fmt.Sprintf("%s=%s,%s=%s", label.InstanceLabel, instanceName, label.HeritageLabel, "kudo"))
	if err != nil {
		return nil, fmt.Errorf("unable to create list of labels to define health: %v", err)
	}

	dList := &appsv1.DeploymentList{}
	err = c.List(context.TODO(), dList, &client.ListOptions{Namespace: instanceNamespace, LabelSelector: instanceLabels})
	if err != nil {
		return nil, fmt.Errorf("unable to pull resources of type Deployment: %v", err)
	}

	ssList := &appsv1.StatefulSetList{}
	err = c.List(context.TODO(), ssList, &client.ListOptions{Namespace: instanceNamespace, LabelSelector: instanceLabels})
	if err != nil {
		return nil, fmt.Errorf("unable to pull resources of type StatefulSet: %v", err)
	}

	rsList := &appsv1.ReplicaSetList{}
	err = c.List(context.TODO(), rsList, &client.ListOptions{Namespace: instanceNamespace, LabelSelector: instanceLabels})
	if err != nil {
		return nil, fmt.Errorf("unable to pull resources of type ReplicaSet: %v", err)
	}

	dsList := &appsv1.DaemonSetList{}
	err = c.List(context.TODO(), dsList, &client.ListOptions{Namespace: instanceNamespace, LabelSelector: instanceLabels})
	if err != nil {
		return nil, fmt.Errorf("unable to pull resources of type DaemonSet: %v", err)
	}

	podsList := &corev1.PodList{}
	err = c.List(context.TODO(), podsList, &client.ListOptions{Namespace: instanceNamespace, LabelSelector: instanceLabels})
	if err != nil {
		return nil, fmt.Errorf("unable to pull resources of type Pod: %v", err)
	}

	var result []runtime.Object
	for i := range dList.Items {
		result = append(result, &dList.Items[i])
	}
	for i := range ssList.Items {
		result = append(result, &ssList.Items[i])
	}
	for i := range rsList.Items {
		result = append(result, &rsList.Items[i])
	}
	for i := range dsList.Items {
		result = append(result, &dsList.Items[i])
	}
	for i := range podsList.Items {
		result = append(result, &podsList.Items[i])
	}

	log.Printf("Computing health out of %d Deployments, %d ReplicaSets, %d StatefulSets, %d DaemonSets, %d Pods", len(dList.Items), len(rsList.Items), len(ssList.Items), len(dsList.Items), len(podsList.Items))

	return result, nil
}

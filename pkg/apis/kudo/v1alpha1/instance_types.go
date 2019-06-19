/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstanceSpec defines the desired state of Instance.
type InstanceSpec struct {
	// Framework specifies a reference to a specific Framework object.
	FrameworkVersion corev1.ObjectReference `json:"frameworkVersion,omitempty"`

	Dependencies []FrameworkDependency `json:"dependencies,omitempty"`
	Parameters   map[string]string     `json:"parameters,omitempty"`
}

// InstanceStatus defines the observed state of Instance
type InstanceStatus struct {
	// TODO turn into struct
	ActivePlan corev1.ObjectReference `json:"activePlan,omitempty"`
	Status     PhaseState             `json:"status,omitempty"`
}

/*
Using Kubernetes cluster: kubernetes-cluster1
deploy (serial strategy) (IN_PROGRESS)
├─ etcd (serial strategy) (STARTED)
│  ├─ etcd-0:[peer] (STARTED)
│  ├─ etcd-1:[peer] (PENDING)
│  └─ etcd-2:[peer] (PENDING)
├─ control-plane (dependency strategy) (PENDING)
│  ├─ kube-control-plane-0:[instance] (PENDING)
│  ├─ kube-control-plane-1:[instance] (PENDING)
│  └─ kube-control-plane-2:[instance] (PENDING)
├─ mandatory-addons (serial strategy) (PENDING)
│  └─ mandatory-addons-0:[instance] (PENDING)
├─ node (dependency strategy) (PENDING)
│  ├─ kube-node-0:[kubelet] (PENDING)
│  ├─ kube-node-1:[kubelet] (PENDING)
│  ├─ kube-node-2:[kubelet] (PENDING)
│  ├─ kube-node-3:[kubelet] (PENDING)
│  └─ kube-node-4:[kubelet] (PENDING)
└─ public-node (dependency strategy) (COMPLETE)
*/

// PhaseState captures the state of the rollout.
type PhaseState string

// PhaseStateInProgress actively deploying, but not yet healthy.
const PhaseStateInProgress PhaseState = "IN_PROGRESS"

// PhaseStatePending Not ready to deploy because dependent phases/steps not healthy.
const PhaseStatePending PhaseState = "PENDING"

// PhaseStateComplete deployed and healthy.
const PhaseStateComplete PhaseState = "COMPLETE"

// PhaseStateError there was an error deploying the application.
const PhaseStateError PhaseState = "ERROR"

// PhaseStateSuspend Spec was triggered to stop this plan execution.
const PhaseStateSuspend PhaseState = "SUSPEND"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Instance is the Schema for the instances API.
// +k8s:openapi-gen=true
type Instance struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanceSpec   `json:"spec,omitempty"`
	Status InstanceStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// InstanceList contains a list of Instance.
type InstanceList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Instance `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Instance{}, &InstanceList{})
}

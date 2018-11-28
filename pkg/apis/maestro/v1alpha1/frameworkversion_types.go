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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// FrameworkVersionSpec defines the desired state of FrameworkVersion
type FrameworkVersionSpec struct {
	// +optional
	Framework corev1.ObjectReference `json:"framework"`
	Version   string                 `json:"version"`
	//Defaults captures the default parameter values defined in the Yaml section.
	Defaults map[string]string `json:"defaults.config"`
	//Yaml captures a mustached yaml list of elements that define the application framework instance

	Yaml string `json:"yaml,omitempty"`

	//Plans specify a map a plans that specify how to
	Plans map[string]Plan `json:"plans,omitempty"`

	//ConnectionString defines a mustached string that can be used
	// to connect to an instance of the Framework
	// +optional
	ConnectionString string `json:"connectionString,omitempty"`

	//Dependencies a list of

	Dependencies []FrameworkDependency `json:"dependencies,omitempty"`

	//UpgradableFrom lists all FrameworkVersions that can upgrade to this FrameworkVersion
	UpgradableFrom []FrameworkVersion `json:"upgradableFrom,omitempty"`
}

//Ordering specifies how the subitems in this plan/phase should be rolled out
type Ordering string

//Serial specifies that the plans or objects should be created in order.  The first should be healthy before
// continuing on
const Serial Ordering = "serial"

//Parallel specifies that the plan or objects in the phase can all be lauched at the same time
const Parallel Ordering = "parallel"

//Plan specifies a series of Phases that need to be completed.
type Plan struct {
	Strategy Ordering `json:"strategy"`
	//Phases maps a phase name to a Phase object
	Phases []Phase `json:"phases"`
}

//Phase specifies a list of steps that contain Kubernetes objects.
type Phase struct {
	Name     string   `json:"name"`
	Strategy Ordering `json:"strategy"`
	//Steps maps a step name to a list of mustached kubernetes objects stored as a string
	Steps []Step `json:"steps"`
}

type Step struct {
	Name     string `json:"name"`
	Mustache string `json:"mustache"`
	//Objects will be serialized for each instance as the params and defaults
	// are provided
	Objects []runtime.Object `json:"-"`
}

// FrameworkVersionStatus defines the observed state of FrameworkVersion
type FrameworkVersionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FrameworkVersion is the Schema for the frameworkversions API
// +k8s:openapi-gen=true
type FrameworkVersion struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   FrameworkVersionSpec   `json:"spec,omitempty"`
	Status FrameworkVersionStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// FrameworkVersionList contains a list of FrameworkVersion
type FrameworkVersionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []FrameworkVersion `json:"items"`
}

func init() {
	SchemeBuilder.Register(&FrameworkVersion{}, &FrameworkVersionList{})
}

//FrameworkDependency references a defined framework
type FrameworkDependency struct {
	//Name specifies the name of the dependency.  Referenced via this in defaults.config
	ReferenceName string `json:"referenceName"`
	corev1.ObjectReference
	//Version captures the requirements for what versions
	// of the above object are allowed
	// Example: ^3.1.4
	Version string `json:"version"`
}

// Type representations for srvc.yml as defined http://mesosphere.github.io/dcos-commons/yaml-reference/
// They are a 1-to-1 mapping of the Java SDK port definition and hence the value ranges are choosen to match the Java
// counterparts.

// ServiceSpec represents the overall service definition.
type ServiceSpec struct {
	Name      *string          `json:"name" yaml:"name"`
	WebUrl    *string          `json:"web-url" yaml:"web-url"`
	Scheduler *Scheduler       `json:"scheduler" yaml:"scheduler"`
	Pods      map[string]*Pod  `json:"pods" yaml:"pods"`
	Plans     map[string]*Plan `json:"plans" yaml:"plans"`
}

// Scheduler contains settings for the (Mesos) scheduler from the original yaml.
// It is not required for this project, but kept for compatibility reasons.
type Scheduler struct {
	Principal *string `json:"principal" yaml:"principal"`
	Zookeeper *string `json:"zookeeper" yaml:"zookeeper"`
	User      *string `json:"user" yaml:"user"`
}

// Pod represents a pod definition which could be deployed as part of a plan.
type Pod struct {
	ResourceSets      map[string]*ResourceSet `json:"resource-sets" yaml:"resource-sets"`
	Placement         *string                 `json:"placement" yaml:"placement"`
	Count             int32                   `json:"count" yaml:"count"`
	Image             *string                 `json:"image" yaml:"image"`
	Networks          map[string]*Network     `json:"networks" yaml:"networks"`
	RLimits           map[string]*RLimit      `json:"rlimits" yaml:"rlimits"`
	Uris              []*string               `json:"uris" yaml:"uris"`
	Tasks             map[string]*Task        `json:"tasks" yaml:"tasks"`
	Volume            *Volume                 `json:"volume" yaml:"volume"`
	Volumes           map[string]*Volume      `json:"volumes" yaml:"volumes"`
	PreReservedRole   *string                 `json:"pre-reserved-role" yaml:"pre-reserved-role"`
	Secrets           map[string]*Secret      `json:"secrets" yaml:"secrets"`
	SharePidNamespace bool                    `json:"share-pid-namespace" yaml:"share-pid-namespace"`
	AllowDecommission bool                    `json:"allow-decommission" yaml:"allow-decommission"`
	HostVolumes       map[string]*HostVolume  `json:"host-volumes" yaml:"host-volumes"`
}

// ResourceSets defines a single set of resources which can be reused across tasks.
type ResourceSet struct {
	Cpus     float64            `json:"cpus" yaml:"cpus"`
	Gpus     float64            `json:"gpus" yaml:"gpus"`
	MemoryMB int32              `json:"memory" yaml:"memory"`
	Ports    map[string]*Port   `json:"ports" yaml:"ports"`
	Volume   *Volume            `json:"volume" yaml:"volume"`
	Volumes  map[string]*Volume `json:"volumes" yaml:"volumes"`
}

// Network represents a network setting or virtual network to be joined.
type Network struct {
	HostPorts      []int32 `json:"host-ports" yaml:"host-ports"`
	ContainerPorts []int32 `json:"container-ports" yaml:"container-ports"`
	Labels         string  `json:"labels" yaml:"labels"`
}

// RLmit represents a rlimit setting to be applied to the pod.
type RLimit struct {
	Soft int64 `json:"soft" yaml:"soft"`
	Hard int64 `json:"hard" yaml:"hard"`
}

// Task specifies a task to be run inside the pod.
type Task struct {
	Goal                    *string                `json:"goal" yaml:"goal"`
	Essential               bool                   `json:"essential" yaml:"essential"`
	Cmd                     *string                `json:"cmd" yaml:"cmd"`
	Env                     map[string]*string     `json:"env" yaml:"env"`
	Configs                 map[string]*Config     `json:"configs" yaml:"configs"`
	Cpus                    float64                `json:"cpus" yaml:"cpus"`
	Gpus                    float64                `json:"gpus" yaml:"gpus"`
	MemoryMB                float64                `json:"memory" yaml:"memory"`
	Ports                   map[string]*Port       `json:"ports" yaml:"ports"`
	HealthCheck             *HealthCheck           `json:"health-check" yaml:"health-check"`
	ReadinessCheck          *ReadinessCheck        `json:"readiness-check" yaml:"readiness-check"`
	Volume                  *Volume                `json:"volume" yaml:"volume"`
	Volumes                 map[string]*Volume     `json:"volumes" yaml:"volumes"`
	ResourceSet             *string                `json:"resource-set" yaml:"resource-set"`
	Discovery               *Discovery             `json:"discovery" yaml:"discovery"`
	TaskKillGracePeriodSecs int32                  `json:"kill-grace-period" yaml:"kill-grace-period"`
	TransportEncryption     []*TransportEncryption `json:"transport-encryption" yaml:"transport-encryption"`
}

// Config represents a config templates which is rendered before the pods is launched.
type Config struct {
	Template *string `json:"template" yaml:"template"`
	Dest     *string `json:"dest" yaml:"dest"`
}

// HealthCheck represents a validation that a pod is healthy.
type HealthCheck struct {
	Cmd                    *string `json:"cmd" yaml:"cmd"`
	GracePeriodSecs        int32   `json:"grace-period" yaml:"grace-period"`
	MaxConsecutiveFailures int32   `json:"max-consecutive-failures" yaml:"max-consecutive-failures"`
	DelaySecs              int32   `json:"delay" yaml:"delay"`
	TimeoutSecs            int32   `json:"timeout" yaml:"timeout"`
}

// ReadinessCheck represents an additional HealthCheck which is used when a task is first started.
type ReadinessCheck struct {
	Cmd          *string `json:"cmd" yaml:"cmd"`
	IntervalSecs int32   `json:"interval" yaml:"interval"`
	DelaySecs    int32   `json:"delay" yaml:"delay"`
	TimeoutSecs  int32   `json:"timeout" yaml:"timeout"`
}

// Discovery represents information about how a tasks should be exposed.
type Discovery struct {
	Prefix     *string `json:"prefix" yaml:"prefix"`
	Visibility *string `json:"visibility" yaml:"visibility"`
}

// TransportEncryption represents information about the TLS certificate used for encrypting traffic.
type TransportEncryption struct {
	Name *string `json:"name" yaml:"name"`
	Type *string `json:"type" yaml:"type"`
}

// Port represents a port an application should listen on.
type Port struct {
	Port      int32   `json:"port" yaml:"port"`
	EnvKey    *string `json:"env-key" yaml:"env-key"`
	Advertise bool    `json:"advertise" yaml:"advertise"`
	VIP       *VIP    `json:"vip" yaml:"vip"`
}

// VIP define a Virtual IP address for a given port.
type VIP struct {
	Port   int32   `json:"port" yaml:"port"`
	Prefix *string `json:"prefix" yaml:"prefix"`
}

// Volume to be mounted in the Pod environment.
type Volume struct {
	Path   *string `json:"path" yaml:"path"`
	Type   *string `json:"type" yaml:"type"`
	SizeMB int32   `json:"size" yaml:"size"`
}

// Secret to be made available in the Pod environment.
type Secret struct {
	SecretPath *string `json:"secret" yaml:"secret"`
	EnvKey     *string `json:"env-key" yaml:"env-key"`
	FilePath   *string `json:"file" yaml:"file"`
}

// HostVolume to be mounted in the Pod environment.
type HostVolume struct {
	HostPath      *string `json:"host-path" yaml:"host-path"`
	ContainerPath *string `json:"container-path" yaml:"container-path"`
}

// // Plan represents a deployment/recovery plan.
// type Plan struct {
// 	Strategy *string           `json:"strategy" yaml:"strategy"`
// 	Phases   map[string]*Phase `json:"phases" yaml:"phases"`
// }

// // Phase represents a subphase of a given plan.
// type Phase struct {
// 	Strategy *string                   `json:"strategy" yaml:"strategy"`
// 	Steps    []*map[string]*[][]string `json:"steps" yaml:"steps"`
// 	Pod      *string                   `json:"pod" yaml:"pod"`
// }

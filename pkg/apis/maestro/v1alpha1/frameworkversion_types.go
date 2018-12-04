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
	"k8s.io/apimachinery/pkg/runtime"
)

// FrameworkVersionSpec defines the desired state of FrameworkVersion
type FrameworkVersionSpec struct {
	// +optional
	Framework corev1.ObjectReference `json:"framework,omitempty"`
	Version   string                 `json:"version,omitempty"`
	//Defaults captures the default parameter values defined in the Yaml section.
	Defaults map[string]string `json:"defaults.config,omitempty"`
	//Yaml captures a mustached yaml list of elements that define the application framework instance
	Templates map[string]string   `json:"templates,omitempty"`
	Tasks     map[string]TaskSpec `json:"tasks,omitempty"`

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

// TaskSpec is a struct containing lists of of Kustomize resources
type TaskSpec struct {
	Resources []string `json:"resources"`
}

//Phase specifies a list of steps that contain Kubernetes objects.
type Phase struct {
	Name     string   `json:"name"`
	Strategy Ordering `json:"strategy"`
	//Steps maps a step name to a list of mustached kubernetes objects stored as a string
	Steps []Step `json:"steps"`
}

type Step struct {
	Name  string   `json:"name"`
	Tasks []string `json:"tasks"`
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
	Name      *string          `json:"name" yaml:"name" validate:"required,gt=0"`                              // makes field mandatory and checks if set and non empty
	WebUrl    *string          `json:"web-url" yaml:"web-url" validate:"required,gt=0"`                        // makes field mandatory and checks if set and non empty
	Scheduler *Scheduler       `json:"scheduler" yaml:"scheduler"`                                             // field optional, no need to validate
	Pods      map[string]*Pod  `json:"pods" yaml:"pods" validate:"gt=0,dive,keys,required,endkeys,required"`   // makes field mandatory but when set it checks for non empty keys and non empty&nil values
	Plans     map[string]*Plan `json:"plans" yaml:"plans" validate:"gt=0,dive,keys,required,endkeys,required"` // makes field mandatory but when set it checks for non empty keys and non empty&nil values
}

// Scheduler contains settings for the (Mesos) scheduler from the original yaml.
// It is not required for this project, but kept for compatibility reasons.
type Scheduler struct {
	Principal *string `json:"principal" yaml:"principal" validate:"omitempty,gt=0"` // makes field optional but when set checks if non empty&nil values
	Zookeeper *string `json:"zookeeper" yaml:"zookeeper" validate:"omitempty,gt=0"` // makes field optional but when set checks if non empty&nil values
	User      *string `json:"user" yaml:"user" validate:"omitempty,gt=0"`           // makes field optional but when set checks if non empty&nil values
}

// Pod represents a pod definition which could be deployed as part of a plan.
type Pod struct {
	ResourceSets      map[string]*ResourceSet `json:"resource-sets" yaml:"resource-sets" validate:"gte=0,dive,keys,required,endkeys,required"` // makes field optional but when set it checks for non empty keys and non empty&nil values
	Placement         *string                 `json:"placement" yaml:"placement" validate:"omitempty,gt=0"`                                    // makes field optional but when set checks if non empty&nil values
	Count             int32                   `json:"count" yaml:"count" validate:"gte=1"`                                                     // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set)
	Image             *string                 `json:"image" yaml:"image" validate:"omitempty,gt=0"`                                            // makes field optional but when set checks if non empty&nil values
	Networks          map[string]*Network     `json:"networks" yaml:"networks" validate:"gte=0,dive,keys,required,endkeys,required"`           // makes field optional but when set it checks for non empty keys and non empty&nil values
	RLimits           map[string]*RLimit      `json:"rlimits" yaml:"rlimits" validate:"gte=0,dive,keys,required,endkeys,required"`             // makes field optional but when set it checks for non empty keys and non empty&nil values
	Uris              []*string               `json:"uris" yaml:"uris" validate:"gte=0,dive,gte=1"`                                            // makes field mandatory and checks if its a valid port (TODO: check for duplicates)
	Tasks             map[string]*Task        `json:"tasks" yaml:"tasks" validate:"required,dive,keys,required,endkeys,required"`              // makes field optional but when set it checks for non empty keys and non empty&nil values
	Volume            *Volume                 `json:"volume" yaml:"volume"`                                                                    // field optional, no need to validate
	Volumes           map[string]*Volume      `json:"volumes" yaml:"volumes" validate:"gte=0,dive,keys,required,endkeys,required"`             // makes field optional but when set it checks for non empty keys and non empty&nil values
	PreReservedRole   *string                 `json:"pre-reserved-role" yaml:"pre-reserved-role" validate:"omitempty,gt=0"`                    // makes field optional but when set checks if non empty&nil values
	Secrets           map[string]*Secret      `json:"secrets" yaml:"secrets" validate:"gte=0,dive,keys,required,endkeys,required"`             // makes field optional but when set it checks for non empty keys and non empty&nil values
	SharePidNamespace bool                    `json:"share-pid-namespace" yaml:"share-pid-namespace"`                                          // no checks needed
	AllowDecommission bool                    `json:"allow-decommission" yaml:"allow-decommission"`                                            // no checks needed
	HostVolumes       map[string]*HostVolume  `json:"host-volumes" yaml:"host-volumes" validate:"gte=0,dive,keys,required,endkeys,required"`   // makes field optional but when set it checks for non empty keys and non empty&nil values
}

// ResourceSets defines a single set of resources which can be reused across tasks.
type ResourceSet struct {
	Cpus     float64            `json:"cpus" yaml:"cpus" validate:"omitempty,gt=0"`                                  // makes field optional but if set value needs to be gt 0 (TODO: we should make this a pointer to check if set)
	Gpus     float64            `json:"gpus" yaml:"gpus" validate:"omitempty,gt=0"`                                  // makes field optional but if set value needs to be gt 0 (TODO: we should make this a pointer to check if set)
	MemoryMB int32              `json:"memory" yaml:"memory" validate:"omitempty,gte=1"`                             // makes field optional but if set value needs to be gte 1 (TODO: we should make this a pointer to check if set)
	Ports    map[string]*Port   `json:"ports" yaml:"ports" validate:"gte=0,dive,keys,required,endkeys,required"`     // makes field optional but when set it checks for non empty keys and non empty&nil values
	Volume   *Volume            `json:"volume" yaml:"volume"`                                                        // field optional, no need to validate
	Volumes  map[string]*Volume `json:"volumes" yaml:"volumes" validate:"gte=0,dive,keys,required,endkeys,required"` // makes field optional but when set it checks for non empty keys and non empty&nil values
}

// Network represents a network setting or virtual network to be joined.
type Network struct {
	HostPorts      []int32 `json:"host-ports" yaml:"host-ports" validate:"gte=1,dive,gte=1,max=65535"`           // makes field mandatory and checks if its a valid port (TODO: we should make this a pointer to check if set, also check for duplicates)
	ContainerPorts []int32 `json:"container-ports" yaml:"container-ports" validate:"gte=1,dive,gte=1,max=65535"` // makes field mandatory and checks if its a valid port (TODO: we should make this a pointer to check if set, also check for duplicates)
	Labels         string  `json:"labels" yaml:"labels"`                                                         // no checks needed (TODO: not sure if needed at all, e.g. would it map to a service in k8s)
}

// RLmit represents a rlimit setting to be applied to the pod.
type RLimit struct {
	Soft int64 `json:"soft" yaml:"soft" validate:"gte=1"` // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set | does this apply in k8s land?)
	Hard int64 `json:"hard" yaml:"hard" validate:"gte=1"` // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set | does this apply in k8s land?)
}

// Task specifies a task to be run inside the pod.
type Task struct {
	Goal                    *string                `json:"goal" yaml:"goal" validate:"required,gt=0,eq=RUNNING|eq=FINISH|eq=ONCE"`          // makes field mandatory and checks if set and non empty, allows just RUNNING,FINISH,ONCE
	Essential               bool                   `json:"essential" yaml:"essential" validate:"isdefault"`                                 // makes field optional but if set it expects it to be false and will fail if true (TODO: we should make this a pointer to check if set)
	Cmd                     *string                `json:"cmd" yaml:"cmd" validate:"required,gt=0"`                                         // makes field mandatory and checks if set and non empty
	Env                     map[string]*string     `json:"env" yaml:"env" validate:"gte=0,dive,keys,required,endkeys,required"`             // makes field optional but when set it checks for non empty keys and non empty&nil values
	Configs                 map[string]*Config     `json:"configs" yaml:"configs" validate:"gte=0,dive,keys,required,endkeys,required"`     // makes field optional but when set it checks for non empty keys and non empty&nil values
	Cpus                    float64                `json:"cpus" yaml:"cpus" validate:"omitempty,gt=0"`                                      // makes field optional but if set value needs to be gt 0 (TODO: we should make this a pointer to check if set)
	Gpus                    float64                `json:"gpus" yaml:"gpus" validate:"omitempty,gt=0"`                                      // makes field optional but if set value needs to be gt 0 (TODO: we should make this a pointer to check if set)
	MemoryMB                float64                `json:"memory" yaml:"memory" validate:"omitempty,gte=1"`                                 // makes field optional but if set value needs to be gte 1 (TODO: we should make this a pointer to check if set)
	Ports                   map[string]*Port       `json:"ports" yaml:"ports" validate:"gte=0,dive,keys,required,endkeys,required"`         // makes field optional but when set it checks for non empty keys and non empty&nil values
	HealthCheck             *HealthCheck           `json:"health-check" yaml:"health-check"`                                                // field optional, no need to validate
	ReadinessCheck          *ReadinessCheck        `json:"readiness-check" yaml:"readiness-check"`                                          // field optional, no need to validate
	Volume                  *Volume                `json:"volume" yaml:"volume"`                                                            // field optional, no need to validate
	Volumes                 map[string]*Volume     `json:"volumes" yaml:"volumes" validate:"gte=0,dive,keys,required,endkeys,required"`     // makes field optional but when set it checks for non empty keys and non empty&nil values
	ResourceSet             *string                `json:"resource-set" yaml:"resource-set" validate:"omitempty,gt=0"`                      // makes field optional but when set checks if non empty&nil values
	Discovery               *Discovery             `json:"discovery" yaml:"discovery"`                                                      // field optional, no need to validate
	TaskKillGracePeriodSecs int32                  `json:"kill-grace-period" yaml:"kill-grace-period" validate:"gte=0"`                     // makes field optional and checks if its gte 0 (TODO: we should make this a pointer to check if set)
	TransportEncryption     []*TransportEncryption `json:"transport-encryption" yaml:"transport-encryption" validate:"omitempty,gt=0,dive"` // makes field optional and checks if its gt 0
}

// Config represents a config templates which is rendered before the pods is launched.
type Config struct {
	Template *string `json:"template" yaml:"template" validate:"required,gt=0"` // makes field mandatory and checks if set and non empty
	Dest     *string `json:"dest" yaml:"dest" validate:"required,gt=0"`         // makes field mandatory and checks if set and non empty
}

// HealthCheck represents a validation that a pod is healthy.
type HealthCheck struct {
	Cmd *string `json:"cmd" yaml:"cmd" validate:"omitempty,gt=0"` // makes field optional but when set checks if non empty&nil values
	//Interval               int32   `json:"interval" yaml:"interval" validate:"gte=1"`                                 // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set)
	GracePeriodSecs        int32 `json:"grace-period" yaml:"grace-period" validate:"gte=1"`                         // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set)
	MaxConsecutiveFailures int32 `json:"max-consecutive-failures" yaml:"max-consecutive-failures" validate:"gte=0"` // makes field optional and checks if its gte 0 (TODO: we should make this a pointer to check if set | can it be zero?)
	DelaySecs              int32 `json:"delay" yaml:"delay" validate:"gte=0"`                                       // makes field optional and checks if its gte 0 (TODO: we should make this a pointer to check if set)
	TimeoutSecs            int32 `json:"timeout" yaml:"timeout" validate:"gte=1"`                                   // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set)
}

// ReadinessCheck represents an additional HealthCheck which is used when a task is first started.
type ReadinessCheck struct {
	Cmd          *string `json:"cmd" yaml:"cmd" validate:"omitempty,gt=0"`  // makes field optional but when set checks if non empty&nil values
	IntervalSecs int32   `json:"interval" yaml:"interval" validate:"gte=1"` // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set)
	DelaySecs    int32   `json:"delay" yaml:"delay" validate:"gte=0"`       // makes field optional and checks if its gte 0 (TODO: we should make this a pointer to check if set)
	TimeoutSecs  int32   `json:"timeout" yaml:"timeout" validate:"gte=1"`   // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set)
}

// Discovery represents information about how a tasks should be exposed.
type Discovery struct {
	Prefix     *string `json:"prefix" yaml:"prefix"`         // no checks needed (TODO: not sure if needed at all, e.g. would it map to a service in k8s)
	Visibility *string `json:"visibility" yaml:"visibility"` // no checks needed (TODO: not sure if needed at all, e.g. would it map to a service in k8s)
}

// TransportEncryption represents information about the TLS certificate used for encrypting traffic.
type TransportEncryption struct {
	Name *string `json:"name" yaml:"name" validate:"required,gt=0"`                    // makes field mandatory and checks if set and non empty
	Type *string `json:"type" yaml:"type" validate:"required,gt=0,eq=TLS|eq=KEYSTORE"` // makes field mandatory and checks if set and non empty, allows just TLS or KEYSTORE
}

// Port represents a port an application should listen on.
type Port struct {
	Port      int32   `json:"port" yaml:"port" validate:"gte=1,max=65535"`      // makes field mandatory and checks if its a valid port (TODO: we should make this a pointer to check if set)
	EnvKey    *string `json:"env-key" yaml:"env-key" validate:"omitempty,gt=0"` // makes field optional but when set checks if non empty&nil values
	Advertise bool    `json:"advertise" yaml:"advertise"`                       // no checks needed (TODO: not sure if needed at all, e.g. would it map to a service in k8s)
	VIP       *VIP    `json:"vip" yaml:"vip"`                                   // no checks needed (TODO: not sure if needed at all, e.g. would it map to a service in k8s)
}

// VIP define a Virtual IP address for a given port.
type VIP struct {
	Port   int32   `json:"port" yaml:"port"`
	Prefix *string `json:"prefix" yaml:"prefix"`
}

// Volume to be mounted in the Pod environment.
type Volume struct {
	Path   *string `json:"path" yaml:"path" validate:"required,gt=0"`                  // makes field mandatory and checks if set and non empty (TODO: MatchRegex Custom Validation)
	Type   *string `json:"type" yaml:"type" validate:"required,gt=0,eq=ROOT|eq=MOUNT"` // makes field mandatory and checks if set and non empty, allows just ROOT or MOUNT
	SizeMB int32   `json:"size" yaml:"size" validate:"gte=1"`                          // makes field mandatory and checks if its gte 1 (TODO: we should make this a pointer to check if set)`
}

// Secret to be made available in the Pod environment.
type Secret struct {
	SecretPath *string `json:"secret" yaml:"secret" validate:"required,gt=0"`    // makes field mandatory and checks if set and non empty
	EnvKey     *string `json:"env-key" yaml:"env-key" validate:"omitempty,gt=0"` // makes field optional but when set checks if non empty&nil values
	FilePath   *string `json:"file" yaml:"file" validate:"omitempty,gt=0"`       // makes field optional but when set checks if non empty&nil values (TODO: MatchRegex Custom Validation)
}

// HostVolume to be mounted in the Pod environment.
type HostVolume struct {
	HostPath      *string `json:"host-path" yaml:"host-path" validate:"required,gt=0"`           // makes field mandatory and checks if set and non empty (TODO: MatchRegex Custom Validation)
	ContainerPath *string `json:"container-path" yaml:"container-path" validate:"required,gt=0"` // makes field mandatory and checks if set and non empty (TODO: MatchRegex Custom Validation)
}

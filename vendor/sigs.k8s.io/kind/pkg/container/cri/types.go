/*
Copyright 2019 The Kubernetes Authors.

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

package cri

/*
These types are from
https://github.com/kubernetes/kubernetes/blob/063e7ff358fdc8b0916e6f39beedc0d025734cb1/pkg/kubelet/apis/cri/runtime/v1alpha2/api.pb.go#L183
*/

// Mount specifies a host volume to mount into a container.
// This is a close copy of the upstream cri Mount type
// see: k8s.io/kubernetes/pkg/kubelet/apis/cri/runtime/v1alpha2
// It additionally serializes the "propagation" field with the string enum
// names on disk as opposed to the int32 values, and the serlialzed field names
// have been made closer to core/v1 VolumeMount field names
// In yaml this looks like:
//  containerPath: /foo
//  hostPath: /bar
//  readOnly: true
//  selinuxRelabel: false
//  propagation: None
// Propagation may be one of: None, HostToContainer, Bidirectional
type Mount struct {
	// Path of the mount within the container.
	ContainerPath string `protobuf:"bytes,1,opt,name=container_path,json=containerPath,proto3" json:"containerPath,omitempty"`
	// Path of the mount on the host. If the hostPath doesn't exist, then runtimes
	// should report error. If the hostpath is a symbolic link, runtimes should
	// follow the symlink and mount the real destination to container.
	HostPath string `protobuf:"bytes,2,opt,name=host_path,json=hostPath,proto3" json:"hostPath,omitempty"`
	// If set, the mount is read-only.
	Readonly bool `protobuf:"varint,3,opt,name=readonly,proto3,json=readOnly,proto3" json:"readOnly,omitempty"`
	// If set, the mount needs SELinux relabeling.
	SelinuxRelabel bool `protobuf:"varint,4,opt,name=selinux_relabel,json=selinuxRelabel,proto3" json:"selinuxRelabel,omitempty"`
	// Requested propagation mode.
	Propagation MountPropagation `protobuf:"varint,5,opt,name=propagation,proto3,enum=runtime.v1alpha2.MountPropagation" json:"propagation,omitempty"`
}

// PortMapping specifies a host port mapped into a container port.
// In yaml this looks like:
//  containerPort: 80
//  hostPort: 8000
//  listenAddress: 127.0.0.1
type PortMapping struct {
	// Port within the container.
	ContainerPort int32 `protobuf:"varint,1,opt,name=container_port,json=containerPort,proto3" json:"containerPort,omitempty"`
	// Port on the host.
	HostPort int32 `protobuf:"varint,2,opt,name=host_path,json=hostPort,proto3" json:"hostPort,omitempty"`
	// TODO: add protocol (tcp/udp) and port-ranges
	ListenAddress string `protobuf:"bytes,3,opt,name=listenAddress,json=hostPort,proto3" json:"listenAddress,omitempty"`
}

// MountPropagation represents an "enum" for mount propagation options,
// see also Mount.
type MountPropagation int32

const (
	// MountPropagationNone specifies that no mount propagation
	// ("private" in Linux terminology).
	MountPropagationNone MountPropagation = 0
	// MountPropagationHostToContainer specifies that mounts get propagated
	// from the host to the container ("rslave" in Linux).
	MountPropagationHostToContainer MountPropagation = 1
	// MountPropagationBidirectional specifies that mounts get propagated from
	// the host to the container and from the container to the host
	// ("rshared" in Linux).
	MountPropagationBidirectional MountPropagation = 2
)

// MountPropagationValueToName is a map of valid MountPropogation values to
// their string names
var MountPropagationValueToName = map[MountPropagation]string{
	MountPropagationNone:            "None",
	MountPropagationHostToContainer: "HostToContainer",
	MountPropagationBidirectional:   "Bidirectional",
}

// MountPropagationNameToValue is a map of valid MountPropogation names to
// their values
var MountPropagationNameToValue = map[string]MountPropagation{
	"None":            MountPropagationNone,
	"HostToContainer": MountPropagationHostToContainer,
	"Bidirectional":   MountPropagationBidirectional,
}

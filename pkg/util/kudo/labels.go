package kudo

const (
	// OperatorLabel is k8s label key for identifying operator
	OperatorLabel = "kudo.dev/operator"
	// OperatorVersionAnnotation is k8s label key for operator version
	OperatorVersionAnnotation = "kudo.dev/operator-version"
	// InstanceLabel is k8s label key for KUDO instance name
	InstanceLabel = "kudo.dev/instance"
	// HeritageLabel is k8s label key for heritage
	HeritageLabel = "heritage" // this is not specific to KUDO

	// PlanAnnotation is k8s annotation key for plan name that created this object
	PlanAnnotation = "kudo.dev/plan"
	// PhaseAnnotation is k8s annotation key for phase that created this object
	PhaseAnnotation = "kudo.dev/phase"
	// StepAnnotation is k8s annotation key for step that created this object
	StepAnnotation = "kudo.dev/step"
)

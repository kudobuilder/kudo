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

	// PlanUIDAnnotation is a k8s annotation key for the last time a given plan was run on the referenced object
	PlanUIDAnnotation = "kudo.dev/last-plan-execution-uid"

	// DependenciesHash is used to trigger pod reloads if one of the dependencies of the resource is changed
	DependenciesHashAnnotation = "kudo.dev/dependencies-hash"

	// Used to ignore this resource in the calculation of the dependencies hash
	SkipHashCalculationAnnotation = "kudo.dev/skip-hash-calculation"

	// Last applied state for three way merges
	LastAppliedConfigAnnotation = "kudo.dev/last-applied-configuration"
)

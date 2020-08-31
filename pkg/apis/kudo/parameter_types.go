package kudo

// Parameter captures the variability of an OperatorVersion being instantiated in an instance.
type Parameter struct {
	// DisplayName can be used by UIs.
	DisplayName string

	// Name is the string that should be used in the template file for example,
	// if `name: COUNT` then using the variable in a spec like:
	//
	// spec:
	//   replicas:  {{ .Params.COUNT }}
	Name string

	// Description captures a longer description of how the parameter will be used.
	Description string

	// Required specifies if the parameter is required to be provided by all instances, or whether a default can suffice.
	Required *bool

	// Default is a default value if no parameter is provided by the instance.
	Default *string

	// Trigger identifies the plan that gets executed when this parameter changes in the Instance object.
	// Default is `update` if a plan with that name exists, otherwise it's `deploy`.
	Trigger string

	// Type specifies the value type. Defaults to `string`.
	Type ParameterType

	// Specifies if the parameter can be changed after the initial installation of the operator
	Immutable *bool
}

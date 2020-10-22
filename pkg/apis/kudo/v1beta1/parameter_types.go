package v1beta1

// Parameter captures the variability of an OperatorVersion being instantiated in an instance.
type Parameter struct {
	// DisplayName can be used by UIs.
	DisplayName string `json:"displayName,omitempty"`

	// Name is the string that should be used in the template file for example,
	// if `name: COUNT` then using the variable in a spec like:
	//
	// spec:
	//   replicas:  {{ .Params.COUNT }}
	Name string `json:"name,omitempty"`

	// Description captures a longer description of how the parameter will be used.
	Description string `json:"description,omitempty"`

	// Required specifies if the parameter is required to be provided by all instances, or whether a default can suffice.
	Required *bool `json:"required,omitempty"`

	// Default is a default value if no parameter is provided by the instance.
	Default *string `json:"default,omitempty"`

	// Trigger identifies the plan that gets executed when this parameter changes in the Instance object.
	// Default is `update` if a plan with that name exists, otherwise it's `deploy`.
	Trigger string `json:"trigger,omitempty"`

	// Type specifies the value type. Defaults to `string`.
	Type ParameterType `json:"value-type,omitempty"`

	// Specifies if the parameter can be changed after the initial installation of the operator
	Immutable *bool `json:"immutable,omitempty"`

	// Defines a list of allowed values. If Default is set and Enum is not nil, the value must be in this list as well
	Enum *[]string `json:"enum,omitempty"`
}

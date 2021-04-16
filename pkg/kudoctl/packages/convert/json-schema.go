package convert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/qri-io/jsonschema"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

const (
	jsonSchemaTypeObject = "object"
	jsonSchemaTypeString = "string"
)

type jsonSchema struct {
	Schema      string                 `json:"$schema,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
	Enum        []interface{}          `json:"enum,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Immutable   bool                   `json:"immutable,omitempty"`
	Advanced    bool                   `json:"advanced,omitempty"`
	Hint        string                 `json:"hint,omitempty"`
	Trigger     string                 `json:"trigger,omitempty"`
	ListName    string                 `json:"listName,omitempty"`
	Properties  map[string]*jsonSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
}

func (j *jsonSchema) hasRequiredPropertyWithoutDefault() bool {
	for _, pName := range j.Required {
		prop := j.Properties[pName]

		if prop.Default == nil {
			return true
		}
		if prop.hasRequiredPropertyWithoutDefault() {
			return true
		}
	}
	return false
}

func newSchema() *jsonSchema {
	return &jsonSchema{
		Properties: map[string]*jsonSchema{},
		Required:   []string{},
	}
}

func buildGroups(ov *kudoapi.OperatorVersion) map[string]kudoapi.ParameterGroup {
	groups := map[string]kudoapi.ParameterGroup{}
	for _, g := range ov.Spec.Groups {
		groups[g.Name] = g
	}
	for _, p := range ov.Spec.Parameters {
		if p.Group != "" {
			if _, ok := groups[p.Group]; !ok {
				groups[p.Group] = kudoapi.ParameterGroup{
					Name: p.Group,
				}
			}
		}
	}

	return groups
}

func buildTopLevelGroups(groups map[string]kudoapi.ParameterGroup) map[string]*jsonSchema {
	topLevelGroups := map[string]*jsonSchema{}

	for _, v := range groups {
		g := newSchema()

		g.Type = jsonSchemaTypeObject

		if v.DisplayName != "" {
			g.Title = v.DisplayName
		} else {
			g.Title = v.Name
		}
		if v.Description != "" {
			g.Description = v.Description
		}
		if v.Priority != 0 {
			g.Priority = v.Priority
		}

		topLevelGroups[v.Name] = g
	}

	return topLevelGroups
}

func jsonSchemaTypeFromKudoType(parameterType kudoapi.ParameterType) string {
	switch parameterType {
	// Most types are exactly as in json-schema
	case kudoapi.StringValueType,
		kudoapi.IntegerValueType,
		kudoapi.NumberValueType,
		kudoapi.BooleanValueType,
		kudoapi.ArrayValueType:
		return string(parameterType)

	// Objects are the equivalent to maps
	case kudoapi.MapValueType:
		return jsonSchemaTypeObject

	// All other types are defined as strings
	default:
		return jsonSchemaTypeString
	}
}

func UnwrapToJSONType(wrapped string, parameterType kudoapi.ParameterType) (unwrapped interface{}, err error) {
	switch parameterType {
	case kudoapi.MapValueType,
		kudoapi.ArrayValueType,
		kudoapi.NumberValueType,
		kudoapi.BooleanValueType:
		return convert.UnwrapParamValue(&wrapped, parameterType)
	case kudoapi.IntegerValueType,
		kudoapi.StringValueType:
		// We keep integers as strings, as JSON only supports numbers, which may not always map correctly to integers
		return wrapped, nil
	default:
		return wrapped, nil
	}
}

func UnwrapEnumValues(values []string, parameterType kudoapi.ParameterType) ([]interface{}, error) {
	result := make([]interface{}, 0, len(values))
	for _, v := range values {
		vUnwrapped, err := UnwrapToJSONType(v, parameterType)
		if err != nil {
			return nil, err
		}
		result = append(result, vUnwrapped)
	}
	return result, nil
}

func buildParamSchema(p kudoapi.Parameter) (*jsonSchema, error) {
	var err error

	param := newSchema()

	if p.DisplayName != "" {
		param.Title = p.DisplayName
	}
	if p.Description != "" {
		param.Description = p.Description
	}
	if p.HasDefault() {
		if param.Default, err = UnwrapToJSONType(*p.Default, p.Type); err != nil {
			return nil, fmt.Errorf("failed to convert default value %s: %v", *p.Default, err)
		}
	}
	param.Type = jsonSchemaTypeFromKudoType(p.Type)
	if p.IsImmutable() {
		param.Immutable = true
	}
	if p.IsAdvanced() {
		param.Advanced = true
	}
	if p.Hint != "" {
		param.Hint = p.Hint
	}
	if p.Trigger != "" {
		param.Trigger = p.Trigger
	}
	param.ListName = p.Name
	if p.IsEnum() {
		if param.Enum, err = UnwrapEnumValues(p.EnumValues(), p.Type); err != nil {
			return nil, fmt.Errorf("failed to convert enum values: %v", err)
		}
	}

	return param, nil
}

func WriteJSONSchema(ov *kudoapi.OperatorVersion, outputType output.Type, out io.Writer) error {
	root := newSchema()
	topLevelGroups := buildTopLevelGroups(buildGroups(ov))

	root.Properties = topLevelGroups
	root.Type = jsonSchemaTypeObject
	root.Description = "All parameters for this operator"
	root.Title = fmt.Sprintf("Parameters for %s", ov.Name)

	for _, p := range ov.Spec.Parameters {
		param, err := buildParamSchema(p)
		if err != nil {
			return fmt.Errorf("failed to convert parameter %s: %v", p.Name, err)
		}

		// Assign to correct parent
		parent := topLevelGroups[p.Group]
		if parent == nil {
			parent = root
		}

		if p.IsRequired() {
			parent.Required = append(parent.Required, p.Name)
		}

		parent.Properties[p.Name] = param
	}

	for k, v := range topLevelGroups {
		if v.hasRequiredPropertyWithoutDefault() {
			root.Required = append(root.Required, k)
		}
	}

	buf := new(bytes.Buffer)
	err := output.WriteObject(root, output.TypeJSON, buf)
	if err != nil {
		return err
	}

	schemaJSON := MustAsset("config/json-schema/limited.json")

	s := &jsonschema.Schema{}
	if err := json.Unmarshal(schemaJSON, s); err != nil {
		return fmt.Errorf("failed to unmarshal json-schema: %v", err)
	}

	var doc interface{}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		return fmt.Errorf("error parsing JSON bytes: %s", err.Error())
	}

	vs := s.Validate(context.TODO(), doc)
	errs := *vs.Errs
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("Error: %v\n", e)
		}
		fmt.Printf("%s\n", buf.String())
		return fmt.Errorf("failed to validate json schema")
	}

	return output.WriteObject(root, outputType, out)
}

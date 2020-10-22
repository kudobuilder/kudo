package convert

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/qri-io/jsonschema"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/cmd/output"
)

type jsonSchema struct {
	Schema      string                 `json:"$schema,omitempty"`
	Title       string                 `json:"title,omitempty"`
	Description string                 `json:"description,omitempty"`
	Priority    int                    `json:"priority,omitempty"`
	Default     interface{}            `json:"default,omitempty"`
	Type        string                 `json:"type,omitempty"`
	Immutable   bool                   `json:"immutable,omitempty"`
	Advanced    bool                   `json:"advanced,omitempty"`
	Hint        string                 `json:"hint,omitempty"`
	ListName    string                 `json:"listName,omitempty"`
	Properties  map[string]*jsonSchema `json:"properties,omitempty"`
	Required    []string               `json:"required,omitempty"`
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
		return "object"

	// All other types are defined as strings
	default:
		return "string"
	}
}

func buildParamSchema(p kudoapi.Parameter) *jsonSchema {
	param := newSchema()

	if p.DisplayName != "" {
		param.Title = p.DisplayName
	}
	if p.Description != "" {
		param.Description = p.Description
	}
	if p.HasDefault() {
		param.Default = p.Default
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
	param.ListName = p.Name

	return param
}

func WriteJSONSchema(ov *kudoapi.OperatorVersion, outputType output.Type, out io.Writer) error {
	root := newSchema()
	topLevelGroups := buildTopLevelGroups(buildGroups(ov))

	root.Properties = topLevelGroups
	root.Type = "object"
	root.Description = "All parameters for this operator"
	root.Title = fmt.Sprintf("Parameters for %s", ov.Name)

	for _, p := range ov.Spec.Parameters {
		param := buildParamSchema(p)

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

	buf := new(bytes.Buffer)
	err := output.WriteObject(root, output.TypeJSON, buf)
	if err != nil {
		return err
	}

	g, err := ioutil.ReadFile("pkg/kudoctl/packages/convert/fullschema.json")
	if err != nil {
		return fmt.Errorf("failed to read thingy: %v", err)
	}

	s := &jsonschema.Schema{}
	if err := json.Unmarshal(g, s); err != nil {
		return fmt.Errorf("failed to unmarshal json-schema: %v", err)
	}

	var doc interface{}
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		return fmt.Errorf("error parsing JSON bytes: %s", err.Error())
	}

	fmt.Printf("Start validating...\n")
	vs := s.Validate(context.TODO(), doc)
	errs := *vs.Errs
	if len(errs) > 0 {
		for _, e := range errs {
			fmt.Printf("Error: %v\n", e)
		}
		return fmt.Errorf("failed to validate json schema")
	}

	return output.WriteObject(root, outputType, out)
}

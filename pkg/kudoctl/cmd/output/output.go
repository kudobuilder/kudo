package output

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/thoas/go-funk"
	"sigs.k8s.io/yaml"
)

// OutputType specifies the type of output a command produces
type Type string

const (
	// StringValueType is used for parameter values that are provided as a string.
	TypeYAML Type = "yaml"

	// ArrayValueType is used for parameter values that described an array of values.
	TypeJSON Type = "json"

	InvalidOutputError = "invalid output format, only support 'yaml' or 'json' or empty"
)

var (
	ValidTypes = []Type{TypeYAML, TypeJSON}
)

func (t *Type) AsStringPtr() *string {
	return (*string)(t)
}

func (t Type) IsFormattedOutput() bool {
	return t != ""
}

func (t Type) Validate() error {
	if t == "" {
		return nil
	}
	if funk.Contains(ValidTypes, t) {
		return nil
	}

	return fmt.Errorf(InvalidOutputError)
}

func WriteObjects(objs []interface{}, outputType Type, out io.Writer) error {
	if outputType == TypeYAML {
		// Write YAML objects with separators
		for _, obj := range objs {
			if _, err := fmt.Fprintln(out, "---"); err != nil {
				return err
			}

			if err := writeObjectYAML(obj, out); err != nil {
				return err
			}
		}

		// YAML ending document boundary marker
		_, err := fmt.Fprintln(out, "...")
		return err
	} else if outputType == TypeJSON {
		return writeObjectJSON(objs, out)
	}

	return fmt.Errorf(InvalidOutputError)
}

func WriteObject(obj interface{}, outputType Type, out io.Writer) error {
	if outputType == TypeYAML {
		return writeObjectYAML(obj, out)
	} else if outputType == TypeJSON {
		return writeObjectJSON(obj, out)
	}

	return fmt.Errorf(InvalidOutputError)
}

func writeObjectJSON(obj interface{}, out io.Writer) error {
	o, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal to json: %v", err)
	}
	_, err = fmt.Fprintln(out, string(o))
	return err
}

func writeObjectYAML(obj interface{}, out io.Writer) error {
	o, err := yaml.Marshal(obj)
	if err != nil {
		return fmt.Errorf("failed to marshal to yaml: %v", err)
	}
	_, err = fmt.Fprintln(out, string(o))
	return err
}

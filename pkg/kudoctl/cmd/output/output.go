package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"sigs.k8s.io/yaml"
)

const (
	TypeYAML = "yaml"
	TypeJSON = "json"
)

func WriteObjects(objs []interface{}, outputType string, out io.Writer) error {
	if strings.ToLower(outputType) == TypeYAML {
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
	} else if strings.ToLower(outputType) == TypeJSON {
		return writeObjectJSON(objs, out)
	}

	return fmt.Errorf("invalid output format, only support yaml or json")
}

func WriteObject(obj interface{}, outputType string, out io.Writer) error {
	if strings.ToLower(outputType) == TypeYAML {
		return writeObjectYAML(obj, out)
	} else if strings.ToLower(outputType) == TypeJSON {
		return writeObjectJSON(obj, out)
	}

	return fmt.Errorf("invalid output format, only support yaml or json")
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

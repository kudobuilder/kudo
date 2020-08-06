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
	var o []byte
	var err error
	if strings.ToLower(outputType) == TypeYAML {
		for _, obj := range objs {
			o, err = yaml.Marshal(obj)
			if err != nil {
				return fmt.Errorf("failed to marshal to yaml: %v", err)
			}
			if _, err := fmt.Fprintln(out, "---"); err != nil {
				return err
			}

			if _, err := fmt.Fprintln(out, string(o)); err != nil {
				return err
			}
		}

		// YAML ending document boundary marker
		_, err = fmt.Fprintln(out, "...")

		return err
	} else if strings.ToLower(outputType) == TypeJSON {
		o, err = json.MarshalIndent(objs, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal to json: %v", err)
		}
		if _, err := fmt.Fprintln(out, string(o)); err != nil {
			return err
		}
		return nil
	}

	return nil
}

func WriteObject(obj interface{}, outputType string, out io.Writer) error {
	var o []byte
	var err error
	if strings.ToLower(outputType) == TypeYAML {
		o, err = yaml.Marshal(obj)
		if err != nil {
			return fmt.Errorf("failed to marshal to yaml: %v", err)
		}
	} else if strings.ToLower(outputType) == TypeJSON {
		o, err = json.MarshalIndent(obj, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal to json: %v", err)
		}
	}

	if _, err := fmt.Fprintln(out, string(o)); err != nil {
		return err
	}
	return nil
}

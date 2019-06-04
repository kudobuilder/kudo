package importcmd

import (
	"encoding/json"
	"fmt"

	// "gopkg.in/yaml.v2"

	"github.com/ghodss/yaml"

	"github.com/spf13/cobra"

	"github.com/kudobuilder/kudo/pkg/kudoctl/util/vars"
	"github.com/kudobuilder/kudo/pkg/util/helm"
)

// Run runs the import command
func Run(cmd *cobra.Command, args []string) error {
	f, fv, e := helm.Import(vars.FrameworkImportPath)
	output := ""
	if e != nil {
		return e
	}
	switch vars.Format {
	// Convert to json first so it respects the `inline` tag for typemeta
	case "yaml":
		b, e := json.Marshal(f)
		if e != nil {
			return e
		}
		y, err := yaml.JSONToYAML(b)
		if err != nil {
			return err
		}
		output += string(y)
		output += "\n---\n"
		b, e = json.Marshal(fv)
		if e != nil {
			return e
		}
		y, err = yaml.JSONToYAML(b)
		if err != nil {
			return err
		}
		output += string(y)
	case "json":
		b, e := json.Marshal(f)
		if e != nil {
			return e
		}
		output += string(b)
		output += "\n"
		b, e = json.Marshal(fv)
		if e != nil {
			return e
		}
		output += string(b)
	default:
		return fmt.Errorf("Invalid output format %v.  Only valid options are \"json\" and \"yaml\"", vars.Format)
	}
	fmt.Printf(output)
	return nil
}

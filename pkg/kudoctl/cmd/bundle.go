package cmd

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kudobuilder/kudo/pkg/bundle"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"
)

const apiVersion = "kudo.k8s.io/v1alpha1"

type bundleCmd struct {
	out        io.Writer
	bundlePath string
}

// NewBundleCmd creates a bundle command for the CLI
func newBundleCmd(out io.Writer) *cobra.Command {
	bundleCmd := &cobra.Command{
		Use:     "bundle <path>",
		Short:   "-> Bundle a package from the Beta Framework format into FrameworkVersion (experimental)",
		Long:    `Bundle a package from the Beta Framework format into FrameworkVersion (experimental)`,
		Example: "kubectl kudo bundle . | kubectl apply -f -",
		RunE: func(cmd *cobra.Command, args []string) error {
			bundle := &bundleCmd{
				out: out,
			}

			if len(args) != 1 {
				return errors.Errorf("Bundle requires 1 path argument, got %d", len(args))
			}

			bp, err := resolveBundlePath(args[0])
			if err != nil {
				return err
			}
			bundle.bundlePath = bp

			return bundle.run()
		},
		SilenceUsage: true,
	}

	const usageFmt = "Usage:\n  %s\n\nFlags:\n%s"
	bundleCmd.SetUsageFunc(func(cmd *cobra.Command) error {
		fmt.Fprintf(bundleCmd.OutOrStderr(), usageFmt, bundleCmd.UseLine(), bundleCmd.Flags().FlagUsages())
		return nil
	})
	return bundleCmd
}

func (b *bundleCmd) run() error {
	frameworkPath := path.Join(b.bundlePath, "framework.yaml")
	if _, err := os.Stat(frameworkPath); err != nil {
		return err
	}

	f, err := ioutil.ReadFile(frameworkPath)
	if err != nil {
		return err
	}

	var bf bundle.Framework

	if err = yaml.Unmarshal(f, &bf); err != nil {
		return errors.Wrap(err, "unmarshal framework")
	}

	templatePath := path.Join(b.bundlePath, "templates")

	if _, err := os.Stat(templatePath); err != nil {
		return err
	}

	tpls := map[string]string{}

	err = filepath.Walk(templatePath, func(path string, info os.FileInfo, _ error) error {
		if info.IsDir() {
			if templatePath == path {
				return nil
			}

			return errors.New("can't parse sub-directories")
		}

		f, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		tpls[info.Name()] = string(f)
		return nil
	})

	if err != nil {
		return errors.Wrap(err, "load templates")
	}

	errs := []string{}

	for k, v := range bf.Tasks {
		for _, res := range v.Resources {
			if _, ok := tpls[res]; !ok {
				errs = append(errs, fmt.Sprintf("task %s missing resource: %s", k, res))
			}
		}
	}

	if len(errs) != 0 {
		return errors.New(strings.Join(errs, ", "))
	}

	framework := &v1alpha1.Framework{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Framework",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: bf.Name,
		},
		Spec: v1alpha1.FrameworkSpec{
			Description:       bf.Description,
			KudoVersion:       bf.KUDOVersion,
			KubernetesVersion: bf.KubernetesVersion,
			Maintainers:       bf.Maintainers,
			URL:               bf.URL,
		},
		Status: v1alpha1.FrameworkStatus{},
	}

	fv := &v1alpha1.FrameworkVersion{
		TypeMeta: metav1.TypeMeta{
			Kind:       "FrameworkVersion",
			APIVersion: apiVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name: fmt.Sprintf("%s-%s", bf.Name, bf.Version),
		},
		Spec: v1alpha1.FrameworkVersionSpec{
			Version:        bf.Version,
			Templates:      tpls,
			Tasks:          bf.Tasks,
			Parameters:     bf.Parameters,
			Plans:          bf.Plans,
			Dependencies:   bf.Dependencies,
			UpgradableFrom: nil,
		},
		Status: v1alpha1.FrameworkVersionStatus{},
	}

	specs := []interface{}{framework, fv}

	var output bytes.Buffer

	for _, spec := range specs {
		output.Write([]byte("---\n"))

		rendered, err := yaml.Marshal(spec)
		if err != nil {
			return err
		}

		output.Write(rendered)
	}

	b.out.Write(output.Bytes())

	return nil
}

func resolveBundlePath(path string) (string, error) {

	if _, err := os.Stat(path); err != nil {
		return path, err
	}

	abs, err := filepath.Abs(path)
	if err != nil {
		return abs, err
	}

	return abs, nil

}

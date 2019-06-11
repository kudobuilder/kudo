package convert

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"

	"github.com/spf13/cobra"
	"sigs.k8s.io/yaml"

	kudo "github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/util/helm"
)

type Options struct {
	ChartImportPath string
	OutputPath string
}

var DefaultOptions = &Options{}

// Run runs the convert command
func Run(cmd *cobra.Command, args []string, options *Options) error {
	bundle, e := helm.ToBundle(options.ChartImportPath)
	if e != nil {
		return e
	}
	if b := exists(options.OutputPath); b {
		return fmt.Errorf("bundle output dir %v shouldn't exist", options.OutputPath)
	}
	e = os.Mkdir(options.OutputPath, 0755)
	if e != nil {
		return e
	}

	bundleParams := bundle.Parameters
	bundle.Parameters = make([]kudo.Parameter, 0)
	b, e := yaml.Marshal(bundle)
	if e != nil {
		return e
	}
	e = ioutil.WriteFile(fmt.Sprintf("%v/operator.yaml", options.OutputPath), b, os.ModePerm)
	if e != nil {
		return e
	}
	type P struct {
		Default     string `json:"default,omitempty"`
		Description string `json:"description,omitempty"`
		Trigger     string `json:"trigger,omitempty"`
	}
	mapParams := make(map[string]P)
	for _, param := range bundleParams {
		mapParams[param.Name] = P{
			Default:     param.Default,
			Description: param.Description,
			Trigger:     param.Trigger,
		}
	}
	b, e = yaml.Marshal(mapParams)
	if e != nil {
		return e
	}
	e = ioutil.WriteFile(fmt.Sprintf("%v/params.yaml", options.OutputPath), b, os.ModePerm)
	if e != nil {
		return e
	}

	e = os.Mkdir(options.OutputPath+"/templates", 0755)
	if e != nil {
		return e
	}

	e = copyDirectory(fmt.Sprintf("%v/templates", options.ChartImportPath), fmt.Sprintf("%v/templates", options.OutputPath))

	return e
}

func copyDirectory(scrDir, dest string) error {
	entries, err := ioutil.ReadDir(scrDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		sourcePath := filepath.Join(scrDir, entry.Name())
		destPath := filepath.Join(dest, entry.Name())

		fileInfo, err := os.Stat(sourcePath)
		if err != nil {
			return err
		}

		stat, ok := fileInfo.Sys().(*syscall.Stat_t)
		if !ok {
			return fmt.Errorf("failed to get raw syscall.Stat_t data for '%s'", sourcePath)
		}

		switch fileInfo.Mode() & os.ModeType {
		case os.ModeDir:
			if err := createIfNotExists(destPath, 0755); err != nil {
				return err
			}
			if err := copyDirectory(sourcePath, destPath); err != nil {
				return err
			}
		case os.ModeSymlink:
			if err := copySymLink(sourcePath, destPath); err != nil {
				return err
			}
		default:
			if err := copy(sourcePath, destPath); err != nil {
				return err
			}
		}

		if err := os.Lchown(destPath, int(stat.Uid), int(stat.Gid)); err != nil {
			return err
		}

		isSymlink := entry.Mode()&os.ModeSymlink != 0
		if !isSymlink {
			if err := os.Chmod(destPath, entry.Mode()); err != nil {
				return err
			}
		}
	}
	return nil
}

func copy(srcFile, dstFile string) error {
	out, err := os.Create(dstFile)
	if err != nil {
		return err
	}
	defer out.Close()

	in, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer in.Close()

	_, err = io.Copy(out, in)
	if err != nil {
		return err
	}

	return nil
}

func exists(filePath string) bool {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return false
	}

	return true
}

func createIfNotExists(dir string, perm os.FileMode) error {
	if exists(dir) {
		return nil
	}

	if err := os.MkdirAll(dir, perm); err != nil {
		return fmt.Errorf("failed to create directory: '%s', error: '%s'", dir, err.Error())
	}

	return nil
}

func copySymLink(source, dest string) error {
	link, err := os.Readlink(source)
	if err != nil {
		return err
	}
	return os.Symlink(link, dest)
}

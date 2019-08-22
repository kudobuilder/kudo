package cmd

import (
	"fmt"
	"io"
	"time"

	"github.com/kudobuilder/kudo/pkg/kudoctl/files"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const repoIndexDesc = `
Read the provided directory and generate an index file based on the packages found.

This tool is used for creating an 'index.yaml' file for a KUDO package repository. To
set an absolute URL to the operators, use '--url' flag.

# Create an index file for all KUDO packages in the repo-dir.
	$ kubectl kudo repo index repo-dir`

type repoIndexCmd struct {
	path      string
	url       string
	overwrite bool
	out       io.Writer
	fs        afero.Fs
}

// newRepoIndexCmd for repo commands such as building a repo index file.
func newRepoIndexCmd(fs afero.Fs, out io.Writer) *cobra.Command {
	index := &repoIndexCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:   "index [flags] <DIR>",
		Short: "Generate an index file given a directory containing KUDO packages",
		Long:  repoIndexDesc,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateRepoIndex(args); err != nil {
				return err
			}
			index.path = args[0]
			if err := index.run(); err != nil {
				return err
			}
			return nil
		},
		SilenceUsage: true,
	}

	f := cmd.Flags()
	f.StringVar(&index.url, "url", "", "URL of the operator repository")
	f.BoolVarP(&index.overwrite, "overwrite", "o", false, "Overwrite existing package")

	return cmd
}

func validateRepoIndex(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - directory containing the operators to package")
	}
	return nil
}

// run returns the errors associated with cmd env
func (ri *repoIndexCmd) run() error {
	target, err := files.FullPathToTarget(ri.fs, ri.path, "index.yaml", ri.overwrite)
	if err != nil {
		return err
	}

	t := time.Now()
	i, err := repo.IndexDirectory(ri.fs, ri.path, ri.url, &t)
	if err != nil {
		return err
	}

	i.WriteFile(fs, target)
	fmt.Fprintf(ri.out, "index %v created.\n", target)
	return nil
}

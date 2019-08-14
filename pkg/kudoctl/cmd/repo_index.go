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

This tool is used for creating an 'index.yaml' file for a kudo package repository. To
set an absolute URL to the charts, use '--url' flag.

To merge the generated index with an existing index file, use the '--merge'
flag. In this case, the charts found in the working directory will be merged
into the existing index, with local packages taking priority over existing packages.

# create an index file for all kudo packages in the repo-dir
	$ kubectl kudo repo index repo-dir`

type repoIndexCmd struct {
	path      string
	url       string
	overwrite bool
	out       io.Writer
	fs        afero.Fs
}

// newRepoIndexCmd for repo commands such as building a repo index
func newRepoIndexCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	index := &repoIndexCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:   "index [flags] [DIR]",
		Short: "Generate an index file given a directory containing kudo packages",
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
	f.StringVar(&index.url, "url", "", "URL of the chart repository")
	//f.BoolVarP(&index.merge, "merge", "", false, "Merge the generated index into the given index")
	f.BoolVarP(&index.overwrite, "overwrite", "o", false, "Overwrite existing package.")

	return cmd
}

func validateRepoIndex(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("expecting exactly one argument - directory of the operator to package")
	}
	return nil
}

// run returns the errors associated with cmd env
func (ri *repoIndexCmd) run() error {
	target, err := files.FullPathToTarget(ri.fs, ri.path, "index.yaml", ri.overwrite)
	if err != nil {
		return err
	}

	//todo: move ^^ into index dir
	repository, err := repo.NewOperatorRepository(repo.Default)
	if err != nil {
		return err
	}
	t := time.Now()
	//todo: need to pass url and need a default for url
	i, err := repository.IndexDirectory(ri.fs, ri.path, target, "", &t)
	if err != nil {
		return err
	}

	fmt.Printf("working with %v\n", target)

	fmt.Printf("archives: %v\n", i)

	// 1. walk folder get a list of tgz files indexDirectory
	// 2. create index objects
	// 3. sort
	// 3. write

	return nil
}

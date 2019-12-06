package cmd

import (
	"errors"
	"fmt"
	"io"

	"github.com/manifoldco/promptui"
	"github.com/spf13/afero"
	"github.com/spf13/cobra"
)

const (
	pkgNewDesc = `Create a new KUDO operator on the local filesystem`

	pkgNewExample = `  # package zookeeper (where zookeeper is a folder in the current directory)
  kubectl kudo package new ./operator
`
)

type packageNewCmd struct {
	path        string
	destination string
	overwrite   bool
	out         io.Writer
	fs          afero.Fs
}

// newPackageNewCmd creates an operator package on the file system
func newPackageNewCmd(fs afero.Fs, out io.Writer) *cobra.Command {

	pkg := &packageNewCmd{out: out, fs: fs}
	cmd := &cobra.Command{
		Use:     "new <operator_dir>",
		Short:   "package a local KUDO operator",
		Long:    pkgNewDesc,
		Example: pkgNewExample,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := validateOperatorArg(args); err != nil {
				return err
			}
			pkg.path = args[0]
			if err := pkg.run(); err != nil {
				return err
			}
			return nil
		},
	}

	f := cmd.Flags()
	f.StringVarP(&pkg.destination, "destination", "d", ".", "Location to write the package.")
	f.BoolVarP(&pkg.overwrite, "overwrite", "w", false, "Overwrite existing package.")
	return cmd
}

// run returns the errors associated with cmd env
func (pkg *packageNewCmd) run() error {

	//validate := func(input string) error {
	//	_, err := strconv.ParseFloat(input, 64)
	//	return err
	//}
	//
	//templates := &promptui.PromptTemplates{
	//	Prompt:  "{{ . }} ",
	//	Valid:   "{{ . | green }} ",
	//	Invalid: "{{ . | red }} ",
	//	Success: "{{ . | bold }} ",
	//}
	//
	//prompt := promptui.Prompt{
	//	Label:     "Spicy Level",
	//	Templates: templates,
	//	Validate:  validate,
	//}
	//
	//result, err := prompt.Run()
	//
	//if err != nil {
	//	fmt.Printf("Prompt failed %v\n", err)
	//	return nil
	//}
	//
	//fmt.Printf("You answered %s\n", result)
	//return nil

	//validate := func(input string) error {
	//	if len(input) < 3 {
	//		return errors.New("Username must have more than 3 characters")
	//	}
	//	return nil
	//}
	//
	//var username string
	//u, err := user.Current()
	//if err == nil {
	//	username = u.Username
	//}
	//
	//prompt := promptui.Prompt{
	//	Label:    "Username",
	//	Validate: validate,
	//	Default:  username,
	//}
	//
	//result, err := prompt.Run()
	//
	//if err != nil {
	//	fmt.Printf("Prompt failed %v\n", err)
	//	return nil
	//}
	//
	//fmt.Printf("Your username is %q\n", result)

	//items := []string{"Vim", "Emacs", "Sublime", "VSCode", "Atom"}
	//index := -1
	//var result string
	//var err error
	//
	//for index < 0 {
	//	prompt := promptui.SelectWithAdd{
	//		Label:    "What's your text editor",
	//		Items:    items,
	//		AddLabel: "Other",
	//	}
	//
	//	index, result, err = prompt.Run()
	//
	//	if index == -1 {
	//		items = append(items, result)
	//	}
	//}
	//
	//if err != nil {
	//	fmt.Printf("Prompt failed %v\n", err)
	//}
	//
	//fmt.Printf("You choose %s\n", result)

//apiVersion: kudo.dev/v1beta1
//name: redis
//version: 0.1.0
//kudoVersion: 0.3.0
//kubernetesVersion: 1.15.0
//appVersion: 5.0.1
//url: https://redis.io/

	name, err := name(pkg.path)
	fmt.Printf(name)

	dir, err := dir("operator")
	fmt.Printf(dir)

	return err
}

func cursor (input []rune) []rune {
	//return []rune("\u258D")
	return input
}

func name(name string) (string, error) {
	validate := func(input string) error {
		if len(input) < 3 {
			return errors.New("Operator name must have more than 3 characters")
		}
		return nil
	}

	prompt := promptui.Prompt{
		Label:    "Operator Name",
		Validate: validate,
		Default:  name,
		Pointer:  cursor,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

func dir(defaultDir string) (string, error) {
	prompt := promptui.Prompt{
		Label:    "Operator directory",
		Default:  defaultDir,
		Pointer:  cursor,
	}

	result, err := prompt.Run()

	if err != nil {
		return "", err
	}

	return result, nil
}

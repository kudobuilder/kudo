package main

import (
	"fmt"
	"io/ioutil"
	"os"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"github.com/cbroglie/mustache"
)

var rootCmd = &cobra.Command{
	Use: "mustache [data] template",
	Example: `  $ mustache data.yml template.mustache
  $ cat data.yml | mustache template.mustache`,
	Args: cobra.RangeArgs(0, 2),
	Run: func(cmd *cobra.Command, args []string) {
		err := run(cmd, args)
		if err != nil {
			fmt.Printf("Error: %s\n", err.Error())
			os.Exit(1)
		}
	},
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func run(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		cmd.Usage()
		return nil
	}

	var data interface{}
	var templatePath string
	if len(args) == 1 {
		var err error
		data, err = parseDataFromStdIn()
		if err != nil {
			return err
		}
		templatePath = args[0]
	} else {
		var err error
		data, err = parseDataFromFile(args[0])
		if err != nil {
			return err
		}
		templatePath = args[1]
	}

	output, err := mustache.RenderFile(templatePath, data)
	if err != nil {
		return err
	}

	fmt.Print(output)
	return nil
}

func parseDataFromStdIn() (interface{}, error) {
	b, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data, nil
}

func parseDataFromFile(filePath string) (interface{}, error) {
	b, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	var data interface{}
	if err := yaml.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	return data, nil
}

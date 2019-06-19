package main

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
)

type Options struct {
	Namespace  string
	Parameters map[string]string
}

// parseParameters parse raw parameter strings into a map of keys and values
func parseParameters(raw []string, parameters map[string]string) error {
	var errs []string

	for _, a := range raw {
		// Using '=' as the delimiter. Split after the first delimiter to support using '=' in values
		s := strings.SplitN(a, "=", 2)
		if len(s) < 2 {
			errs = append(errs, fmt.Sprintf("parameter not set: %+v", a))
			continue
		}
		if s[0] == "" {
			errs = append(errs, fmt.Sprintf("parameter name can not be empty: %+v", a))
			continue
		}
		if s[1] == "" {
			errs = append(errs, fmt.Sprintf("parameter value can not be empty: %+v", a))
			continue
		}
		parameters[s[0]] = s[1]
	}

	if errs != nil {
		return errors.New(strings.Join(errs, ", "))
	}

	return nil
}

func main() {

	var echoTimes int
	var parameters []string
	var options = &Options{
		Namespace: "default",
	}

	var cmdPrint = &cobra.Command{
		Use:   "print [string to print]",
		Short: "Print anything to the screen",
		Long:  "print is for printing anything back to the screen. For many years people have printed back to the screen.",
		RunE: func(cmd *cobra.Command, args []string) error {

			fmt.Printf("PreRun: with unparsed parameters %v", parameters)

			options.Parameters = make(map[string]string)
			if err := parseParameters(parameters, options.Parameters); err != nil {
				return errors.New("could not parse parameters: " + err.Error())
			}

			fmt.Printf("Print: \"%v\" with parameters: %v\n", strings.Join(args, " "), options.Parameters)

			return nil
		},
	}

	cmdPrint.Flags().IntVarP(&echoTimes, "times", "t", 1, "times to echo the input")
	cmdPrint.Flags().StringArrayVarP(&parameters, "parameter", "p", nil, "The parameter name and value separated by '='")

	var rootCmd = &cobra.Command{Use: "app"}
	rootCmd.AddCommand(cmdPrint, cmdPrint)
	rootCmd.Execute()
}

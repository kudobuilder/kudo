package install

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/afero"
	"gopkg.in/yaml.v2"
)

// GetParameterMap takes a slice of parameter strings and a slice of parameter filenames as well as a filesystem,
// parses parameters into a map of keys and values
func GetParameterMap(raw []string, filePaths []string, fs afero.Fs) (map[string]string, error) {
	var errs []string
	parameters := make(map[string]string)

	for _, filePath := range filePaths {
		var err error
		rawData, err := afero.ReadFile(fs, filePath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("error reading from parameter file %s: %v", filePath, err))
			continue
		}
		data := make(map[string]string)
		err = yaml.Unmarshal(rawData, data)
		if err != nil {
			errs = append(errs, fmt.Sprintf("error unmarshalling content of parameter file %s: %v", filePath, err))
			continue
		}
		for key, value := range data {
			parameters[key] = value
		}
	}

	for _, a := range raw {
		key, value, err := parseParameter(a)
		if err != nil {
			errs = append(errs, *err)
			continue
		}
		parameters[key] = value
	}

	if errs != nil {
		return nil, errors.New(strings.Join(errs, ", "))
	}

	return parameters, nil
}

// parseParameter does all the parsing logic for an instance of a parameter provided to the command line
// it expects `=` as a delimiter as in key=value.  It separates keys from values as a return.   Any unexpected param will result in a
// detailed error message.
func parseParameter(raw string) (key string, param string, err *string) {

	var errMsg string
	s := strings.SplitN(raw, "=", 2)
	if len(s) < 2 {
		errMsg = fmt.Sprintf("parameter not set: %+v", raw)
	} else if s[0] == "" {
		errMsg = fmt.Sprintf("parameter name can not be empty: %+v", raw)
	} else if s[1] == "" {
		errMsg = fmt.Sprintf("parameter value can not be empty: %+v", raw)
	}
	if errMsg != "" {
		return "", "", &errMsg
	}
	return s[0], s[1], nil
}

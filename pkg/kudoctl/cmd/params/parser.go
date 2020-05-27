package params

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

// GetParameterMap takes a slice of parameter strings and a slice of parameter filenames as well as a filesystem,
// parses parameters into a map of keys and values.
// All values are marshalled into string form.
func GetParameterMap(fs afero.Fs, raw []string, filePaths []string) (map[string]string, error) {
	var errs []string

	paramsFromCmdline, errs := getParamsFromCmdline(raw, errs)
	paramsFromFiles, errs := getParamsFromFiles(fs, filePaths, errs)

	if errs != nil {
		return nil, errors.New(strings.Join(errs, ", "))
	}

	return mergeParams(paramsFromCmdline, paramsFromFiles), nil
}

func getParamsFromCmdline(raw []string, errs []string) (map[string]string, []string) {
	parameters := make(map[string]string)
	for _, a := range raw {
		key, value, err := parseParameter(a)
		if err != nil {
			errs = append(errs, *err)
			continue
		}
		parameters[key] = value
	}
	return parameters, errs
}

func getParamsFromFiles(fs afero.Fs, filePaths []string, errs []string) (map[string]string, []string) {
	parameters := make(map[string]string)
	for _, filePath := range filePaths {
		var err error
		rawData, err := afero.ReadFile(fs, filePath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("error reading from parameter file %s: %v", filePath, err))
			continue
		}

		errs = GetParametersFromFile(filePath, rawData, errs, parameters)

	}
	return parameters, errs
}

func GetParametersFromFile(filePath string, bytes []byte, errs []string, parameters map[string]string) []string {
	data := make(map[string]interface{})
	err := yaml.Unmarshal(bytes, &data)
	if err != nil {
		errs = append(errs, fmt.Sprintf("error unmarshalling content of parameter file %s: %v", filePath, err))
		return errs
	}
	clog.V(2).Printf("Unmarshalling %q...", filePath)
	for key, value := range data {
		clog.V(3).Printf("Value of parameter %q is a %T: %v", key, value, value)
		var valueType v1beta1.ParameterType
		switch value.(type) {
		case map[string]interface{}:
			valueType = v1beta1.MapValueType
		case []interface{}:
			valueType = v1beta1.ArrayValueType
		case string:
			valueType = v1beta1.StringValueType
		default:
			valueType = v1beta1.StringValueType
		}
		wrapped, err := convert.WrapParamValue(value, valueType)
		if err != nil {
			errs = append(errs, fmt.Sprintf("error converting value of parameter %s from file %s %q to a string: %v", key, filePath, value, err))
			continue
		}
		parameters[key] = *wrapped
	}
	return errs
}

func mergeParams(paramsFromCmdline map[string]string, paramsFromFiles map[string]string) map[string]string {
	params := make(map[string]string)
	for key, value := range paramsFromFiles {
		params[key] = value
	}
	// parameters specified on command line override those provided in parameter value files
	for key, value := range paramsFromCmdline {
		params[key] = value
	}
	return params
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

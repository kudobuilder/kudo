package params

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/afero"
	"sigs.k8s.io/yaml"

	kudoapi "github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/clog"
	"github.com/kudobuilder/kudo/pkg/util/convert"
)

// GetParameterMap takes a slice of parameter strings and a slice of parameter filenames as well as a filesystem,
// parses parameters into a map of keys and values.
// All values are marshalled into string form.
func GetParameterMap(fs afero.Fs, raw []string, filePaths []string) (map[string]string, error) {
	var errs []string

	paramsFromCmdline, cmdErrs := getParamsFromCmdline(raw)
	errs = append(errs, cmdErrs...)
	paramsFromFiles, fileErrs := getParamsFromFiles(fs, filePaths)
	errs = append(errs, fileErrs...)

	if errs != nil {
		return nil, errors.New(strings.Join(errs, ", "))
	}

	return mergeParams(paramsFromCmdline, paramsFromFiles), nil
}

func getParamsFromCmdline(raw []string) (map[string]string, []string) {
	var errs []string
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

func getParamsFromFiles(fs afero.Fs, filePaths []string) (map[string]string, []string) {
	var errs []string
	parameters := make(map[string]string)
	for _, filePath := range filePaths {
		rawData, err := afero.ReadFile(fs, filePath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("error reading from parameter file %s: %v", filePath, err))
			continue
		}

		err = GetParametersFromFile(filePath, rawData, parameters)
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return parameters, errs
}

func GetParametersFromFile(filePath string, bytes []byte, parameters map[string]string) error {
	data := make(map[string]interface{})
	err := yaml.Unmarshal(bytes, &data)
	if err != nil {
		return fmt.Errorf("error unmarshalling content of parameter file %s: %v", filePath, err)
	}

	clog.V(2).Printf("Unmarshalling %q...", filePath)
	var errs []string
	for key, value := range data {
		clog.V(3).Printf("Value of parameter %q is a %T: %v", key, value, value)
		var valueType kudoapi.ParameterType
		switch value.(type) {
		case map[string]interface{}:
			valueType = kudoapi.MapValueType
		case []interface{}:
			valueType = kudoapi.ArrayValueType
		case string:
			valueType = kudoapi.StringValueType
		default:
			valueType = kudoapi.StringValueType
		}
		wrapped, err := convert.WrapParamValue(value, valueType)
		if err != nil {
			errs = append(errs, fmt.Sprintf("%s: %v", key, err))
			continue
		}
		if wrapped == nil {
			errs = append(errs, fmt.Sprintf("%s has a null value (https://yaml.org/spec/1.2/spec.html#id2803362) which is currently not supported", key))
			continue
		}
		parameters[key] = *wrapped
	}
	if errs != nil {
		return fmt.Errorf("errors while unmarshaling following keys of the parameter file %s: %s", filePath, strings.Join(errs, ", "))
	}
	return nil
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

	// TODO (kensipe): this function and calling code should be refactored to NOT use `err` for strings.
	var errMsg string
	s := strings.SplitN(raw, "=", 2)
	switch {
	case len(s) < 2:
		errMsg = fmt.Sprintf("parameter not set: %+v", raw)
	case s[0] == "":
		errMsg = fmt.Sprintf("parameter name can not be empty: %+v", raw)
	case s[1] == "":
		errMsg = fmt.Sprintf("parameter value can not be empty: %+v", raw)
	}
	if errMsg != "" {
		return "", "", &errMsg
	}
	return s[0], s[1], nil
}

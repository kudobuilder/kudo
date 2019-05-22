package test

import (
	"fmt"
	"reflect"
)

// Error type used by IsSubset for tracking the path in the struct.
type SubsetError struct {
	path    []string
	message string
}

// Append key to the existing struct path. For example, in struct member `a.Key1.Key2`, the path would be ["Key1", "Key2"]
func (e *SubsetError) AppendPath(key string) {
	if e.path == nil {
		e.path = []string{}
	}

	e.path = append(e.path, key)
}

// Implement the error interface.
func (e *SubsetError) Error() string {
	if e.path == nil || len(e.path) == 0 {
		return e.message
	}

	path := ""
	for i := len(e.path) - 1; i >= 0; i-- {
		path = fmt.Sprintf("%s.%s", path, e.path[i])
	}

	return fmt.Sprintf("%s: %s", path, e.message)
}

// Check to see if `expected` is a subset of `actual`. A "subset" is an object that is equivalent to
// the other object, but where map keys found in actual that are not defined in expected are ignored.
func IsSubset(expected, actual interface{}) error {
	if reflect.TypeOf(expected) != reflect.TypeOf(actual) {
		return &SubsetError{
			message: fmt.Sprintf("type mismatch: %v != %v", reflect.TypeOf(expected), reflect.TypeOf(actual)),
		}
	}

	if reflect.DeepEqual(expected, actual) {
		return nil
	}

	if reflect.TypeOf(expected).Kind() == reflect.Slice {
		if reflect.ValueOf(expected).Len() != reflect.ValueOf(actual).Len() {
			return &SubsetError{
				message: fmt.Sprintf("slice length mismatch: %d != %d", reflect.ValueOf(expected).Len(), reflect.ValueOf(actual).Len()),
			}
		}

		for i := 0; i < reflect.ValueOf(expected).Len(); i++ {
			if err := IsSubset(reflect.ValueOf(expected).Index(i).Interface(), reflect.ValueOf(actual).Index(i).Interface()); err != nil {
				return err
			}
		}
	} else if reflect.TypeOf(expected).Kind() == reflect.Map {
		iter := reflect.ValueOf(expected).MapRange()

		for iter.Next() {
			actualValue := reflect.ValueOf(actual).MapIndex(iter.Key())

			if !actualValue.IsValid() {
				return &SubsetError{
					path:    []string{iter.Key().String()},
					message: fmt.Sprintf("key is missing from map"),
				}
			}

			if err := IsSubset(iter.Value().Interface(), actualValue.Interface()); err != nil {
				subsetErr := err.(*SubsetError)
				subsetErr.AppendPath(iter.Key().String())
				return subsetErr
			}
		}
	} else {
		return &SubsetError{
			message: fmt.Sprintf("value mismatch: %v != %v", expected, actual),
		}
	}

	return nil
}

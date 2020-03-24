package diagnostics

import "strings"

type multiError struct {
	errs []error
}

func appendError(m *multiError, err error) *multiError {
	if err == nil {
		return m
	}
	if m == nil {
		m = new(multiError)
	}
	m.errs = append(m.errs, err)
	return m
}

func (e *multiError) Error() string {
	var errs []string
	for _, err := range e.errs {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return strings.Join(errs, "\n")
}

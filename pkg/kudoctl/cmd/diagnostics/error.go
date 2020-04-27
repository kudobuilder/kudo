package diagnostics

import "strings"

type MultiError struct {
	errs []error
}

func AppendError(m *MultiError, err error) *MultiError {
	if err == nil {
		return m
	}
	if m == nil {
		m = new(MultiError)
	}
	m.errs = append(m.errs, err)
	return m
}

func (e *MultiError) Error() string {
	var errs []string
	for _, err := range e.errs {
		if err != nil {
			errs = append(errs, err.Error())
		}
	}
	return strings.Join(errs, "\n")
}

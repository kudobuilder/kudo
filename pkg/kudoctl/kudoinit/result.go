package kudoinit

import (
	"bytes"
	"fmt"
	"io"

	"github.com/gosuri/uitable"
)

// Result holds the errors and warnings of a package verification
type Result struct {
	Errors   []string
	Warnings []string
}

// NewResult initializes the Result struct
func NewResult() Result {
	return Result{
		Errors:   []string{},
		Warnings: []string{},
	}
}

func NewError(err ...string) Result {
	result := NewResult()
	result.AddErrors(err...)
	return result
}

func (vr *Result) Error() string {
	buf := new(bytes.Buffer)
	vr.printErrorsWithoutHeader(buf)
	return buf.String()
}

// AddErrors adds an arbitrary error string to the verification result
func (vr *Result) AddErrors(err ...string) { vr.Errors = append(vr.Errors, err...) }

// AddWarnings adds an arbitrary warning string to the verification result
func (vr *Result) AddWarnings(wrn ...string) { vr.Warnings = append(vr.Warnings, wrn...) }

// Merge method merges the errors and warnings from two verification results
func (vr *Result) Merge(other Result) {
	vr.AddErrors(other.Errors...)
	vr.AddWarnings(other.Warnings...)
}

// IsValid returns true if verification result does not have errors
func (vr *Result) IsValid() bool { return len(vr.Errors) == 0 }

func (vr *Result) IsError() bool { return len(vr.Errors) > 0 }

// PrintErrors is a simple printer for errors in Error context
func (vr *Result) printErrorsWithoutHeader(out io.Writer) {
	if len(vr.Errors) == 0 {
		return
	}
	table := uitable.New()
	for _, err := range vr.Errors {
		table.AddRow(err)
	}
	fmt.Fprintln(out, table) //nolint:errcheck
}

// PrintErrors is a pretty printer for errors
func (vr *Result) PrintErrors(out io.Writer) {
	if len(vr.Errors) == 0 {
		return
	}
	table := uitable.New()
	table.AddRow("Errors")
	for _, err := range vr.Errors {
		table.AddRow(err)
	}
	fmt.Fprintln(out, table) //nolint:errcheck
}

// PrintWarnings is a pretty printer for warnings
func (vr *Result) PrintWarnings(out io.Writer) {
	if len(vr.Warnings) == 0 {
		return
	}
	table := uitable.New()
	table.AddRow("Warnings")
	for _, warning := range vr.Warnings {
		table.AddRow(warning)
	}
	fmt.Fprintln(out, table) //nolint:errcheck
}

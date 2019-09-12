package utils

import (
	"io"
	"os"
	"regexp"
	"runtime/pprof"
	"testing"
)

// RunTests runs a Go test method without requiring the Go compiler.
// This does not currently support test caching.
// If testToRun is set to a non-empty string, it is passed as a `-run` argument to the go test harness.
// If paralellism is set, it limits the number of concurrently running tests.
func RunTests(testName string, testToRun string, parallelism int, testFunc func(*testing.T)) {
	// TODO: set testing flags. Using 'flags.Set' doesn't work.
	os.Exit(testing.MainStart(&testDeps{}, []testing.InternalTest{
		{
			Name: testName,
			F:    testFunc,
		},
	}, nil, nil).Run())
}

// testDeps implements the testDeps interface for MainStart.
type testDeps struct{}

var matchPat string
var matchRe *regexp.Regexp

func (testDeps) MatchString(pat, str string) (result bool, err error) {
	if matchRe == nil || matchPat != pat {
		matchPat = pat
		matchRe, err = regexp.Compile(matchPat)

		if err != nil {
			return
		}
	}

	return matchRe.MatchString(str), nil
}

func (testDeps) StartCPUProfile(w io.Writer) error {
	return pprof.StartCPUProfile(w)
}

func (testDeps) StopCPUProfile() {
	pprof.StopCPUProfile()
}

func (testDeps) WriteProfileTo(name string, w io.Writer, debug int) error {
	return pprof.Lookup(name).WriteTo(w, debug)
}

func (testDeps) ImportPath() string {
	return ""
}

func (testDeps) StartTestLog(w io.Writer) {}

func (testDeps) StopTestLog() error {
	return nil
}

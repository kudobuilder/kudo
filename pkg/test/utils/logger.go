package utils

import (
	"fmt"
	"testing"
	"time"
)

// Logger is an interface used by the KUDO test operator to provide logging of tests.
type Logger interface {
	Log(args ...interface{})
	Logf(format string, args ...interface{})
	WithPrefix(string) Logger
}

// TestLogger implements the Logger interface to be compatible with the go test operator's
// output buffering (without this, the use of Parallel tests combined with subtests causes test
// output to be mixed).
type TestLogger struct {
	prefix string
	test   *testing.T
}

// NewTestLogger creates a new test logger.
func NewTestLogger(test *testing.T, prefix string) *TestLogger {
	return &TestLogger{
		prefix: prefix,
		test:   test,
	}
}

// Log logs the provided arguments with the logger's prefix. See testing.Log for more details.
func (t *TestLogger) Log(args ...interface{}) {
	args = append([]interface{}{
		fmt.Sprintf("%s | %s |", time.Now().Format("15:04:05"), t.prefix),
	}, args...)
	t.test.Log(args...)
}

// Logf logs the provided arguments with the logger's prefix. See testing.Logf for more details.
func (t *TestLogger) Logf(format string, args ...interface{}) {
	t.Log(fmt.Sprintf(format, args...))
}

// WithPrefix returns a new TestLogger with the provided prefix appended to the current prefix.
func (t *TestLogger) WithPrefix(prefix string) Logger {
	return NewTestLogger(t.test, fmt.Sprintf("%s/%s", t.prefix, prefix))
}

package clog

import (
	"fmt"
	"io"
	"os"
	"strconv"

	"github.com/spf13/pflag"
)

// this package provides (C)lient logs, or CLI logs. Although this is called a log, it is for writing to STDOUT and is designed
// for CLI output (not to log files etc.). We want -V behavior familiar to kubectl users, but klog comes with a lot of dependencies
// and global flags that obfuscate KUDO.  glog is built more for a multi-threaded logger (managing buffers etc) which
// KUDO CLI doesn't have a need for.  clog provides verbosity control for CLI output.

// guidance for use of V level
//  0-1 normal standard out
//  2-4 as debug-level logs
//  5-6 logical chooses
//  7-8 input/output details
//  9-10 as trace-level (http details)

// Level specifies a level of verbosity for V logs.
type Level int8

// get returns the value of the Level.
func (l *Level) get() Level {
	return *l
}

// set sets the value of the Level.
func (l *Level) set(val Level) {
	*l = val
}

// Get is required for https://godoc.org/flag#Getter
// the return of interface{} is forced by Getter interface
func (l *Level) Get() interface{} {
	return *l
}

// Required for pflag.Value defined: https://godoc.org/github.com/spf13/pflag#Value

// String is part of the flag.Value interface.
func (l *Level) String() string {
	return strconv.FormatInt(int64(*l), 10)
}

// Set is part of the flag.Value interface.
func (l *Level) Set(value string) error {
	v, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	l.set(Level(v))
	return nil
}

// Type is part of flag.Value interface
func (l *Level) Type() string {
	return fmt.Sprint(*l)
}

// pflag.Value interface ends

type loggingT struct {
	verbosity Level // V logging level, the value of the -v flag/
	out       io.Writer
}

func (l *loggingT) printf(format string, args ...interface{}) {
	fmt.Fprintf(l.out, format, args...)
	fmt.Fprintf(l.out, "\n")
}

var logging loggingT

// Verbose is boolean type that implements Println, Printf
// See glog and documentation for V
type Verbose bool

// V reports true if the verbosity at the call site is at least the request level.
// This the following glog style code samples are possible:
//
//  if clog.V(2) { clog.Print("log this") }
// or
//
//  clog.V(2).Print("log this")
//
// Whether the call site logs is determined by the `-v` flags.
func V(level Level) Verbose {
	// This function tries hard to be cheap unless there's work to do.
	// The fast path is two atomic loads and compares.

	// Here is a cheap but safe test to see if V logging is enabled globally.
	if logging.verbosity.get() >= level {
		return Verbose(true)
	}
	return Verbose(false)
}

// Printf is equivalent to the global Printf function, guarded by the value of v.
// See the documentation of V for usage.
func (v Verbose) Printf(format string, args ...interface{}) {
	if v {
		logging.printf(format, args...)
	}
}

// InitWithFlags allows for the initialization of log via root command
func InitWithFlags(f *pflag.FlagSet, out io.Writer) {
	// allows for initialization of writer in testing without CLI flags
	if f != nil {
		f.VarP(&logging.verbosity, "v", "v", "Log level for V logs")
	}
	logging.out = out
}

// InitNoFlag initializes without CLI flag which means the level must be initialized
// useful for tests
func InitNoFlag(out io.Writer, level Level) {
	logging.verbosity.set(level)
	logging.out = out
}

// Printf provides default level printing for things that will always print
func Printf(format string, args ...interface{}) {
	V(0).Printf(format, args...)
}

// Errorf formats and returns error and logs at level 2
func Errorf(format string, a ...interface{}) error {
	err := fmt.Errorf(format, a...)
	V(2).Printf(err.Error())
	return err
}

func init() {
	// expected to be overridden with InitWithFlags().  This simplifies testing and default behavior
	logging.out = os.Stdout
}

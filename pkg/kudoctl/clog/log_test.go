package clog

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevelCheck(t *testing.T) {

	var buf bytes.Buffer

	// verbosity set to 4 -v=4
	logging.verbosity = Level(4)
	defer func() { logging.verbosity = Level(0) }()
	logging.out = &buf

	// 3 prints with newline
	V(3).Printf("level 3")
	assert.Equal(t, "level 3\n", buf.String())
	buf.Reset()

	// 4 prints with newline
	V(4).Printf("level 4")
	assert.Equal(t, "level 4\n", buf.String())
	buf.Reset()

	// 5  does not print (v==4)
	V(5).Printf("level 5")
	assert.Equal(t, "", buf.String())
}

func TestDefaultPrintLevel(t *testing.T) {

	//default is verbosity of 0 -v=0 or not supplied
	var buf bytes.Buffer

	logging.out = &buf

	// 3 does not print
	V(3).Printf("level 3")
	assert.Equal(t, "", buf.String())
	buf.Reset()

	V(0).Printf("level 0")
	assert.Equal(t, "level 0\n", buf.String())
	buf.Reset()

	// the clog.Printf defaults to level 0
	Printf("level 0 check")
	assert.Equal(t, "level 0 check\n", buf.String())
}

func TestErrorf(t *testing.T) {
	//default is verbosity of 0 -v=0 or not supplied
	var buf bytes.Buffer

	logging.verbosity = Level(0)
	logging.out = &buf

	// error f prints at level 2, no output by default
	Errorf("error msg") //nolint:errcheck
	assert.Equal(t, "", buf.String())
	buf.Reset()

	logging.verbosity = Level(2)
	defer func() { logging.verbosity = Level(0) }()
	Errorf("error msg") //nolint:errcheck
	assert.Equal(t, "error msg\n", buf.String())
}

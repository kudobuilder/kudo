package task

// This file has been adopted from kubectl/pkg/generate/versioned/env_file.go and used to read files containing
// env var pairs in pipe-tasks.

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"k8s.io/apimachinery/pkg/util/validation"
)

var utf8bom = []byte{0xEF, 0xBB, 0xBF}

// proccessEnvFileLine returns a blank key if the line is empty or a comment.
// The value will be retrieved from the environment if necessary.
func proccessEnvFileLine(line []byte, currentLine int) (key, value string, err error) {

	if !utf8.Valid(line) {
		return ``, ``, fmt.Errorf("invalid utf8 bytes at line %d: %v", currentLine+1, line)
	}

	// We trim UTF8 BOM from the first line of the file but no others
	if currentLine == 0 {
		line = bytes.TrimPrefix(line, utf8bom)
	}

	// trim the line from all leading whitespace first
	line = bytes.TrimLeftFunc(line, unicode.IsSpace)

	// If the line is empty or a comment, we return a blank key/value pair.
	if len(line) == 0 || line[0] == '#' {
		return ``, ``, nil
	}

	data := strings.SplitN(string(line), "=", 2)
	if len(data) != 2 {
		return ``, ``, fmt.Errorf("%q is not a valid env var definition (KEY=VAL)", line)
	}

	key = data[0]
	if errs := validation.IsEnvVarName(key); len(errs) != 0 {
		return ``, ``, fmt.Errorf("%q is not a valid key name: %s", key, strings.Join(errs, ";"))
	}
	value = data[1]

	return key, value, nil
}

// addFromEnvFile processes an env file allows a generic addTo to handle the
// collection of key value pairs or returns an error.
func addFromEnvFile(data []byte, addTo func(key, value string)) error {
	r := bytes.NewReader(data)
	scanner := bufio.NewScanner(r)
	currentLine := 0
	for scanner.Scan() {
		// Process the current line, retrieving a key/value pair if possible.
		scannedBytes := scanner.Bytes()
		key, value, err := proccessEnvFileLine(scannedBytes, currentLine)
		if err != nil {
			return err
		}
		currentLine++

		if len(key) == 0 {
			// no key means line was empty or a comment
			continue
		}

		addTo(key, value)
	}
	return nil
}

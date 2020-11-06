package params

import (
	"os"
	"testing"

	"github.com/spf13/afero"
	"github.com/stretchr/testify/assert"
)

var parameterParsingTests = []struct {
	paramStr string
	key      string
	value    string
	err      string
}{
	{"foo", "", "", "parameter not set: foo"},
	{"foo=", "", "", "parameter value can not be empty: foo="},
	{"=bar", "", "", "parameter name can not be empty: =bar"},
	{"foo=bar", "foo", "bar", ""},
}

func TestTableParameterParsing(t *testing.T) {
	for _, test := range parameterParsingTests {
		key, value, err := parseParameter(test.paramStr)
		assert.Equal(t, key, test.key)
		assert.Equal(t, value, test.value)
		if err != nil {
			assert.Equal(t, *err, test.err)
		}
	}
}

func TestGetParameterMap(t *testing.T) {
	emptyMap := make(map[string]string)
	tests := []struct {
		name          string
		params        []string
		paramFiles    []string
		fsContent     map[string]string
		expectedError string
		expected      map[string]string
	}{
		{
			"no params",
			[]string{},
			[]string{},
			nil,
			"",
			emptyMap,
		},
		{
			"no files",
			[]string{"a=1", "b=2", "a=3"},
			[]string{},
			nil,
			"",
			map[string]string{
				"a": "3",
				"b": "2",
			},
		},
		{
			"files only",
			[]string{},
			[]string{"file-a", "file-b"},
			map[string]string{
				"file-a": "a: 42\nb: 43\n",
				"file-b": "a: 742\nc: [7, 7, 7]\n",
			},
			"",
			map[string]string{
				"a": "742",
				"b": "43",
				"c": "- 7\n- 7\n- 7\n",
			},
		},
		{
			"files only in reverse order",
			[]string{},
			[]string{"file-b", "file-a"},
			map[string]string{
				"file-a": "a: 42\nb: 43\n",
				"file-b": "a: 742\nc: 777\n",
			},
			"",
			map[string]string{
				"a": "42",
				"b": "43",
				"c": "777",
			},
		},
		{
			"file overridden with cmdline param",
			[]string{"a=1", "c=2"},
			[]string{"file-a"},
			map[string]string{
				"file-a": "a: 42\nb: 43\n",
			},
			"",
			map[string]string{
				"a": "1",
				"b": "43",
				"c": "2",
			},
		},
		{
			"missing file",
			[]string{"a=1"},
			[]string{"missing-file"},
			nil,
			"error reading from parameter file missing-file: open missing-file: file does not exist",
			nil,
		},
		{
			"regression test for #1437",
			nil,
			[]string{"param-file"},
			map[string]string{
				"param-file": "A:\n- foo: bar\n",
			},
			"",
			map[string]string{
				"A": "- foo: bar\n",
			},
		},
		{
			"regression test for #1602",
			nil,
			[]string{"param-file"},
			map[string]string{
				"param-file": "a:\nb: 1\n",
			},
			"errors while unmarshaling following keys of the parameter file param-file: a has a null value (https://yaml.org/spec/1.2/spec.html#id2803362) which is currently not supported",
			nil,
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			fs := afero.NewMemMapFs()
			for fileName, content := range test.fsContent {
				assert.NoError(t, afero.WriteFile(fs, fileName, []byte(content), os.ModePerm))
			}
			params, err := GetParameterMap(fs, test.params, test.paramFiles)
			if len(test.expectedError) == 0 {
				assert.NoError(t, err)
				assert.Equal(t, test.expected, params)
			} else if assert.Error(t, err) {
				assert.Equal(t, test.expectedError, err.Error())
			}
		})
	}
}

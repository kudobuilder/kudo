package renderer

import (
	"strings"
	"testing"
	"text/template"

	"github.com/stretchr/testify/assert"
)

func TestToYAML(t *testing.T) {
	var b strings.Builder
	values := map[string]interface{}{"foo": "bar"}
	err := template.Must(template.New("test").Funcs(funcMap()).Parse("{{ toYaml . }}")).Execute(&b, values)

	assert.NoError(t, err)
	assert.Equal(t, `foo: bar`, b.String(), "ToYaml")
}

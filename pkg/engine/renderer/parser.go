package renderer

import (
	"bytes"
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

const yamlSeparator = "\n---"

// YamlToObject parses a list of runtime.Objects from the provided yaml
// If the type is not known in the scheme, it tries to parse it as Unstructured
// We used to use 'apimachiner/pkg/util/yaml' for splitting the input into multiple yamls,
// however under the covers it uses bufio.NewScanner with token defaults with no option to modify.
// see: https://github.com/kubernetes/apimachinery/blob/release-1.6/pkg/util/yaml/decoder.go#L94
// The YAML input can be too large for the default scan token size used by these packages.
// For more detail read: https://github.com/kudobuilder/kudo/pull/1400
// TODO(av) could we use something else than a global scheme here? Should we somehow inject it?
func YamlToObject(yaml string) (objs []runtime.Object, err error) {
	yamls := strings.Split(yaml, yamlSeparator)
	for _, y := range yamls {
		if len(strings.TrimSpace(y)) == 0 {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, _, e := decode([]byte(y), nil, nil)

		if e != nil {
			// if parsing to scheme known types fails, just try to parse into unstructured
			unstructuredObj := &unstructured.Unstructured{}
			fileBytes := []byte(y)
			decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewBuffer(fileBytes), len(fileBytes))
			if err = decoder.Decode(unstructuredObj); err != nil {
				return nil, fmt.Errorf("decoding chunk %q failed: %v", fileBytes, err)
			}

			// Skip those chunks/documents which (after rendering) consist solely of whitespace or comments.
			if len(unstructuredObj.UnstructuredContent()) != 0 {
				objs = append(objs, unstructuredObj)
			}
		} else {
			objs = append(objs, obj)
		}
	}
	return
}

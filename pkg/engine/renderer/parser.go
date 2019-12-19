package renderer

import (
	"bytes"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	yamlutil "k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/client-go/kubernetes/scheme"
)

// YamlToObject parses a list of runtime.Objects from the provided yaml
// If the type is not known in the scheme, it tries to parse it as Unstructured
// TODO(av) could we use something else than a global scheme here? Should we somehow inject it?
func YamlToObject(yaml string) (objs []runtime.Object, err error) {
	sepYamlfiles := strings.Split(yaml, "---")
	for _, f := range sepYamlfiles {
		if f == "\n" || f == "" {
			// ignore empty cases
			continue
		}

		decode := scheme.Codecs.UniversalDeserializer().Decode
		obj, gvk, e := decode([]byte(f), nil, nil)

		if e != nil {
			// if parsing to scheme known types fails, just try to parse into unstructured
			unstructuredObj := &unstructured.Unstructured{}
			fileBytes := []byte(f)
			decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewBuffer(fileBytes), len(fileBytes))
			if err = decoder.Decode(unstructuredObj); err != nil {
				return nil, err
			}
			objs = append(objs, unstructuredObj)
		} else {
			if gvk != nil {
				obj.GetObjectKind().SetGroupVersionKind(*gvk)
			}
			objs = append(objs, obj)
		}
	}
	return
}

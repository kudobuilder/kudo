package renderer

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
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
	decode := scheme.Codecs.UniversalDeserializer().Decode
	// There can be any number of chunks in a YAML file, separated with "\n---".
	// This "decoder" returns one chunk at a time.
	docDecoder := yamlutil.NewDocumentDecoder(ioutil.NopCloser(strings.NewReader(yaml)))
	for {
		// Prepare slice large enough from the start, rather than do the ErrShortBuffer dance,
		// since we're reading from an in-memory string already anyway.
		chunk := make([]byte, len(yaml))
		n, err := docDecoder.Read(chunk)
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("reading from document decoder failed: %v", err)
		}
		// Truncate to what was actually read, to not confuse the consumers with NUL bytes.
		chunk = chunk[:n]

		obj, _, e := decode(chunk, nil, nil)

		if e != nil {
			// if parsing to scheme known types fails, just try to parse into unstructured
			unstructuredObj := &unstructured.Unstructured{}
			decoder := yamlutil.NewYAMLOrJSONDecoder(bytes.NewBuffer(chunk), n)
			if err = decoder.Decode(unstructuredObj); err != nil {
				return nil, fmt.Errorf("decoding chunk %q failed: %v", chunk, err)
			}
			objs = append(objs, unstructuredObj)
		} else {
			objs = append(objs, obj)
		}
	}
	return
}

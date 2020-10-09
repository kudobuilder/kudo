package convert

import (
	"context"

	"github.com/qri-io/jsonpointer"
	"github.com/qri-io/jsonschema"
)

// Advanced keyword
type IsAdvanced bool

// newIsAdvanced is a jsonschama.KeyMaker
func newIsAdvanced() jsonschema.Keyword {
	return new(IsAdvanced)
}

// Register implements jsonschema.Keyword
func (f *IsAdvanced) Register(uri string, registry *jsonschema.SchemaRegistry) {}

// Resolve implements jsonschema.Keyword
func (f *IsAdvanced) Resolve(pointer jsonpointer.Pointer, uri string) *jsonschema.Schema {
	return nil
}

// ValidateKeyword implements jsonschema.Keyword
func (f *IsAdvanced) ValidateKeyword(ctx context.Context, currentState *jsonschema.ValidationState, data interface{}) {
	isAdvanced, _ := data.(bool)
	if isAdvanced {

		currentState.Local.HasKeyword("default")

	}
}

func init() {
	jsonschema.LoadDraft2019_09()
	jsonschema.RegisterKeyword("advanced", newIsAdvanced)
}

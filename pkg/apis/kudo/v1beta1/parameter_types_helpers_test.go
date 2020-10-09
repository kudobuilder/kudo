/*

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetChangedParamDefs(t *testing.T) {
	bTrue := true
	cmpImmutable := func(p1, p2 Parameter) bool {
		return p1.Immutable == p2.Immutable
	}

	tests := []struct {
		name string
		old  []Parameter
		new  []Parameter
		want []Parameter
		cmp  func(p1, p2 Parameter) bool
	}{
		{
			name: "no changes with empty",
			old:  []Parameter{},
			new:  []Parameter{},
			want: []Parameter{},
			cmp:  cmpImmutable,
		},
		{
			name: "no changes with content",
			old:  []Parameter{{Name: "p"}},
			new:  []Parameter{{Name: "p"}},
			want: []Parameter{},
			cmp:  cmpImmutable,
		},
		{
			name: "changed attribute",
			old:  []Parameter{{Name: "p"}},
			new:  []Parameter{{Name: "p", Immutable: &bTrue}},
			want: []Parameter{{Name: "p", Immutable: &bTrue}},
			cmp:  cmpImmutable,
		},
		{
			name: "changed other attribute does not register",
			old:  []Parameter{{Name: "p"}},
			new:  []Parameter{{Name: "p", Required: &bTrue}},
			want: []Parameter{},
			cmp:  cmpImmutable,
		},
		{
			name: "added and removed params do not show up",
			old:  []Parameter{{Name: "p"}},
			new:  []Parameter{{Name: "p1"}},
			want: []Parameter{},
			cmp:  cmpImmutable,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			changedDefs := GetChangedParamDefs(tt.old, tt.new, tt.cmp)
			assert.Equal(t, tt.want, changedDefs)
		})
	}
}

func TestGetAddedRemovedParamDefs(t *testing.T) {
	bTrue := true

	tests := []struct {
		name        string
		old         []Parameter
		new         []Parameter
		wantAdded   []Parameter
		wantRemoved []Parameter
	}{
		{
			name:        "no result when empty",
			old:         []Parameter{},
			new:         []Parameter{},
			wantAdded:   []Parameter{},
			wantRemoved: []Parameter{},
		},
		{
			name:        "no result when no changes",
			old:         []Parameter{{Name: "p"}},
			new:         []Parameter{{Name: "p"}},
			wantAdded:   []Parameter{},
			wantRemoved: []Parameter{},
		},
		{
			name:        "added/removed param show up in correct direct",
			old:         []Parameter{{Name: "p"}},
			new:         []Parameter{{Name: "p"}, {Name: "p1"}},
			wantAdded:   []Parameter{{Name: "p1"}},
			wantRemoved: []Parameter{},
		},
		{
			name:        "added param does not show up in other direction",
			old:         []Parameter{{Name: "p"}, {Name: "p1"}},
			new:         []Parameter{{Name: "p"}},
			wantAdded:   []Parameter{},
			wantRemoved: []Parameter{{Name: "p1"}},
		},
		{
			name:        "changed attribute does not register",
			old:         []Parameter{{Name: "p"}},
			new:         []Parameter{{Name: "p", Required: &bTrue}},
			wantAdded:   []Parameter{},
			wantRemoved: []Parameter{},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			addedDefs := GetAddedParameters(tt.old, tt.new)
			removedDefs := GetRemovedParamDefs(tt.old, tt.new)
			assert.Equal(t, tt.wantAdded, addedDefs)
			assert.Equal(t, tt.wantRemoved, removedDefs)
		})
	}
}

func TestValidateType(t *testing.T) {

	tests := []struct {
		name        string
		pValue      interface{}
		pType       ParameterType
		expectedErr bool
	}{
		{
			name:   "simple int",
			pValue: 23,
			pType:  IntegerValueType,
		},
		{
			name:   "int8",
			pValue: int8(23),
			pType:  IntegerValueType,
		},
		{
			name:   "int64",
			pValue: int64(23),
			pType:  IntegerValueType,
		},
		{
			name:   "intAsString",
			pValue: "42",
			pType:  IntegerValueType,
		},
		{
			name:   "float32",
			pValue: float32(3.14),
			pType:  NumberValueType,
		},
		{
			name:   "float64",
			pValue: float64(3.1415),
			pType:  NumberValueType,
		},
		{
			name:   "floatAsString",
			pValue: "3.1415",
			pType:  NumberValueType,
		},
		{
			name:   "bool",
			pValue: true,
			pType:  BooleanValueType,
		},
		{
			name:   "boolAsString",
			pValue: "true",
			pType:  BooleanValueType,
		},
		{
			name:   "array",
			pValue: []string{"oneString", "twoString"},
			pType:  ArrayValueType,
		},
		{
			name:   "arrayAsString",
			pValue: `[ "oneString", "twoString" ]`,
			pType:  ArrayValueType,
		},
		{
			name:   "map",
			pValue: map[string]string{"oneString": "oneValue", "twoString": "twoValue"},
			pType:  MapValueType,
		},
		{
			name:   "mapAsString",
			pValue: `{ "oneString": "oneValue", "twoString": "twoValue" }`,
			pType:  MapValueType,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateParameterValueForType(tt.pType, tt.pValue)

			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

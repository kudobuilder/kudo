// Copyright 2019
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package main

// NOT part of the KUDO distribution
// Tooling for generating KUDO schemas

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/kudobuilder/kudo/pkg/apis/kudo/v1alpha1"
	"github.com/kudobuilder/kudo/pkg/kudoctl/packages"
	"github.com/kudobuilder/kudo/pkg/kudoctl/util/repo"

	"github.com/alecthomas/jsonschema"
)

// Params need to create params schema
// todo: replace with in pkg code struct once we have agreement
type Params struct {
	Params map[string]v1alpha1.Parameter `json:"params"`
}

func main() {

	// arg[0] is program, 2 args means 1 arg was passed :)
	if len(os.Args) != 2 {
		fmt.Println("schema-gen requires 1 argument which is the path to generate the schemas")
		os.Exit(-1)
	}
	path := os.Args[1]
	fi, err := os.Stat(path)
	if err != nil {
		fmt.Println(err)
	}
	if !fi.IsDir() {
		fmt.Printf("%v is not a directory", path)
	}
	r := jsonschema.Reflector{
		AllowAdditionalProperties:  false,
		RequiredFromJSONSchemaTags: false,
		ExpandedStruct:             true,
		IgnoredTypes:               []interface{}{v1alpha1.OperatorDependency{}},
		TypeMapper:                 nil,
	}

	// index file
	indexPath := filepath.Join(path, "index-file.schema")
	err = writeSchema(indexPath, r, &repo.IndexFile{})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// operator file
	operatorPath := filepath.Join(path, "operator.schema")
	err = writeSchema(operatorPath, r, &packages.Operator{})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	// params file
	paramsPath := filepath.Join(path, "params.schema")
	err = writeSchema(paramsPath, r, &Params{})
	if err != nil {
		fmt.Println(err)
		os.Exit(-1)
	}

	fmt.Println("schemas created.")
}

func writeSchema(file string, r jsonschema.Reflector, v interface{}) error {
	schema := r.Reflect(v)
	j, _ := json.MarshalIndent(schema, "", "  ")
	return ioutil.WriteFile(file, j, 0755)
}

This package contains the KUDO API.

It implements a Hub and Spokes model as Kubernets internally. See https://book.kubebuilder.io/multiversion-tutorial/conversion-concepts.html for details.


- This Folder contains the hub classes. They do not contain an json annotations, as they are never serialized.

- The subfolders (v1beta1, v1beta2) contain the actual APIs.

To add a new API version:
- Copy an existing api folder (for example v1beta2 to v1beta3)
- Adjust the new API structures
- Adjust CRD storage attribute:
  - Set or remove the `// +kubebuilder:storageversion` annotations. Each CRD can have this annotation in only one package
- Adjust CRD served attribute:
  - Set or remove the `// +kubebuilder:unservedversion` annotations
- Adjust the hub structures
- run `./hack/update_codegen.sh` to generate new conversion code
  - The Conversion Generator may print some messages for missing conversions:
  ```
  Generating conversions
    E0831 14:47:50.231077   51587 conversion.go:755] Warning: could not find nor generate a final Conversion function for github.com/kudobuilder/kudo/pkg/apis/kudo/v1beta1.InstanceSpec -> github.com/kudobuilder/kudo/pkg/apis/kudo.InstanceSpec
    E0831 14:47:50.231306   51587 conversion.go:756]   the following fields need manual conversion:
    E0831 14:47:50.231315   51587 conversion.go:758]       - Parameters
  ```
  Implement the missing conversions in `conversion.go` in the respective package and re-run `update_codegen.sh`. See https://github.com/kubernetes/code-generator/blob/master/cmd/conversion-gen/main.go for details.
  
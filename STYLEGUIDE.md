# Style Guide

In an effort to encourage clear, clean, uniform and readable code it is useful to have a style guide to establish standards and expectations for code and artifact files.

KUDO is a [Kubernetes (K8S)](https://kubernetes.io/) project written primarily in [Golang](https://golang.org/), both of which have some differences in their style and expectation of code and APIs. In addition the founders of KUDO have some of their own preferences. This guide captures the preferred standards of this project.

## Golang

When writing Golang code, We favor Golang idioms over Kubernetes style. Worth reading:

* [Effective Go](https://golang.org/doc/effective_go.html)
* [Idiomatic Go](https://dmitri.shuralyov.com/idiomatic-go)
* [Idiomatic Go Resources](https://medium.com/@dgryski/idiomatic-go-resources-966535376dba)
* [Go for Industrial Programming](https://peter.bourgon.org/go-for-industrial-programming/)
* [A theory of modern Go](https://peter.bourgon.org/blog/2017/06/09/theory-of-modern-go.html)
* [Go best practices, six years in](https://peter.bourgon.org/go-best-practices-2016/)
* [What's in a name?](https://talks.golang.org/2014/names.slide#1)

Here are some common cases / deviations:

### Lint and Code Checks

All code should pass the linter. For cases of intentional lint deviation, it is expected that:

* The linter is configured with the new rule.
* The linter is configured to ignore the case.
* The case is documented in code.

`make lint`

All code should pass [staticcheck](http://staticcheck.io/).

`make staticcheck`

All code should pass `go vet`.

`make vet`


### import

The general Golang approach is to have a line of separation between Golang libraries and external packages. We prefer to have an additional line of separation grouping Kubernetes packages, and kudo packages grouped separately at the end. Example:

```
import (
	// standard library packages
	"context"
	"fmt"

	// third-party library packages that are *not* k8s
	"github.com/go-logr/logr"
	"github.com/onsi/gomega"

	// k8s library packages
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/kustomize/k8sdeps/kunstruct"
	"sigs.k8s.io/kustomize/pkg/target"
	ktypes "sigs.k8s.io/kustomize/pkg/types"
	
	// kudo packages
	"github.com/kudobuilder/kudo/pkg/util/kudo"
	"github.com/kudobuilder/kudo/pkg/version"
)
```

Executing `make import` will help you *somewhat*, but it will *not* do everything. It:
1. makes sure imports are sorted *within* each section, and
1. if an import that should be in the first (standard library) and last (kudo) section - but is somewhere else - then it will move it out of that section, but not necessarily into the correct one. as appropriate.

So you still need to manually keep the k8s imports separate from other 3rd-party imports.

### Naming

In general, naming should follow [Go conventions](https://golang.org/doc/effective_go.html#names) over [Kubernetes code conventions](https://github.com/kubernetes/community/blob/master/contributors/guide/coding-conventions.md#code-conventions).

---
title: Framework repository
type: docs
weight: 4
---
# Framework repository

KUDO CLI comes with built-in official repository of verified frameworks. Every time you use `kudo install ...` command, it pulls the package from this repository. 

## Repository implementation

KUDO can work with any repository exposed over HTTP that conforms the expected structure. The official repository is hosted on Google Cloud Storage.

In the root of the repository we expect `index.yaml` file similar to the following example:

```yaml
apiVersion: v1
entries:
  yourframework:
  - apiVersion: v1alpha1
    appVersion: 7.0.0
    name: yourframework
    urls:
    - https://kudo-repository.storage.googleapis.com/elastic-0.1.0.tgz
    version: 0.1.0
```

The url leads to a location where the tarball package is hosted. It could be internal as well as external url (inside that repository or outside). 

## How to add new package

All official packages right now are mirrored from the [github repository](https://github.com/kudobuilder/frameworks). To add new framework, create a PR against that repo.

## How to update package

The process here is the same as for adding new package. You need to create PR against the [github repository](https://github.com/kudobuilder/frameworks).
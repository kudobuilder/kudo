## Diagnostics Bundle with Sonobuoy  

1. Build the Docker image `docker build -f Plugin.Dockerfile -t vvy/easy-sonobuoy-cmds:v0.1 .` (TODO: hardcoded image name should be customized)
Sonobuoy will create a pod with a container from this image. Image translates the arguments from the plugin config into commands, the outputs 
of the latter will be collected by the sonobuoy collector sidecar .

2. Update `CRD`s with `k kudo init` (or `kubectl-kudo init --version 0.11.1` if no version var provided for build)

3. By default, sonobuoy collects information about resources and logs. If some additional data is needed, e.g. configuration files from the pods,
output of commands executed on pods or a response from a certain service, then a `diagnostics.bundle` key should be added to operator.yaml as in
the example [operator.yaml](./example/operator.yaml)

4. `kubectl-kudo diagnostics sonobuoy-gen` will generate a configuration file and a plugin (for additional info mentioned in the previous item)
for sonobuoy diagnostics collection on the instance and a configuration file for kudo controller diagnostics collection (no additional info for 
controller).

5. `sonobuoy run --config config.json --plugin plugin.yaml` will collect instance diagnostics and `sonobuoy run --config config-kudo.yaml` will
collect kudo controller diagnostics

## Pros and Cons of Sonobuoy usage for KUDO diagnostics bundle (as compared to KUDO-only approach)

### Pros

1. Ready-to-use tool which already takes care of default and custom data collection

2. Ready-to-use architecture: master pod + sidecars alongside collector pods, shared volumes, proper RBAC, services to allow monitoring

3. A specialized and maintained product that has already been well tested for different clusters and cloud services (while KUDO-based approach will
be just one of the features)

### Cons

1. Sonobuoy is designed to test clusters and thus is wonderful to collect _all_ the cluster data with uniformly applied filters, but attempt to collect
resources based on different filters or rules will require different configs and several runs. E.g., if we filter resources by kudo.dev/operator=zookeeper
the operator itself and the operator version will drop out as not being labeled so.

2. It is unclear how to deliver a feature of additional data post-processing, e.g. redaction, as long as KUDO avoids invoking sonobuoy from its 
code (which may presumably be illegal for commercial purposes)

3. If invoking sonobuoy from the code is avoided, the diagnostics collection requires two steps: config generation and run, the latter step
being also a complex one as due to con.#1 we cannot collect only the resources we needed in one run

4. Implementation of `--continue` feature or stopping and resuming diagnostics collection will become complicated

5. Almost all necessary configuration except minor details like number of log tail lines is known in advances and can be simply stored in
the original sonobuoy config files from the very beginning. KUDO basically would just encode these configs into its code and operator version,
which looks dubious.

6. If a cluster is in a severely unstable state starting new pods may present and issue. Even though, with KUDO-only approach it's still hard
to avoid creating pods for collecting "additional data", plain resources and logs collection may well be done just via K8 API calls.

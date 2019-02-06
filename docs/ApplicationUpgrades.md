# Maestro Managing Deployment Upgrades



## Register Rollout plan for Parameter

The operations required to safely update a running application can vary depending on which parameter is
being updated.  For instance updating the `BROKER_COUNT`, may require a simple update of the deployment, whereas
updating the `APPLICATION_MEMORY` may require rolling out a new version via a canary or blue/green deployment.

Each parameter can be configured with a default update plan via the `deploy` plan that is required by all `FrameworkVersions`.

```yaml
...
spec:
  parameters:
  - name: REPLICAS
    update: deploy
  - name: APPLICATION_MEMORY
    update: canary
  plans:
    canary:
    ...
    deploy:
    ...
```


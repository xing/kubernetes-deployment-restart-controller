# Kubernetes Deployment Restart Controller

This Kubernetes controller watches ConfigMaps and Secrets referenced by Deployments and
StatefulSets and triggers restarts as soon as configuration or secret values change.

## Installation

The [k8s-manifests] folder contains the necessary configuration files. Adjust them to your
taste and apply in the given order:

    kubectl apply -f k8s-manifests/rbac.yaml -f k8s-manifests/deployment.yaml

Optionally, create the [metrics Service]:

    kubectl apply -f k8s-manifests/metrics-service.yaml

## Configuration

Automatic restart functionality is enabled on per-Deployment (StatefulSet) basis.

The only thing you need to do is set the `com.xing.deployment-restart` annotation on the
desired Deployment (or StatefulSet) to `enabled`:

```yml
apiVersion: apps/v1beta2
kind: Deployment
metadata:
  annotations:
    com.xing.deployment-restart: enabled
  # the rest of the deployment manifest
```

Controller monitors deployment manifests and automatically watches or stops watching
relevant ConfigMaps and Secrets. It also stops restarting a deployment as soon as
annotation is removed or changed to anything else than `enabled`.

## Implementation Details

Kubernetes exhibits several constraints that shaped the implementation of the controller a
great deal.

Even though Kubernetes assigns a version to every resource deployed to the cluster, it is
impossible to figure out which version of a particular ConfigMap or Secret was used when a
Pod was started. Because of that, the controller maintains its own dataset of
configuration object versions applied to Deployments and StatefulSets. Kubernetes resource
versions get incremented when any part of the resource definition is changed, not just the
configuration data. To avoid unnecessary restarts, the controller calculates a
configuration data checksum for every ConfigMap and Secret and uses it instead of resource
version to detect changes.

Another constraint is related to resource discovery. Kubernetes Informers are asynchronous
and do not guarantee that the client gets the most recent cluster changes immediately. The
the order in which the client gets the resource definitions is also not fixed. This makes
it necessary for the controller to always operate on a potentially incomplete view of the
cluster state. For example, the controller can become aware of a new Deployment resource
recently created in the cluster, but yet have no information on a ConfigMap referenced in
the Deployment manifest.

### How It Works

Every deployment [configured for automatic restarts][installation and configuration]
eventually gets updated with a **checksums annotation**. The annotation contains names and
checksums of all the configs referenced by the deployment. The controller maintains this
information and uses it to detect config changes.

The annotation itself is a JSON object:

```yaml
metadata:
  annotations:
    com.xing.deployment-restart.applied-config-checksums: '{"configmap/namespace-one/config-one":"189832cc316e7594","secret/namespace-one/secret-two":"6e79832c18c31594"}'
```

The controller watches all ConfigMap, Secret, Deployment and StatefulSet resources in the
cluster and builds a **catalog** of deployments and related configs in memory as resources
become known to it.

Catalog entries for config resources can represent either actual resources in the cluster
and have **checksums** associated with them, or they can be dummies merely stating the
fact that some of the known deployment resources reference a particular config name.

Deployment entries in the catalog always represent real deployment objects in the cluster.

When receiving the information about another config or deployment, controller can
instantiate a **change** object. There are several situations when it happens:

 * New config is added to the catalog.
 * New deployment is added to the catalog.
 * Known deployment references a config that it was not referencing before.
 * Known deployment does not reference a config that it was referencing before.
 * Known deployment has its checksums annotation changed.
 * Known config has its checksum changed.

Every change is identified by the name of the resource it was initiated by and has a
timestamp and a **counter** associated with it.

Instantiated changes are stored in the queue without any actions to them for the amount of
time determined by [RESTART_GRACE_PERIOD setting][command line arguments]. If the resource
is changed again during the grace period, the change counter gets incremented.

After the grace period is exhausted, the change gets processed:

1. Using the catalog, controller discovers deployments that are potentially affected by
the change. A deployment change can only affect the deployment itself, while a config
change affects all the deployments that reference the config.

2. Every potentially affected deployment has its checksums annotation compared to the
current state of the catalog. Based on the comparison, controller decides if the
annotation should be updated and if the deployment needs to be restarted.

3. If an annotation update or a restart is necessary, controller patches the deployment.
It always issues a single patch request to update the annotation and restart the
deployment at the same time if necessary. A restart is triggered by setting the
`com.xing.deployment-restart.timestamp` annotation in `spec.template.metadata.annotations`
of the deployment.

There are two situations when a deployment restart is triggered by a config update:

1. The deployment **already has a checksum of the updated config** in the checksums
annotation, and the new checksum of the config is different. This is the "normal"
situation when a config that deployment has already been referencing for some time gets
updated.

2. The deployment does not have a checksum of the updated config in the checksums
annotation, but **the corresponding change counter is greater than one**. This is the
situation when a config got added to the deployment, but before the checksum got saved in
the deployment annotation, the config got updated again.

### Caveats

1. Combined with other automation tools, controller can cause deployments to be restarted
more often they need to be. For example, if some deployment pipeline takes longer than 5
seconds ([by default][command line arguments]) to update two ConfigMaps that are
referenced by the same deployment, the deployment will be restarted twice. This can be
mitigated by either changing the pipeline or increasing the restart check period.

2. When forcefully terminated, the controller might miss some restarts. Consider the
situation: a new deployment is added to the cluster. Soon after that, a config referenced
by that deployment is updated, and while that change is being on hold for the grace
period, controller gets forcefully terminated. The fact that there was a config change
observed would not be stored anywhere, and another instance of the controller will just
mark the config as already applied to the deployment. The chances of this happening are
rather low since controller needs to be killed exactly during the grace period of a
deployment change followed by a config change.

## Runtime Metrics

The controller exposes several metrics at `0.0.0.0:10254/metrics` endpoint in Prometheus
format. These metrics can be used to monitor the controller status and observe actions
that it takes.

Metric | Type | Description
------ | ---- | -----------
deployment_restart_controller_resource_versions_total | counter | The number of distinct resource versions observed.
deployment_restart_controller_configs_total | gauge | The number of tracked configs.
deployment_restart_controller_deployments_total | gauge | The number of tracked deployments.
deployment_restart_controller_deployment_annotation_updates_total | counter | The number of deployment annotation updates.
deployment_restart_controller_deployment_restarts_total | counter | The number of deployment restarts triggered.
deployment_restart_controller_changes_processed_total | counter | The number of resource changes processed.
deployment_restart_controller_changes_waiting_total | gauge |  The number of changes waiting in the queue.

## Command Line Arguments

```
Usage:
  kubernetes-deployment-restart-controller [OPTIONS]

Application Options:
  -c, --restart-check-period= Time interval to check for pending restarts in milliseconds (default: 500) [$RESTART_CHECK_PERIOD]
  -r, --restart-grace-period= Time interval to compact restarts in seconds (default: 5) [$RESTART_GRACE_PERIOD]
  -v, --verbose=              Be verbose [$VERBOSE]
      --version               Print version information and exit

Help Options:
  -h, --help                  Show this help message
```

## Development

This project uses go modules introduced by [go 1.11][go-modules]. Please put the project
somewhere outside of your GOPATH to make go automatically recogninze this.

All build and install steps are managed in the [Makefile](Makefile). `make test` will
fetch external dependencies, compile the code and run the tests. If all goes well, hack
along and submit a pull request. You might need to run the `go mod tidy` after updating
dependencies.

### Releases

Releases are a two-step process, beginning with a manual step:

* Create a release commit
  * Increase the version number in [kubernetes-deployment-restart-controller.go/VERSION](kubernetes-deployment-restart-controller.go#25)
  * Adjust the [CHANGELOG](CHANGELOG.md)
* Run `make release`, which will create an image, retrieve the version from the
  binary, create a git tag and push both your commit and the tag

The Travis CI run will then realize that the current tag refers to the current master commit and
will tag the built docker image accordingly.


[22368]: https://github.com/kubernetes/kubernetes/issues/22368
[implementation details]: #implementation-details
[metrics service]: #runtime-metrics
[k8s-manifests]: tree/master/k8s-manifests
[installation and configuration]: #installation-and-configuration
[command line arguments]: #command-line-arguments
[go-modules]: https://github.com/golang/go/wiki/Modules

# CVMS Helm Chart

The Cosmos Validator Monitoring Service (CVMS) is an integrated monitoring system for validators within the Cosmos app chain ecosystem. This helm chart is fot installing cvms on kubernetes.

## Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

## Installation

To install the chart with the release name `my-release`:

```bash
helm install my-release cvms/helm
```

## Uninstallation

To uninstall/delete the `my-release` deployment:

```bash
helm uninstall my-release
```

## Configuration

The following table lists the configurable parameters of the CVMS chart and their default values.

**Note**: You must provide cvmsConfig.monikers and cvmsConfig.chains for monitoring Cosmos validators with cvms, checkout the example in https://github.com/cosmostation/cvms/blob/release/docs/setup.md

| Parameter                | Description                           | Default                        |
|--------------------------|---------------------------------------|--------------------------------|
| `imagePullSecrets`       | Image pull secrets                    | `[]`                           |
| `nameOverride`           | Override the name of the chart        | `""`                           |
| `fullnameOverride`       | Override the full name of the chart   | `""`                           |
| `namespaceOverride`      | Override the namespace of the chart   | `""`                           |
| `customChainsConfig.enabled` | Enable custom chains configuration | `false`                        |
| `customChainsConfig.name`    | Name of the custom chains configmap | `custom-chains-cm`             |
| `cvmsConfig.name`        | Name of the CVMS configmap            | `cvms-cm`                      |
| `cvmsConfig.monikers`    | CVMS config for validator or network mode | `[]`                       |
| `cvmsConfig.chains`      | CVMS config for monitoring chains     | `[]`                           |
| `indexer.replicaCount`   | Number of replicas for the indexer    | `1`                            |
| `indexer.image.repository` | Image repository for the indexer     | `cosmostation/cvms`            |
| `indexer.image.pullPolicy` | Image pull policy for the indexer    | `Always`                       |
| `indexer.image.tag`      | Image tag for the indexer             | `latest`                       |
| `indexer.command`        | Command to run in the indexer container | `["/bin/cvms"]`              |
| `indexer.args`           | Arguments for the indexer container   | `["start", "indexer", "--config=/var/lib/cvms/config.yaml", "--log-color-disable", "false", "--log-level", "4", "--port=9300"]` |
| `indexer.env`            | Environment variables for the indexer | `{ DB_RETENTION_PERIOD: 1h }`  |
| `indexer.revisionHistoryLimit` | Number of old ReplicaSets to retain | `2`                        |
| `indexer.podAnnotations` | Annotations for the indexer pods      | `{}`                           |
| `indexer.podLabels`      | Labels for the indexer pods           | `{}`                           |
| `indexer.podSecurityContext.enabled` | Enable pod security context for the indexer | `false`          |
| `indexer.securityContext.enabled` | Enable security context for the indexer | `true`                |
| `indexer.service.type`   | Service type for the indexer          | `ClusterIP`                    |
| `indexer.service.port`   | Service port for the indexer          | `9300`                         |
| `indexer.resources.enabled` | Enable resource requests and limits for the indexer | `true`          |
| `indexer.livenessProbe.enabled` | Enable liveness probe for the indexer | `true`                  |
| `indexer.readinessProbe.enabled` | Enable readiness probe for the indexer | `true`                |
| `indexer.volumes`        | Additional volumes for the indexer    | See `values.yaml`              |
| `indexer.volumeMounts`   | Additional volume mounts for the indexer | See `values.yaml`           |
| `indexer.nodeSelector`   | Node selector for the indexer pods    | `{}`                           |
| `indexer.tolerations`    | Tolerations for the indexer pods      | `[]`                           |
| `indexer.affinity`       | Affinity rules for the indexer pods   | `{}`                           |
| `indexer.metrics.type`   | Metrics service type for the indexer  | `ClusterIP`                    |
| `indexer.metrics.servicePort` | Metrics service port for the indexer | `9300`                     |
| `indexer.serviceMonitor.enabled` | Enable Prometheus ServiceMonitor for the indexer | `false`         |
| `exporter.replicaCount`  | Number of replicas for the exporter   | `1`                            |
| `exporter.image.repository` | Image repository for the exporter   | `cosmostation/cvms`            |
| `exporter.image.pullPolicy` | Image pull policy for the exporter  | `Always`                       |
| `exporter.image.tag`     | Image tag for the exporter            | `latest`                       |
| `exporter.command`       | Command to run in the exporter container | `["/bin/cvms"]`            |
| `exporter.args`          | Arguments for the exporter container  | `["start", "exporter", "--config=/var/lib/cvms/config.yaml", "--log-color-disable", "false", "--log-level", "4", "--port=9200"]` |
| `exporter.env`           | Environment variables for the exporter | `{}`                          |
| `exporter.revisionHistoryLimit` | Number of old ReplicaSets to retain | `2`                        |
| `exporter.podAnnotations` | Annotations for the exporter pods    | `{}`                           |
| `exporter.podLabels`     | Labels for the exporter pods          | `{}`                           |
| `exporter.podSecurityContext.enabled` | Enable pod security context for the exporter | `false`        |
| `exporter.securityContext.enabled` | Enable security context for the exporter | `true`              |
| `exporter.service.type`  | Service type for the exporter         | `ClusterIP`                    |
| `exporter.service.port`  | Service port for the exporter         | `9200`                         |
| `exporter.resources.enabled` | Enable resource requests and limits for the exporter | `true`        |
| `exporter.livenessProbe.enabled` | Enable liveness probe for the exporter | `true`                |
| `exporter.readinessProbe.enabled` | Enable readiness probe for the exporter | `true`              |
| `exporter.volumes`       | Additional volumes for the exporter   | See `values.yaml`              |
| `exporter.volumeMounts`  | Additional volume mounts for the exporter | See `values.yaml`           |
| `exporter.nodeSelector`  | Node selector for the exporter pods   | `{}`                           |
| `exporter.tolerations`   | Tolerations for the exporter pods     | `[]`                           |
| `exporter.affinity`      | Affinity rules for the exporter pods  | `{}`                           |
| `exporter.metrics.type`  | Metrics service type for the exporter | `ClusterIP`                    |
| `exporter.metrics.servicePort` | Metrics service port for the exporter | `9200`                   |
| `exporter.serviceMonitor.enabled` | Enable Prometheus ServiceMonitor for the exporter | `false`       |
| `postgresql.enabled`     | Enable PostgreSQL                     | `true`                         |
| `postgresql.auth.username` | PostgreSQL username                 | `cvms`                         |
| `postgresql.auth.password` | PostgreSQL password                 | `mysecretpassword`             |
| `postgresql.auth.database` | PostgreSQL database                 | `cvms`                         |
| `postgresql.architecture` | PostgreSQL architecture              | `standalone`                   |
| `postgresql.primary.resourcesPreset` | PostgreSQL primary resources preset | `nano`                 |
| `postgresql.primary.resources` | PostgreSQL primary resources    | `{}`                           |
| `postgresql.primary.persistence.storageClass` | PostgreSQL primary storage class | `""`              |
| `postgresql.primary.persistence.size` | PostgreSQL primary storage size | `500Mi`                   |
| `postgresql.external.host` | External PostgreSQL host            | `""`                           |
| `postgresql.external.port` | External PostgreSQL port            | `5432`                         |
| `postgresql.external.user` | External PostgreSQL user            | `cvms`                         |
| `postgresql.external.password` | External PostgreSQL password    | `""`                           |
| `postgresql.external.database` | External PostgreSQL database    | `cvms`                         |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```bash
helm install my-release /path/to/cvms --set indexer.image.tag=1.0.0 --set exporter.image.tag=1.0.0
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while installing the chart. For example:

```bash
helm install my-release /path/to/cvms -f values.yaml
```

### Detailed Configuration

Below is a more detailed explanation of the values that can be set in the `values.yaml` file:

- `image.repository`: The Docker image repository for the CVMS application.
- `image.tag`: The tag of the Docker image to use.
- `image.pullPolicy`: The Kubernetes image pull policy.
- `service.type`: The type of Kubernetes service to create (e.g., `ClusterIP`, `NodePort`, `LoadBalancer`).
- `service.port`: The port on which the service will be exposed.
- `resources`: Resource requests and limits for the CVMS pods.
- `nodeSelector`: Node labels for pod assignment.
- `tolerations`: Tolerations for pod assignment.
- `affinity`: Affinity rules for pod assignment.

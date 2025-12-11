[![CircleCI](https://circleci.com/gh/giantswarm/teleport-exporter.svg?style=shield)](https://circleci.com/gh/giantswarm/teleport-exporter)

# teleport-exporter

A Prometheus exporter that exposes metrics about resources registered in [Teleport](https://goteleport.com/), including nodes, Kubernetes clusters, databases, and applications.

## Overview

The teleport-exporter connects to a Teleport cluster and periodically collects information about all registered resources, exposing them as Prometheus metrics. This allows you to:

- Monitor the number of nodes, Kubernetes clusters, databases, and applications in your Teleport cluster
- Get visibility into your Teleport infrastructure
- Set up alerts based on changes in your Teleport resources
- Create dashboards showing your Teleport cluster state

## Metrics

The exporter exposes the following metrics:

### General Metrics

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_up` | Whether the exporter can successfully connect to Teleport (1 = connected, 0 = disconnected) | - |
| `teleport_exporter_cluster_info` | Information about the Teleport cluster | `cluster_name` |
| `teleport_exporter_collect_duration_seconds` | Duration of the last metrics collection in seconds | - |

### Node Metrics

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_nodes_total` | Total number of nodes registered in the Teleport cluster | `cluster_name` |
| `teleport_exporter_node_info` | Information about each node registered in Teleport | `cluster_name`, `node_name`, `hostname`, `address`, `namespace`, `subkind` |

### Kubernetes Cluster Metrics

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_kubernetes_clusters_total` | Total number of Kubernetes clusters registered in the Teleport cluster | `cluster_name` |
| `teleport_exporter_kubernetes_cluster_info` | Information about each Kubernetes cluster registered in Teleport | `cluster_name`, `kube_cluster_name` |

### Database Metrics

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_databases_total` | Total number of databases registered in the Teleport cluster | `cluster_name` |
| `teleport_exporter_database_info` | Information about each database registered in Teleport | `cluster_name`, `database_name`, `protocol`, `type` |

### Application Metrics

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_apps_total` | Total number of applications registered in the Teleport cluster | `cluster_name` |
| `teleport_exporter_app_info` | Information about each application registered in Teleport | `cluster_name`, `app_name`, `public_addr` |

## Prerequisites

### Teleport Identity File

The exporter requires a Teleport identity file for authentication. This can be generated using:

1. **Machine ID (recommended for production)**:
   ```bash
   # Create a bot for the exporter
   tctl bots add teleport-exporter --roles=auditor

   # The bot will output instructions for running tbot to generate the identity file
   ```

2. **Manual generation (for testing)**:
   ```bash
   tctl auth sign --user=teleport-exporter --out=identity --ttl=8760h
   ```

The identity file should be stored in a Kubernetes secret:

```bash
kubectl create secret generic teleport-exporter-identity \
  --from-file=identity=/path/to/identity \
  -n <namespace>
```

## Installation

### Using Helm

```bash
helm repo add giantswarm https://giantswarm.github.io/giantswarm-catalog
helm repo update

helm upgrade -i teleport-exporter \
  --namespace <namespace> \
  --set teleport.address=teleport.example.com:443 \
  --set identity.existingSecret=teleport-exporter-identity \
  giantswarm/teleport-exporter
```

### Configuration

The following table lists the configurable parameters:

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicas` | Number of replicas | `1` |
| `teleport.address` | Address of the Teleport proxy/auth server | `""` (required) |
| `teleport.identityFilePath` | Path to the identity file inside the container | `/var/run/teleport/identity` |
| `teleport.insecure` | Skip TLS certificate verification | `false` |
| `exporter.refreshInterval` | How often to refresh metrics from Teleport API | `30s` |
| `identity.existingSecret` | Name of existing secret containing the identity file | `""` (required) |
| `monitoring.serviceMonitor.enabled` | Create a ServiceMonitor for Prometheus Operator | `true` |
| `networkpolicy.enabled` | Create a NetworkPolicy | `true` |
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `resources.limits.cpu` | CPU limit | `100m` |
| `resources.limits.memory` | Memory limit | `128Mi` |

## Command Line Arguments

| Argument | Description | Default |
|----------|-------------|---------|
| `--metrics-bind-address` | The address the metric endpoint binds to | `:8080` |
| `--health-probe-bind-address` | The address the probe endpoint binds to | `:8081` |
| `--teleport-addr` | The address of the Teleport proxy/auth server | `""` |
| `--identity-file` | Path to the identity file for authentication | `""` |
| `--refresh-interval` | How often to refresh metrics from Teleport API | `30s` |
| `--insecure` | Skip TLS certificate verification | `false` |

## Example Prometheus Queries

```promql
# Total number of nodes in the Teleport cluster
teleport_exporter_nodes_total

# List all Kubernetes clusters
teleport_exporter_kubernetes_cluster_info

# Alert when a node disappears
absent(teleport_exporter_node_info{node_name="my-important-node"})

# Track database count over time
rate(teleport_exporter_databases_total[1h])
```

## Example Grafana Dashboard

You can create a dashboard with panels for:

1. **Overview Panel**: Show `teleport_exporter_up` status
2. **Node Count**: Display `teleport_exporter_nodes_total`
3. **Kubernetes Clusters**: List from `teleport_exporter_kubernetes_cluster_info`
4. **Databases**: Show count and details from database metrics
5. **Applications**: Display application information

## Development

### Building

```bash
go build -o teleport-exporter .
```

### Running locally

```bash
./teleport-exporter \
  --teleport-addr=teleport.example.com:443 \
  --identity-file=/path/to/identity
```

### Docker

```bash
docker build -t teleport-exporter .
docker run -v /path/to/identity:/identity:ro \
  teleport-exporter \
  --teleport-addr=teleport.example.com:443 \
  --identity-file=/identity
```

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

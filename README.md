[![CircleCI](https://circleci.com/gh/giantswarm/teleport-exporter.svg?style=shield)](https://circleci.com/gh/giantswarm/teleport-exporter)
[![OpenSSF Scorecard](https://api.securityscorecards.dev/projects/github.com/giantswarm/teleport-exporter/badge)](https://securityscorecards.dev/viewer/?uri=github.com/giantswarm/teleport-exporter)

# teleport-exporter

A Prometheus exporter that exposes metrics about resources registered in [Teleport](https://goteleport.com/), including nodes, Kubernetes clusters, databases, and applications.

## Overview

The teleport-exporter connects to a Teleport cluster and periodically collects information about all registered resources, exposing them as Prometheus metrics. This allows you to:

- Monitor the number of nodes, Kubernetes clusters, databases, and applications in your Teleport cluster
- Get visibility into your Teleport infrastructure
- Set up alerts based on changes in your Teleport resources
- Create dashboards showing your Teleport cluster state

## Metrics

### Connection Status

| Metric | Description |
|--------|-------------|
| `teleport_exporter_up` | Connection status (1 = connected, 0 = disconnected) |

### SSH Nodes

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_nodes_total` | Total SSH nodes | `cluster_name` |
| `teleport_exporter_nodes_identified_total` | Nodes with identified K8s cluster | `cluster_name` |
| `teleport_exporter_nodes_unidentified_total` | Nodes with unknown K8s cluster | `cluster_name` |

### Kubernetes Clusters

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_kubernetes_clusters_total` | Total Kubernetes clusters | `cluster_name` |
| `teleport_exporter_kubernetes_management_clusters_total` | Management clusters (no hyphen in name) | `cluster_name` |
| `teleport_exporter_kubernetes_workload_clusters_total` | Workload clusters (has hyphen in name) | `cluster_name` |

### Databases

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_databases_total` | Total databases | `cluster_name` |
| `teleport_exporter_databases_by_protocol_total` | Databases by protocol | `cluster_name`, `protocol` |
| `teleport_exporter_databases_by_type_total` | Databases by type | `cluster_name`, `type` |

### Applications

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_apps_total` | Total applications | `cluster_name` |

### Exporter Health

| Metric | Description | Labels |
|--------|-------------|--------|
| `teleport_exporter_collect_duration_seconds` | Collection duration | `cluster_name` |
| `teleport_exporter_collect_errors_total` | Total collection errors | `cluster_name` |
| `teleport_exporter_last_successful_collect_timestamp_seconds` | Last successful collection timestamp | `cluster_name` |

## Installation

### Option 1: Deploy in Teleport Cluster (Recommended)

When deploying in the same cluster where Teleport is running, the chart can automatically create all required Teleport resources and use tbot for automatic identity management.

```bash
helm upgrade -i teleport-exporter \
  --namespace teleport \
  --set teleport.address=teleport.example.com:443 \
  --set teleport.createResources=true \
  --set tbot.enabled=true \
  giantswarm/teleport-exporter
```

This will automatically:
1. Create a TeleportRoleV7 with permissions to list nodes, clusters, databases, and apps
2. Create a TeleportBotV1 for the exporter
3. Create a TeleportProvisionToken for Kubernetes join method
4. Deploy tbot to manage identity renewal automatically
5. Store the identity in a Kubernetes secret

### Option 2: Deploy with Pre-created Identity

If you prefer to manage the identity manually or deploy outside the Teleport cluster:

#### Step 1: Create the Teleport Role

Apply the following role in your Teleport cluster:

```yaml
kind: role
version: v7
metadata:
  name: teleport-exporter
spec:
  allow:
    node_labels:
      '*': '*'
    kubernetes_labels:
      '*': '*'
    db_labels:
      '*': '*'
    app_labels:
      '*': '*'
    rules:
      - resources: [node]
        verbs: [list, read]
      - resources: [kube_server, kube_cluster]
        verbs: [list, read]
      - resources: [db_server, db]
        verbs: [list, read]
      - resources: [app_server, app]
        verbs: [list, read]
      - resources: [cluster_name]
        verbs: [list, read]
  options:
    max_session_ttl: 12h
```

```bash
tctl create -f teleport-exporter-role.yaml
```

#### Step 2: Create a Bot and Generate Identity

```bash
# Create the bot
tctl bots add teleport-exporter --roles=teleport-exporter

# Use tbot to generate the identity (replace TOKEN with the token from the previous command)
tbot start --oneshot \
  --token=<TOKEN> \
  --proxy-server=teleport.example.com:443 \
  --join-method=token \
  --destination-dir=/tmp/tbot-identity \
  --data-dir=/tmp/tbot-data
```

#### Step 3: Create the Kubernetes Secret

```bash
kubectl create secret generic teleport-exporter-identity \
  --from-file=identity=/tmp/tbot-identity/identity \
  -n <namespace>
```

#### Step 4: Deploy the Chart

```bash
helm upgrade -i teleport-exporter \
  --namespace <namespace> \
  --set teleport.address=teleport.example.com:443 \
  --set identity.existingSecret=teleport-exporter-identity \
  giantswarm/teleport-exporter
```

## Configuration

### Core Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicas` | Number of replicas | `1` |
| `teleport.address` | Address of the Teleport proxy/auth server | `""` (required) |
| `teleport.identityFilePath` | Path to the identity file inside the container | `/var/run/teleport/identity` |
| `teleport.insecure` | Skip TLS certificate verification | `false` |
| `teleport.createResources` | Create Teleport CRD resources (Role, Bot, Token) | `false` |
| `exporter.refreshInterval` | How often to refresh metrics from Teleport API | `30s` |

### Identity Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `identity.existingSecret` | Name of existing secret containing the identity file | `""` |

### tbot Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `tbot.enabled` | Deploy tbot for automatic identity renewal | `false` |
| `tbot.identitySecretName` | Name of the secret where tbot stores the identity | `<release-name>-identity` |
| `tbot.renewalInterval` | How often tbot renews the certificate | `20m` |
| `tbot.certificateTTL` | Certificate TTL | `24h` |
| `tbot.image.registry` | tbot image registry | `gsoci.azurecr.io` |
| `tbot.image.name` | tbot image name | `giantswarm/tbot-distroless` |
| `tbot.image.tag` | tbot image tag | `16.1.4` |

### Monitoring Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `monitoring.serviceMonitor.enabled` | Create a ServiceMonitor for Prometheus Operator | `true` |
| `monitoring.serviceMonitor.labels` | Additional labels for the ServiceMonitor | `{}` |
| `monitoring.serviceMonitor.interval` | Scrape interval | `""` |
| `networkpolicy.enabled` | Create a NetworkPolicy | `true` |

### Resource Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `resources.requests.cpu` | CPU request | `50m` |
| `resources.requests.memory` | Memory request | `64Mi` |
| `resources.limits.cpu` | CPU limit | `100m` |
| `resources.limits.memory` | Memory limit | `128Mi` |

## Example Values Files

### Full Teleport Cluster Deployment

```yaml
# values-teleport-cluster.yaml
teleport:
  address: "teleport.giantswarm.io:443"
  createResources: true

tbot:
  enabled: true
  renewalInterval: "20m"
  certificateTTL: "24h"

monitoring:
  serviceMonitor:
    enabled: true
```

### External Deployment with Existing Identity

```yaml
# values-external.yaml
teleport:
  address: "teleport.giantswarm.io:443"

identity:
  existingSecret: "my-teleport-identity"

monitoring:
  serviceMonitor:
    enabled: true
```

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
# Check if exporter is healthy
teleport_exporter_up == 1

# Total number of nodes in the Teleport cluster
teleport_exporter_nodes_total

# Total number of Kubernetes clusters
teleport_exporter_kubernetes_clusters_total

# Management vs workload cluster counts
teleport_exporter_kubernetes_management_clusters_total
teleport_exporter_kubernetes_workload_clusters_total

# Databases by protocol
sum by (protocol) (teleport_exporter_databases_by_protocol_total)

# Nodes with identified vs unknown cluster
teleport_exporter_nodes_identified_total
teleport_exporter_nodes_unidentified_total

# Track changes in resource counts over time
changes(teleport_exporter_kubernetes_clusters_total[1h])
```

## Grafana Dashboard

A pre-built dashboard is included at `grafana/teleport-exporter-dashboard.json`.

Features:
- Connection status and exporter health
- Resource totals with trend sparklines
- Node identification breakdown (identified vs unknown K8s cluster)
- Kubernetes cluster breakdown (MC vs WC)
- Database breakdown by protocol and type
- Resource trends over time

## Troubleshooting

### Identity Issues

If the exporter shows `teleport_exporter_up = 0`:

1. Check the pod logs: `kubectl logs -l app.kubernetes.io/name=teleport-exporter`
2. Verify the identity secret exists and has the correct key (`identity`)
3. Ensure the Teleport role has the required permissions (see role template above)
4. Check network connectivity to the Teleport proxy

### tbot Issues

If tbot fails to start:

1. Check tbot logs: `kubectl logs -l app.kubernetes.io/component=tbot`
2. Verify the TeleportProvisionToken was created: `tctl get tokens`
3. Ensure the ServiceAccount token audience matches the Teleport proxy address

### Permission Issues

If metrics show 0 for all resources but `teleport_exporter_up = 1`:

1. The Teleport role may be missing `*_labels` fields (node_labels, kubernetes_labels, etc.)
2. Regenerate the identity after updating the role

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

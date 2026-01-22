# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- New metrics for detailed resource breakdowns:
  - `teleport_exporter_nodes_identified_total` - nodes with identified K8s cluster (via labels or hostname)
  - `teleport_exporter_nodes_unidentified_total` - nodes with unknown K8s cluster
  - `teleport_exporter_nodes_by_kubernetes_cluster` - count of nodes per Kubernetes cluster
  - `teleport_exporter_kubernetes_management_clusters_total` - management clusters (no hyphen in name)
  - `teleport_exporter_kubernetes_workload_clusters_total` - workload clusters (has hyphen in name)
  - `teleport_exporter_kubernetes_cluster_info` - info metric for each Kubernetes cluster name
  - `teleport_exporter_databases_by_protocol_total` - databases grouped by protocol (postgres, mysql, etc.)
  - `teleport_exporter_databases_by_type_total` - databases grouped by type (rds, self-hosted, etc.)
- Node K8s cluster identification from labels (`giantswarm.io/cluster`, `cluster`, `kubernetes-cluster`) or hostname patterns
- All health metrics now include `cluster_name` label for multi-cluster monitoring

### Changed

- Grafana dashboard redesigned with modern visualizations:
  - Resource totals with trend sparklines
  - Node identification breakdown (identified vs unknown)
  - Kubernetes cluster breakdown (MC vs WC) with pie chart
  - Database breakdown by protocol and type with pie charts
  - Resource trends over time

### Removed

- **BREAKING**: Removed per-resource info metrics (replaced with aggregate breakdowns):
  - `teleport_exporter_node_info`
  - `teleport_exporter_kubernetes_cluster_info`
  - `teleport_exporter_database_info`
  - `teleport_exporter_app_info`

## [0.1.2] - 2026-01-20

### Changed

  - Reduced metrics cardinality

## [0.1.1] - 2025-12-23

### Changed

  - Reduced metrics cardinality

## [0.1.0] - 2025-12-17

## [0.0.2] - 2025-12-11

### Added

- Grafana dashboard for visualizing Teleport metrics

### Fixed

- ServiceMonitor now includes `application.giantswarm.io/team` label required by Prometheus Operator

## [0.0.1] - 2025-12-11

### Added

- Initial release of teleport-exporter
- Prometheus metrics for Teleport resources:
  - `teleport_exporter_up` - Connection status
  - `teleport_exporter_cluster_info` - Cluster information
  - `teleport_exporter_nodes_total` - Total SSH nodes count
  - `teleport_exporter_node_info` - Detailed node information
  - `teleport_exporter_kubernetes_clusters_total` - Total Kubernetes clusters count
  - `teleport_exporter_kubernetes_cluster_info` - Detailed Kubernetes cluster information
  - `teleport_exporter_databases_total` - Total databases count
  - `teleport_exporter_database_info` - Detailed database information
  - `teleport_exporter_apps_total` - Total applications count
  - `teleport_exporter_app_info` - Detailed application information
  - `teleport_exporter_collect_duration_seconds` - Collection duration
- Helm chart with support for:
  - Automatic Teleport resource creation via CRDs (TeleportRoleV7, TeleportBotV1, TeleportProvisionToken)
  - tbot deployment for automatic identity renewal
  - Manual identity configuration via existing secrets
  - ServiceMonitor for Prometheus Operator
  - NetworkPolicy for security
  - VerticalPodAutoscaler support
- Health and readiness probes
- Configurable refresh interval for metrics collection

[Unreleased]: https://github.com/giantswarm/teleport-exporter/compare/v0.1.2...HEAD
[0.1.2]: https://github.com/giantswarm/teleport-exporter/compare/v0.1.1...v0.1.2
[0.1.1]: https://github.com/giantswarm/teleport-exporter/compare/v0.1.0...v0.1.1
[0.1.0]: https://github.com/giantswarm/teleport-exporter/compare/v0.0.2...v0.1.0
[0.0.2]: https://github.com/giantswarm/teleport-exporter/compare/v0.0.1...v0.0.2
[0.0.1]: https://github.com/giantswarm/teleport-exporter/releases/tag/v0.0.1

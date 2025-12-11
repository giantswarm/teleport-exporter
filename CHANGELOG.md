# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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

[Unreleased]: https://github.com/giantswarm/teleport-exporter/compare/v0.0.1...HEAD
[0.0.1]: https://github.com/giantswarm/teleport-exporter/releases/tag/v0.0.1

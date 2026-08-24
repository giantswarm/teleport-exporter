# teleport-exporter

A Helm chart for teleport-exporter, which exposes Prometheus metrics about nodes, clusters, databases, and applications registered in Teleport.

**Homepage:** <https://github.com/giantswarm/teleport-exporter>

## Values

| Key | Type | Default | Description |
|-----|------|---------|-------------|
| replicas | int | `1` |  |
| registry.domain | string | `"gsoci.azurecr.io"` |  |
| image.name | string | `"giantswarm/teleport-exporter"` |  |
| imagePullSecrets | list | `[]` |  |
| global.podSecurityStandards.enforced | bool | `true` |  |
| pod.user.id | int | `65532` |  |
| pod.group.id | int | `65532` |  |
| nodeSelector | object | `{}` |  |
| tolerations | list | `[]` |  |
| podLabels | object | `{}` |  |
| podSecurityContext.runAsNonRoot | bool | `true` |  |
| podSecurityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| securityContext.allowPrivilegeEscalation | bool | `false` |  |
| securityContext.capabilities.drop[0] | string | `"ALL"` |  |
| securityContext.privileged | bool | `false` |  |
| securityContext.readOnlyRootFilesystem | bool | `true` |  |
| securityContext.runAsNonRoot | bool | `true` |  |
| securityContext.seccompProfile.type | string | `"RuntimeDefault"` |  |
| resources.requests.cpu | string | `"50m"` |  |
| resources.requests.memory | string | `"64Mi"` |  |
| resources.limits.cpu | string | `"100m"` |  |
| resources.limits.memory | string | `"128Mi"` |  |
| teleport.address | string | `""` |  |
| teleport.identityFilePath | string | `"/var/run/teleport/identity"` |  |
| teleport.insecure | bool | `false` |  |
| teleport.createResources | bool | `false` |  |
| exporter.refreshInterval | string | `"30s"` |  |
| identity.existingSecret | string | `""` |  |
| tbot.enabled | bool | `false` |  |
| tbot.identitySecretName | string | `""` |  |
| tbot.renewalInterval | string | `"20m"` |  |
| tbot.certificateTTL | string | `"24h"` |  |
| tbot.image.registry | string | `"gsoci.azurecr.io"` |  |
| tbot.image.name | string | `"giantswarm/tbot-distroless"` |  |
| tbot.image.tag | string | `"18.2.4"` |  |
| tbot.image.pullPolicy | string | `"IfNotPresent"` |  |
| tbot.resources.requests.cpu | string | `"50m"` |  |
| tbot.resources.requests.memory | string | `"64Mi"` |  |
| tbot.resources.limits.cpu | string | `"250m"` |  |
| tbot.resources.limits.memory | string | `"256Mi"` |  |
| monitoring.serviceMonitor.enabled | bool | `true` |  |
| monitoring.serviceMonitor.labels | object | `{}` |  |
| monitoring.serviceMonitor.interval | string | `""` |  |
| monitoring.serviceMonitor.scrapeTimeout | string | `""` |  |
| monitoring.serviceMonitor.relabelings[0].action | string | `"labeldrop"` |  |
| monitoring.serviceMonitor.relabelings[0].regex | string | `"pod|service|container"` |  |
| monitoring.serviceMonitor.metricRelabelings | list | `[]` |  |
| networkpolicy.enabled | bool | `true` |  |
| podAnnotations | object | `{}` |  |
| verticalPodAutoscaler.enabled | bool | `false` |  |
| verticalPodAutoscaler.containerPolicies.minAllowed.cpu | string | `"25m"` |  |
| verticalPodAutoscaler.containerPolicies.minAllowed.memory | string | `"32Mi"` |  |
| verticalPodAutoscaler.containerPolicies.maxAllowed.cpu | string | `"500m"` |  |
| verticalPodAutoscaler.containerPolicies.maxAllowed.memory | string | `"512Mi"` |  |

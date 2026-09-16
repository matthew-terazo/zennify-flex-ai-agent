# PostgreSQL Helm Chart

A Helm chart for deploying PostgreSQL on Kubernetes for the Zennify AI Assistant.

## Installation

```bash
helm install postgres ./helm/postgres
```

Or with custom values:

```bash
helm install postgres ./helm/postgres \
  --set postgres.password=mypassword \
  --set storage.size=20Gi
```

## Configuration

| Parameter | Description | Default |
|-----------|-------------|---------|
| `replicaCount` | Number of replicas | `1` |
| `image.repository` | Docker image repository | `postgres` |
| `image.tag` | Docker image tag | `15-alpine` |
| `postgres.username` | PostgreSQL username | `postgres` |
| `postgres.password` | PostgreSQL password | `changeme` |
| `postgres.database` | Initial database name | `zennify` |
| `postgres.port` | PostgreSQL port | `5432` |
| `storage.enabled` | Enable persistent storage | `true` |
| `storage.size` | PVC size | `10Gi` |
| `storage.storageClass` | Storage class name | `""` |
| `resources.limits.cpu` | CPU limit | `500m` |
| `resources.limits.memory` | Memory limit | `512Mi` |
| `resources.requests.cpu` | CPU request | `250m` |
| `resources.requests.memory` | Memory request | `256Mi` |

## Uninstallation

```bash
helm uninstall postgres
```

## Notes

- Uses a StatefulSet to ensure stable network identity and persistent storage
- Default PostgreSQL version is 15 (Alpine Linux)
- Database credentials are stored in a Kubernetes Secret
- Liveness and readiness probes use `pg_isready` command
- Security context runs PostgreSQL as non-root user (uid: 999)

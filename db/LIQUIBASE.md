# Liquibase Database Migration Guide

This directory contains Liquibase configuration for managing database migrations for the Zennify AI Assistant.

## Installation

### Local Development

1. Download Liquibase from https://www.liquibase.org/get-started/download or install via Homebrew:
```bash
brew install liquibase
```

2. Download the PostgreSQL JDBC driver:
```bash
cd db
liquibase extension --approve-license --yes
```

### Docker

Use the official Liquibase Docker image:
```bash
docker run --rm -v $(pwd)/db:/liquibase/changelog liquibase/liquibase:latest
```

## Configuration

- `liquibase.properties` — Local development configuration
- `liquibase-docker.properties` — Docker/Kubernetes configuration (uses environment variables)

Edit these files to match your database connection details.

## Commands

### Update Database

Apply all pending changesets:
```bash
liquibase update
```

With Docker:
```bash
docker run --rm \
  -v $(pwd)/db:/liquibase/changelog \
  -e DB_HOST=postgres \
  -e DB_PORT=5432 \
  -e DB_NAME=zennify \
  -e DB_USER=postgres \
  -e DB_PASSWORD=changeme \
  liquibase/liquibase:latest update
```

### Check Status

View migration status without applying changes:
```bash
liquibase status
```

### View History

Show applied changesets:
```bash
liquibase history
```

### Rollback

Rollback to a specific date:
```bash
liquibase rollback-by-date --date=2026-06-23
```

Rollback last N changesets:
```bash
liquibase rollback --count=1
```

### Generate SQL

Preview SQL that would be executed:
```bash
liquibase update-sql
```

### Validate

Validate changelog for errors:
```bash
liquibase validate
```

## Directory Structure

```
db/
  migrations/
    001_create_prompt_configuration_schema.sql
  liquibase.yaml              # Changelog definition
  liquibase.properties        # Local configuration
  liquibase-docker.properties # Docker configuration
  LIQUIBASE.md               # This file
```

## Adding New Migrations

1. Create a new SQL file in `migrations/` directory:
```
002_add_new_table.sql
```

2. Add changeset to `liquibase.yaml`:
```yaml
- changeSet:
    id: 002_add_new_table
    author: zennify
    description: "Add new table"
    sqlFile:
      path: migrations/002_add_new_table.sql
      relativeToChangelogFile: true
    rollback:
      - dropTable:
          tableName: new_table
```

3. Run `liquibase update` to apply

## Liquibase with Helm

To run Liquibase migrations as a Kubernetes Job in Helm:

```yaml
apiVersion: batch/v1
kind: Job
metadata:
  name: liquibase-migrate
spec:
  template:
    spec:
      containers:
      - name: liquibase
        image: liquibase/liquibase:latest
        env:
        - name: DB_HOST
          value: postgres
        - name: DB_PORT
          value: "5432"
        - name: DB_NAME
          value: zennify
        - name: DB_USER
          valueFrom:
            secretKeyRef:
              name: postgres
              key: username
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: postgres
              key: password
        volumeMounts:
        - name: changelog
          mountPath: /liquibase/changelog
      volumes:
      - name: changelog
        configMap:
          name: liquibase-changelog
      restartPolicy: Never
```

## References

- [Liquibase Documentation](https://docs.liquibase.com)
- [Liquibase Change Types](https://docs.liquibase.com/change-types/home.html)
- [PostgreSQL Liquibase Integration](https://docs.liquibase.com/workflows/database-setup-tutorials/postgresql.html)

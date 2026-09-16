#!/bin/bash
set -e

# Liquibase migration helper script

DB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Default values
HOST="${DB_HOST:-localhost}"
PORT="${DB_PORT:-65025}"
DATABASE="${DB_NAME:-zenn_ai}"
USER="${DB_USER:-postgres}"
PASSWORD="${DB_PASSWORD:-changeme}"
LIQUIBASE_CMD="${1:-update}"

# Available commands
case "$LIQUIBASE_CMD" in
  update)
    echo "Applying pending database migrations..."
    liquibase \
      --driver=org.postgresql.Driver \
      --url="jdbc:postgresql://${HOST}:${PORT}/${DATABASE}" \
      --username="${USER}" \
      --password="${PASSWORD}" \
      --changelogFile="./liquibase.yaml" \
      update
    echo "Database migration completed successfully"
    ;;

  status)
    echo "Checking database migration status..."
    liquibase \
      --driver=org.postgresql.Driver \
      --url="jdbc:postgresql://${HOST}:${PORT}/${DATABASE}" \
      --username="${USER}" \
      --password="${PASSWORD}" \
      --changelogFile="./liquibase.yaml" \
      status
    ;;

  history)
    echo "Displaying migration history..."
    liquibase \
      --driver=org.postgresql.Driver \
      --url="jdbc:postgresql://${HOST}:${PORT}/${DATABASE}" \
      --username="${USER}" \
      --password="${PASSWORD}" \
      --changelogFile="./liquibase.yaml" \
      history
    ;;

  validate)
    echo "Validating changelog..."
    liquibase \
      --driver=org.postgresql.Driver \
      --url="jdbc:postgresql://${HOST}:${PORT}/${DATABASE}" \
      --username="${USER}" \
      --password="${PASSWORD}" \
      --changelogFile="./liquibase.yaml" \
      validate
    ;;

  update-sql)
    echo "Previewing pending SQL changes..."
    liquibase \
      --driver=org.postgresql.Driver \
      --url="jdbc:postgresql://${HOST}:${PORT}/${DATABASE}" \
      --username="${USER}" \
      --password="${PASSWORD}" \
      --changelogFile="./liquibase.yaml" \
      update-sql
    ;;

  rollback)
    COUNT="${2:-1}"
    echo "Rolling back last ${COUNT} changeset(s)..."
    liquibase \
      --driver=org.postgresql.Driver \
      --url="jdbc:postgresql://${HOST}:${PORT}/${DATABASE}" \
      --username="${USER}" \
      --password="${PASSWORD}" \
      --changelogFile="./liquibase.yaml" \
      rollback --count="${COUNT}"
    ;;

  *)
    echo "Usage: liquibase.sh [command] [options]"
    echo ""
    echo "Commands:"
    echo "  update         Apply all pending migrations (default)"
    echo "  status         Show migration status"
    echo "  history        Show applied migrations"
    echo "  validate       Validate changelog"
    echo "  update-sql     Preview pending SQL changes"
    echo "  rollback [N]   Rollback last N changesets (default 1)"
    echo ""
    echo "Environment variables:"
    echo "  DB_HOST        Database host (default: localhost)"
    echo "  DB_PORT        Database port (default: 5432)"
    echo "  DB_NAME        Database name (default: zennify)"
    echo "  DB_USER        Database user (default: postgres)"
    echo "  DB_PASSWORD    Database password (default: changeme)"
    exit 1
    ;;
esac

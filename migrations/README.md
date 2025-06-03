# Database Migrations

This directory contains database migrations for the iddaa-core project.

## Setup

Install the `migrate` tool:

```bash
# macOS
brew install golang-migrate

# Linux
curl -L https://github.com/golang-migrate/migrate/releases/download/v4.16.2/migrate.linux-amd64.tar.gz | tar xvz
sudo mv migrate /usr/local/bin/
```

## Usage

### Create a new migration
```bash
migrate create -ext sql -dir migrations -seq migration_name
```

### Run migrations
```bash
export DATABASE_URL="postgres://user:password@localhost:5432/iddaa_core?sslmode=disable"
migrate -path migrations -database $DATABASE_URL up
```

### Rollback migrations
```bash
migrate -path migrations -database $DATABASE_URL down 1
```

### Check migration status
```bash
migrate -path migrations -database $DATABASE_URL version
```

## Migration Files

- `000001_initial.up.sql` - Initial schema creation
- `000001_initial.down.sql` - Initial schema rollback

## Notes

- Migrations are run automatically by the application on startup in development
- In production, run migrations manually before deploying new versions
- Always test migrations on a copy of production data first
- Keep migrations small and focused on single changes